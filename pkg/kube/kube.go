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
const pemResourceName = "certificator-pem"
const jksResourceName = "certificator-jks"

// Commit certificate bundle into all team namespaces
const namespaceSelector = "team"

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

func applyWithRetry(ctx context.Context, client *kubernetes.Clientset, namespace string, resource *v1.ConfigMap) error {
	nsclient := client.CoreV1().ConfigMaps(namespace)

	for ctx.Err() == nil {
		err := createOrUpdate(ctx, nsclient, resource)
		if err == nil {
			log.Debugf("Applied %q to namespace %q", resource.Name, namespace)
			return nil
		}
		log.Errorf("Failed to apply %q to namespace %q: %s", resource.Name, namespace, err)
		time.Sleep(3 * time.Second)
	}

	return ctx.Err()
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

	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: namespaceSelector,
	})

	if err != nil {
		return err
	}

	log.Debugf("Applying certificate bundles to %d team namespaces", len(namespaces.Items))

	wg := &sync.WaitGroup{}
	errs := make(chan error, len(namespaces.Items)*2+1)

	for _, namespace := range namespaces.Items {
		wg.Add(2)
		go func(ns string) {
			errs <- applyWithRetry(ctx, client, ns, pem)
			wg.Done()
		}(namespace.Name)
		go func(ns string) {
			errs <- applyWithRetry(ctx, client, ns, jks)
			wg.Done()
		}(namespace.Name)
	}

	log.Debugf("Waiting for goroutines to finish applying...")

	wg.Wait()
	close(errs)

	errorCount := 0
	for err = range errs {
		if err == nil {
			continue
		}
		errorCount++
	}

	if errorCount > 0 {
		return fmt.Errorf("applying certificate bundles resulted in %d errors", errorCount)
	}

	return nil
}
