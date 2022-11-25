package kube

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Namespaces map[string]*Namespace

type Namespace struct {
	Name        string
	LastSeen    time.Time
	LastSuccess time.Time
	LastFailure time.Time
	Deleted     bool
}

func (namespaces Namespaces) UnsuccessfulSince(t time.Time) Namespaces {
	result := make(Namespaces)
	for k, v := range namespaces {
		if v.LastSuccess.Before(t) {
			result[k] = v
		}
	}
	return result
}

func Watch(ctx context.Context, client *kubernetes.Clientset, labelSelector string, namespaces chan<- *Namespace) error {
	defer close(namespaces)

	watcher, err := client.CoreV1().Namespaces().Watch(ctx, metav1.ListOptions{
		LabelSelector:   labelSelector,
		ResourceVersion: "0",
	})

	if err != nil {
		return err
	}

	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		namespace, ok := event.Object.(*v1.Namespace)
		if !ok {
			log.Debugf("watch: skip %T %v", event.Object, event.Object)
			continue
		}
		namespaces <- &Namespace{
			Name:     namespace.Name,
			LastSeen: time.Now(),
			Deleted:  namespace.DeletionTimestamp != nil,
		}
	}

	return nil
}
