package certbundle_test

import (
	"bytes"
	"encoding/hex"
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

func TestEqual(t *testing.T) {
	const expectedHash = "d0a2624c7600d1a72e9b4a3c7c2d8d8b2202283789de3cd98d64d80f9cc0ba68"

	b1 := bundleFromTestData()
	b2 := bundleFromTestData()

	assert.Equal(t, expectedHash, hex.EncodeToString(b1.Hash()))
	assert.Equal(t, expectedHash, hex.EncodeToString(b2.Hash()))

	assert.True(t, b1.Equal(b2))
	assert.True(t, b2.Equal(b1))

	f, err := os.Open("../../testdata/cacert.pem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = b1.ReadAll(f)
	if err != nil {
		panic(err)
	}

	assert.False(t, b1.Equal(b2))
	assert.False(t, b2.Equal(b1))
}
