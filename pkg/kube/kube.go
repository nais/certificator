package kube

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubernetes CM data keys, i.e. filename in pod
const pemFilename = "ca-bundle.pem"
const jksFilename = "ca-bundle.jks"

// Kubernetes CM names
const pemResourceName = "ca-bundle-pem"
const jksResourceName = "ca-bundle-jks"

// Backoff time per apply
const retryBackoff = time.Second * 3

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

func Apply(ctx context.Context, client *kubernetes.Clientset, bundle BundleWriter, namespaces Namespaces) error {
	jks, err := ConfigMapJKS(bundle)
	if err != nil {
		return err
	}

	pem, err := ConfigMapPEM(bundle)
	if err != nil {
		return err
	}

	log.Debugf("Applying certificate bundles to %d team namespaces", len(namespaces))

	wg := &sync.WaitGroup{}
	errs := make(chan error, len(namespaces)*2+1)

	apply := func(ns *Namespace, cm *v1.ConfigMap) {
		nsclient := client.CoreV1().ConfigMaps(ns.Name)
		er := createOrUpdate(ctx, nsclient, cm)
		if er == nil {
			log.Debugf("Applied %q to namespace %q", cm.Name, ns.Name)
			ns.LastSuccess = time.Now()
		} else {
			log.Errorf("Failed to apply %q to namespace %q: %s", cm.Name, ns.Name, er)
			ns.LastFailure = time.Now()
			errs <- er
		}
		wg.Done()
	}

	for _, namespace := range namespaces {
		wg.Add(2)
		go apply(namespace, pem)
		go apply(namespace, jks)
	}

	log.Debugf("Waiting for goroutines to finish applying...")

	wg.Wait()
	close(errs)

	errorCount := len(errs)
	for err = range errs {
		log.Errorf("Apply to Kubernetes: %s", err)
	}

	if errorCount > 0 {
		return fmt.Errorf("applying certificate bundles resulted in %d errors", errorCount)
	}

	return nil
}
