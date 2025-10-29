package kube

import (
	"fmt"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewInformerFactory creates a shared informer factory that works both
// in-cluster and out of cluster.
func NewInformerFactory() (informers.SharedInformerFactory, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("build Kubernetes config: %w", err)
		}
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("construct Kubernetes client: %w", err)
	}

	return informers.NewSharedInformerFactory(clientSet, 0), nil
}
