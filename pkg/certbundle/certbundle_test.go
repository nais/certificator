package certbundle_test

import (
	"bytes"
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

func TestWrite(t *testing.T) {
	const password = "foobar"

	bundle := bundleFromTestData()
	for _, cert := range bundle.Certificates() {
		t.Logf("CA=%v %v", cert.IsCA, cert.Subject)
	}

	jksout := &bytes.Buffer{}
	pemout := &bytes.Buffer{}

	err := bundle.WriteJKS(jksout)
	assert.NoError(t, err)

	err = bundle.WritePEM(pemout)
	assert.NoError(t, err)

	t.Logf("JKS bundle with password %q encoded into memory", password)
	t.Logf("PEM bundle encoded into memory")
}
