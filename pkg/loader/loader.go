package loader

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/nais/certificator/pkg/certbundle"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Well-known default password for JKS format.
// Bundle does not contain any private data, so this is fine.
const password = "changeme"

// Download some content and copy it into an io.Reader buffer.
func download(ctx context.Context, url string) (io.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Create a certificate bundle from the content of a list of URLs.
func BundleFromURLs(ctx context.Context, bundle *certbundle.Bundle, urls []string) error {
	errors := make(chan error, len(urls)+1)
	readers := make(chan io.Reader, len(urls)+1)

	wg := &sync.WaitGroup{}
	wg.Add(len(urls))
	for _, url := range urls {
		go func(u string) {
			defer wg.Done()
			log.Infof("Downloading certificates from %s", u)
			r, err := download(ctx, u)
			if err != nil {
				errors <- err
			} else {
				readers <- r
			}
		}(url)
	}
	wg.Wait()

	close(errors)
	close(readers)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	for r := range readers {
		err := bundle.ReadAll(r)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create a certificate bundle from the content of file system directories.
func BundleFromPaths(paths []string, bundle *certbundle.Bundle) error {
	var err error

	for _, directory := range paths {
		directory, err = filepath.Abs(directory)
		if err != nil {
			return err
		}
		log.Infof("Scanning directory %s", directory)
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			log.Infof("Load %s", path)
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			err = bundle.ReadAll(f)
			f.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
