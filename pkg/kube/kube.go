package kube

import (
	"bytes"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nais/certificator/pkg/metrics"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Kubernetes CM data keys, i.e. filename in pod
const pemFilename = "ca-bundle.pem"
const jksFilename = "ca-bundle.jks"

// Kubernetes CM names
const pemResourceName = "ca-bundle-pem"
const jksResourceName = "ca-bundle-jks"

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
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "certificator",
			},
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

func GenerateApplyOperations(ctx context.Context, client *kubernetes.Clientset, bundle BundleWriter, namespaces Namespaces, applies chan func() error) error {
	jks, err := ConfigMapJKS(bundle)
	if err != nil {
		return err
	}

	pem, err := ConfigMapPEM(bundle)
	if err != nil {
		return err
	}

	apply := func(ns *Namespace, cmaps ...*v1.ConfigMap) error {
		nsclient := client.CoreV1().ConfigMaps(ns.Name)
		for _, cm := range cmaps {
			er := createOrUpdate(ctx, nsclient, cm)
			if er == nil {
				log.Debugf("Applied %q to namespace %q", cm.Name, ns.Name)
				metrics.IncSync(0)
			} else {
				ns.LastFailure = time.Now()
				metrics.IncSync(1)
				return fmt.Errorf("apply %q to namespace %q: %s", cm.Name, ns.Name, er)
			}
		}
		ns.LastSuccess = time.Now()
		return nil
	}

	go func() {
		log.Debugf("Generating %d team namespace Kubernetes operations", len(namespaces))
		for _, namespace := range namespaces {
			ns := namespace
			applies <- func() error {
				return apply(ns, pem, jks)
			}
		}
	}()

	return nil
}
