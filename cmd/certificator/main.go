package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/config"
	"github.com/nais/certificator/pkg/kube"
	"github.com/nais/certificator/pkg/loader"
	"github.com/nais/certificator/pkg/metrics"
	"github.com/nais/certificator/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	var bundle, updatedBundle *certbundle.Bundle
	var namespaceWatcher chan *kube.Namespace
	var namespaces = make(kube.Namespaces)
	var applies chan func() error
	var applyContext context.Context
	var applyCancel context.CancelFunc

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

	go func() {
		log.Infof("Starting metrics server at %s", cfg.MetricsAddress)
		err := http.ListenAndServe(cfg.MetricsAddress, promhttp.Handler())
		if err != nil {
			log.Errorf("Metrics server shut down: %s", err)
			cancel()
		}
	}()

	downloadTimer := time.NewTimer(1 * time.Millisecond)
	bundleTimer := time.NewTimer(time.Hour)
	bundleTimer.Stop()

	setupNamespaceWatch := func() {
		namespaceWatcher = make(chan *kube.Namespace, 1024)
		go func() {
			log.Infof("Starting Kubernetes namespace watcher.")
			err := kube.Watch(ctx, clientset, cfg.NamespaceLabelSelector, namespaceWatcher)
			if err != nil {
				log.Errorf("Init Kubernetes namespace watcher: %s", err)
			} else {
				log.Errorf("Kubernetes namespace watcher stopped.")
			}
		}()
	}

	setupNamespaceWatch()
	applyContext, applyCancel = context.WithTimeout(ctx, cfg.ApplyTimeout)
	applies = make(chan func() error, 1024)

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
				metrics.SetTotalNamespaces(len(namespaces))
				continue
			}
			namespace, ok := namespaces[watchedNamespace.Name]
			if ok {
				namespace.LastSeen = watchedNamespace.LastSeen
				continue
			}
			log.Debugf("Namespace %q added to update candidates.", watchedNamespace.Name)
			namespaces[watchedNamespace.Name] = watchedNamespace
			metrics.SetTotalNamespaces(len(namespaces))
			bundleTimer.Reset(time.Millisecond)

		case <-bundleTimer.C:
			// Run the configmap synchronization for all namespaces that haven't been updated
			// since the last bundle update.
			if bundle == nil {
				continue
			}
			candidates := namespaces.UnsuccessfulSince(bundle.ChangedAt())
			metrics.SetPendingNamespaces(len(candidates))
			if len(candidates) == 0 {
				log.Debugf("No namespaces in need of new CA certificate bundle")
				bundleTimer.Stop()
				continue
			}
			applyContext, applyCancel = context.WithTimeout(ctx, cfg.ApplyTimeout)
			log.Infof("Generating %d CA certificate bundle ConfigMap operations, timeout %s", len(candidates), cfg.ApplyTimeout)
			err = kube.GenerateApplyOperations(applyContext, clientset, bundle, candidates, applies)
			if err != nil {
				log.Errorf("Failed to generate CA certificate bundles: %s", err)
				applyCancel()
			}

		case apply := <-applies:
			err = apply()
			pending := len(namespaces.UnsuccessfulSince(bundle.ChangedAt()))
			metrics.SetPendingNamespaces(pending)
			if err != nil {
				log.Error(err)
			}
			if len(applies) > 0 {
				continue
			}
			applyCancel()
			if pending == 0 {
				log.Warnf("Certificate bundle applied to Kubernetes namespaces successfully")
				bundleTimer.Stop()
				continue
			}
			log.Errorf("Still have %d pending namespaces to apply certificate bundle into", pending)
			log.Debugf("Waiting %s before next attempt", cfg.ApplyBackoff)
			bundleTimer.Reset(cfg.ApplyBackoff)

		case <-downloadTimer.C:
			// Refresh the certificate bundle.
			updatedBundle, err = update(ctx, cfg)
			if err == nil {
				metrics.IncRefresh(0)
				log.Warnf("Refreshed certificate list from external sources with %d entries", updatedBundle.Len())
				downloadTimer.Reset(cfg.DownloadInterval)
				log.Debugf("Next refresh in %s", cfg.DownloadInterval)
				if bundle != nil && bundle.Equal(updatedBundle) {
					log.Warnf("Certificate bundle is exactly the same as last time, no cluster updates necessary.")
					continue
				}
				bundle = updatedBundle
				metrics.SetCertificates(bundle.Len())
				bundleTimer.Reset(time.Millisecond)
			} else {
				metrics.IncRefresh(1)
				log.Errorf("Refresh certificate list: %s", err)
				downloadTimer.Reset(cfg.DownloadRetryInterval)
				log.Debugf("Next attempt at refresh in %s", cfg.DownloadRetryInterval)
			}
		}
	}

	return nil
}
