package kube_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/nais/certificator/pkg/certbundle"
	"github.com/nais/certificator/pkg/kube"
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

func TestConfigMapPEM(t *testing.T) {
	bundle := bundleFromTestData()

	cm, err := kube.ConfigMapPEM(bundle)
	if err != nil {
		panic(err)
	}

	out, err := os.CreateTemp(os.TempDir(), "configmap-pem-")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.Encode(cm)

	t.Logf("ConfigMap written to %s", out.Name())
}

func TestConfigMapJKS(t *testing.T) {
	bundle := bundleFromTestData()

	cm, err := kube.ConfigMapJKS(bundle)
	if err != nil {
		panic(err)
	}

	out, err := os.CreateTemp(os.TempDir(), "configmap-jks-")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.Encode(cm)

	t.Logf("ConfigMap written to %s", out.Name())
}
