package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/config"
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
	log.Infof("Clean exit.")
}

func update(ctx context.Context, cfg *config.Config) (*certbundle.Bundle, error) {
	bundle := certbundle.New(cfg.JksPassword)
	err := loader.BundleFromPaths(cfg.CAPaths, bundle)
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

	cfg, err := config.NewFromEnv()
	if err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	log.Infof("Starting certificator")
	log.Infof("Configured %d CA certificate sources", len(cfg.CAUrls)+len(cfg.CAPaths))
	for _, src := range cfg.CAPaths {
		log.Infof("File system source: %v", src)
	}
	for _, src := range cfg.CAUrls {
		log.Infof("Remote URL source: %v", src)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	downloadTimer := time.NewTimer(1 * time.Millisecond)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-downloadTimer.C:
			bundle, err = update(ctx, cfg)
			if err == nil {
				downloadTimer.Reset(cfg.DownloadInterval)
				log.Warnf("Refreshed certificate list from external sources with %d entries", bundle.Len())
			} else {
				downloadTimer.Reset(cfg.DownloadRetryInterval)
				log.Errorf("Refresh certificate list: %s", err)
				log.Debugf("Next attempt at refresh at %s", time.Now().Add(cfg.DownloadRetryInterval))
			}
		}
	}

	return ctx.Err()
}
