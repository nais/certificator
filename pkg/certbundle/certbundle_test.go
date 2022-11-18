package certbundle_test

import (
	"os"
	"testing"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/stretchr/testify/assert"
)

const password = "foobar"

func bundleFromTestData() *certbundle.Bundle {
	f, err := os.Open("../../testdata/cacert.pem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	bundle := certbundle.New(password)

	err = bundle.ReadAll(f)

	if err != nil {
		panic(err)
	}

	return bundle
}

func TestReadAllPEM(t *testing.T) {
	bundle := bundleFromTestData()

	for _, cert := range bundle.Certificates() {
		t.Logf("CA=%v %v", cert.IsCA, cert.Subject)
	}
}

func TestWrite(t *testing.T) {
	const password = "foobar"

	bundle := bundleFromTestData()

	jksout, err := os.CreateTemp(os.TempDir(), "jks-")
	if err != nil {
		panic(err)
	}
	defer jksout.Close()

	pemout, err := os.CreateTemp(os.TempDir(), "pem-")
	if err != nil {
		panic(err)
	}
	defer pemout.Close()

	err = bundle.WriteJKS(jksout)
	assert.NoError(t, err)

	err = bundle.WritePEM(pemout)
	assert.NoError(t, err)

	t.Logf("JKS certificates with password %q encoded into %s", password, jksout.Name())
	t.Logf("PEM certificates encoded into %s", jksout.Name())
}
