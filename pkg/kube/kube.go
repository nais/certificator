package kube

import (
	"bytes"
	"io"
	"time"

	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const pemFilename = "ca-bundle.pem"
const jksFilename = "ca-bundle.jks"

const pemResourceName = "certificator-pem"
const jksResourceName = "certificator-jks"

type PEMWriter interface {
	WritePEM(w io.Writer) error
}

type JKSWriter interface {
	WriteJKS(w io.Writer) error
}

type BundleWriter interface {
	JKSWriter
	PEMWriter
}

func configMap(filename, resourceName string, writer func(io.Writer) error) (*v1.ConfigMap, error) {
	raw := &bytes.Buffer{}
	err := writer(raw)
	if err != nil {
		return nil, err
	}
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
			Annotations: map[string]string{
				"certificator.nais.io/last-applied-at": time.Now().Format(time.RFC3339),
			},
		},
		BinaryData: map[string][]byte{
			filename: raw.Bytes(),
		},
	}, nil
}

func ConfigMapPEM(bundle PEMWriter) (*v1.ConfigMap, error) {
	return configMap(pemFilename, pemResourceName, bundle.WritePEM)
}

func ConfigMapJKS(bundle JKSWriter) (*v1.ConfigMap, error) {
	return configMap(jksFilename, jksResourceName, bundle.WriteJKS)
}

func Client() (*kubernetes.Clientset, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil)
	rest, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(rest)
}

func createOrUpdate(ctx context.Context, client corev1.ConfigMapInterface, resource *v1.ConfigMap) error {
	_, err := client.Create(ctx, resource, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = client.Update(ctx, resource, metav1.UpdateOptions{})
	}
	return err
}

func Apply(ctx context.Context, client *kubernetes.Clientset, bundle BundleWriter) error {
	jks, err := ConfigMapJKS(bundle)
	if err != nil {
		return err
	}

	pem, err := ConfigMapPEM(bundle)
	if err != nil {
		return err
	}

	namespace := client.CoreV1().ConfigMaps("default")

	err = createOrUpdate(ctx, namespace, jks)
	if err != nil {
		return err
	}

	err = createOrUpdate(ctx, namespace, pem)
	if err != nil {
		return err
	}

	return nil
}
