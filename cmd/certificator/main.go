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
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		return config.Usage()
	}

	cfg, err := config.NewFromEnv()
	if err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	log.Infof("Starting certificator")
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	downloadTimer := time.NewTimer(1 * time.Millisecond)
	bundleTimer := time.NewTimer(time.Hour)
	bundleTimer.Stop()

	var bundle *certbundle.Bundle

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return nil
		case <-bundleTimer.C:
			ac, acc := context.WithTimeout(ctx, cfg.ApplyTimeout)
			log.Infof("Applying CA certificate bundle to Kubernetes")
			err = kube.Apply(ac, clientset, bundle)
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
