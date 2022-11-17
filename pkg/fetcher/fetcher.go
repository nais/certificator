package fetcher

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/nais/certificator/pkg/certbundle"
	"golang.org/x/net/context"
)

// Well-known default password for JKS format.
// Bundle does not contain any private data, so this is fine.
const password = "changeme"

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

func MakeCertificateBundle(ctx context.Context, urls []string) (*certbundle.Bundle, error) {
	errors := make(chan error, len(urls)+1)
	readers := make(chan io.Reader, len(urls)+1)

	wg := &sync.WaitGroup{}
	wg.Add(len(urls))
	for _, url := range urls {
		go func(u string) {
			defer wg.Done()
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
			return nil, err
		}
	}

	bundle := certbundle.New(password)

	for r := range readers {
		err := bundle.ReadAllPEM(r)
		if err != nil {
			return nil, err
		}
	}

	return bundle, nil
}
