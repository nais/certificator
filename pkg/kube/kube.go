package kube

import (
	"bytes"
	"io"

	v1 "k8s.io/api/core/v1"
)

const pemFilename = "ca-bundle.pem"
const jksFilename = "ca-bundle.jks"

type PEMWriter interface {
	WritePEM(w io.Writer) error
}

type JKSWriter interface {
	WriteJKS(w io.Writer) error
}

func configMap(filename string, writer func(io.Writer) error) (*v1.ConfigMap, error) {
	raw := &bytes.Buffer{}
	err := writer(raw)
	if err != nil {
		return nil, err
	}
	return &v1.ConfigMap{
		BinaryData: map[string][]byte{
			filename: raw.Bytes(),
		},
	}, nil
}

func ConfigMapPEM(bundle PEMWriter) (*v1.ConfigMap, error) {
	return configMap(pemFilename, bundle.WritePEM)
}

func ConfigMapJKS(bundle JKSWriter) (*v1.ConfigMap, error) {
	return configMap(jksFilename, bundle.WriteJKS)
}
