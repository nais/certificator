package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/config"
	"github.com/nais/certificator/pkg/kube"
	"github.com/nais/certificator/pkg/loader"
	"github.com/nais/certificator/pkg/version"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s", err)
		os.Exit(1)
	}
}

func update(ctx context.Context, cfg *config.Config) (*certbundle.Bundle, error) {
	bundle := certbundle.New(cfg.JksPassword)
	err := loader.BundleFromPaths(cfg.CADirectories, bundle)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.DownloadTimeout)
	defer cancel()
	err = loader.BundleFromURLs(ctx, bundle, cfg.CAUrls)
	if err != nil {
		return nil, err
	}
	return bundle, err

}

func run() error {
	var bundle *certbundle.Bundle
	var namespaceWatcher chan *kube.Namespace
	var namespaces = make(kube.Namespaces)

	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		return config.Usage()
	}

	cfg, err := config.NewFromEnv()
	if err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	log.SetFormatter(cfg.LogFormat.Formatter)
	log.SetLevel(log.Level(cfg.LogLevel))

	buildTime, _ := version.BuildTime()
	log.Infof("Certificator %s built on %s", version.Version(), buildTime)
	log.Infof("Configured %d CA certificate sources", len(cfg.CAUrls)+len(cfg.CADirectories))

	for _, src := range cfg.CADirectories {
		log.Infof("File system source: %v", src)
	}
	for _, src := range cfg.CAUrls {
		log.Infof("Remote URL source: %v", src)
	}

	clientset, err := kube.Client()
	if err != nil {
		return fmt.Errorf("init kubernetes client: %w", err)
	}

	log.Infof("Configuration complete, starting application.")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	downloadTimer := time.NewTimer(1 * time.Millisecond)
	bundleTimer := time.NewTimer(time.Hour)
	bundleTimer.Stop()

	setupNamespaceWatch := func() {
		namespaceWatcher = make(chan *kube.Namespace, 1024)
		go func() {
			log.Infof("Starting Kubernetes namespace watcher.")
			err := kube.Watch(ctx, clientset, namespaceWatcher)
			if err != nil {
				log.Errorf("Init Kubernetes namespace watcher: %s", err)
			} else {
				log.Errorf("Kubernetes namespace watcher stopped.")
			}
			close(namespaceWatcher)
		}()
	}

	setupNamespaceWatch()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			log.Infof("Received signal, shutting down.")
			return nil

		case watchedNamespace, ok := <-namespaceWatcher:
			// Each time a namespace is returned from the watcher, add it to the list of candidates,
			// and trigger a synchronization.
			if !ok {
				setupNamespaceWatch()
				continue
			}
			if watchedNamespace.Deleted {
				log.Warnf("Namespace %q deleted; removed from update candidates.", watchedNamespace.Name)
				delete(namespaces, watchedNamespace.Name)
				continue
			}
			namespace, ok := namespaces[watchedNamespace.Name]
			if ok {
				namespace.LastSeen = watchedNamespace.LastSeen
				continue
			}
			log.Debugf("Namespace %q added to update candidates.", watchedNamespace.Name)
			namespaces[watchedNamespace.Name] = watchedNamespace
			bundleTimer.Reset(time.Millisecond)

		case <-bundleTimer.C:
			// Run the configmap synchronization for all namespaces that haven't been updated
			// since the last bundle update.
			if bundle == nil {
				continue
			}
			candidates := namespaces.UnsuccessfulSince(bundle.ChangedAt())
			if len(candidates) == 0 {
				log.Debugf("No namespaces in need of new CA certificate bundle")
				bundleTimer.Stop()
				continue
			}
			ac, acc := context.WithTimeout(ctx, cfg.ApplyTimeout)
			log.Infof("Applying CA certificate bundle to %d Kubernetes namespaces, timeout %s", len(candidates), cfg.ApplyTimeout)
			err = kube.Apply(ac, clientset, bundle, candidates)
			acc()
			if err == nil {
				log.Warnf("Certificate bundle applied to Kubernetes namespaces successfully")
				bundleTimer.Stop()
				continue
			}
			log.Errorf("Apply certificate bundle to Kubernetes: %s", err)
			log.Debugf("Sleeping %s before next attempt", cfg.ApplyBackoff)
			bundleTimer.Reset(cfg.ApplyBackoff)

		case <-downloadTimer.C:
			// Refresh the certificate bundle.
			bundle, err = update(ctx, cfg)
			if err == nil {
				log.Warnf("Refreshed certificate list from external sources with %d entries", bundle.Len())
				downloadTimer.Reset(cfg.DownloadInterval)
				log.Debugf("Next refresh in %s", cfg.DownloadInterval)
				bundleTimer.Reset(time.Millisecond)
			} else {
				log.Errorf("Refresh certificate list: %s", err)
				downloadTimer.Reset(cfg.DownloadRetryInterval)
				log.Debugf("Next attempt at refresh in %s", cfg.DownloadRetryInterval)
			}
		}
	}

	return nil
}
