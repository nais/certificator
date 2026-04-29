package kube_test

import (
	"os"
	"testing"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/kube"
	"github.com/stretchr/testify/assert"
)

const password = "foobar"

func bundleFromTestData() *certbundle.Bundle {
	f, err := os.Open("../../testdata/cacert.pem")
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	bundle := certbundle.New(password)

	err = bundle.ReadAll(f)
	if err != nil {
		panic(err)
	}

	return bundle
}

func TestConfigMapPEM(t *testing.T) {
	bundle := bundleFromTestData()

	cm, err := kube.ConfigMapPEM(bundle)
	if err != nil {
		panic(err)
	}

	assert.NotEmpty(t, cm.BinaryData)
}

func TestConfigMapJKS(t *testing.T) {
	bundle := bundleFromTestData()

	cm, err := kube.ConfigMapJKS(bundle)
	if err != nil {
		panic(err)
	}

	assert.NotEmpty(t, cm.BinaryData)
}
