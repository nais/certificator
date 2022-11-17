package fetcher_test

import (
	"testing"
	"time"

	"github.com/nais/certificator/pkg/fetcher"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestMakeCertificateBundle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bundle, err := fetcher.MakeCertificateBundle(ctx, []string{
		"https://curl.se/ca/cacert.pem",
		"http://crl.adeo.no/crl/eksterne/webproxy.nav.no.crt",
	})

	assert.NoError(t, err)
	assert.True(t, len(bundle.Certificates()) > 0)

	t.Logf("Bundle contains %d certificates", len(bundle.Certificates()))
}
