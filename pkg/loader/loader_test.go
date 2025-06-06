package loader_test

import (
	"testing"
	"time"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/loader"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

const password = "foobar"

func TestMakeCertificateBundle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bundle := certbundle.New(password)
	err := loader.BundleFromURLs(ctx, bundle, []string{
		"https://curl.se/ca/cacert.pem",
	})

	assert.NoError(t, err)
	assert.True(t, len(bundle.Certificates()) > 0)

	t.Logf("Bundle contains %d certificates", len(bundle.Certificates()))
}

func TestBundleFromPaths(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	bundle := certbundle.New(password)
	err := loader.BundleFromPaths([]string{"../../testdata"}, bundle)
	assert.NoError(t, err)
	assert.True(t, len(bundle.Certificates()) > 0)

	t.Logf("Bundle contains %d certificates", len(bundle.Certificates()))
}
