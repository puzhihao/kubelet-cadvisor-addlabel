package metrics

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var errCacheSyncFailed = errors.New("failed to sync informer caches")

// Service manages the informer lifecycle and exposes cached cluster metadata
// needed by the metrics collector.
type Service struct {
	factory      informers.SharedInformerFactory
	cache        *Cache
	nodeInformer cache.SharedIndexInformer
	podInformer  cache.SharedIndexInformer
	synced       chan struct{}
	syncOnce     sync.Once
}

// NewService wires the informers and cache used to look up labels and node IPs.
func NewService(factory informers.SharedInformerFactory) *Service {
	return &Service{
		factory:      factory,
		cache:        NewCache(),
		nodeInformer: factory.Core().V1().Nodes().Informer(),
		podInformer:  factory.Core().V1().Pods().Informer(),
		synced:       make(chan struct{}),
	}
}

// Run starts the informers and blocks until the context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	s.registerHandlers()

	klog.InfoS("starting shared informers")
	s.factory.Start(ctx.Done())

	if ok := cache.WaitForCacheSync(ctx.Done(), s.nodeInformer.HasSynced, s.podInformer.HasSynced); !ok {
		return errCacheSyncFailed
	}

	s.syncOnce.Do(func() {
		close(s.synced)
		klog.InfoS("informer caches synced")
	})

	<-ctx.Done()
	return ctx.Err()
}

// WaitForSync blocks until the informers report a synced cache or the context is cancelled.
func (s *Service) WaitForSync(ctx context.Context) error {
	select {
	case <-s.synced:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NodeIPs returns the cached node IP addresses.
func (s *Service) NodeIPs() []string {
	return s.cache.NodeIPs()
}

// PodLabels resolves pod labels with a cache-first lookup and informer fallback.
func (s *Service) PodLabels(namespace, podName string) map[string]string {
	if labels, ok := s.cache.PodLabels(namespace, podName); ok {
		return labels
	}

	pod, err := s.factory.Core().V1().Pods().Lister().Pods(namespace).Get(podName)
	if err != nil {
		klog.V(4).InfoS("pod labels unavailable from lister", "pod", cacheKey(namespace, podName), "err", err)
		return nil
	}

	s.cache.StorePodLabels(namespace, podName, pod.Labels)
	return cloneStringMap(pod.Labels)
}

func (s *Service) registerHandlers() {
	s.nodeInformer.AddEventHandler(newNodeEventHandler(s.cache))
	s.podInformer.AddEventHandler(newPodEventHandler(s.cache))
}

// DebugString returns a snapshot of key cache statistics for logging.
func (s *Service) DebugString() string {
	return fmt.Sprintf("nodeIPs=%d", len(s.NodeIPs()))
}
