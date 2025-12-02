package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"k8s.io/klog/v2"
)

// Cache keeps a lightweight in-memory view of node IPs and pod labels.
// It is populated by informer event handlers and queried by the collector.
type Cache struct {
	nodeIPs   sync.Map
	podLabels sync.Map
}

// NewCache returns an initialized Cache instance.
func NewCache() *Cache {
	return &Cache{}
}

// PodLabels returns a defensive copy of the cached pod labels.
// The second return value reports whether the labels were present.
func (c *Cache) PodLabels(namespace, podName string) (map[string]string, bool) {
	key := cacheKey(namespace, podName)
	if labels, ok := c.podLabels.Load(key); ok {
		klog.V(5).InfoS("pod labels cache hit", "pod", key)
		return cloneStringMap(labels.(map[string]string)), true
	}

	klog.V(6).InfoS("pod labels cache miss", "pod", key)
	return nil, false
}

// NodeIPs returns the known node IP addresses.
func (c *Cache) NodeIPs() []string {
	var ips []string
	c.nodeIPs.Range(func(_, value interface{}) bool {
		ip := value.(string)
		if ip != "" {
			ips = append(ips, ip)
		}
		return true
	})

	return ips
}

// UniqueLabelValues returns all unique, non-empty values observed for a specific label key.
func (c *Cache) UniqueLabelValues(label string) []string {
	label = strings.TrimSpace(label)
	if label == "" {
		return nil
	}

	values := make(map[string]struct{})
	c.podLabels.Range(func(_, value interface{}) bool {
		labels := value.(map[string]string)
		if v := strings.TrimSpace(labels[label]); v != "" {
			values[v] = struct{}{}
		}
		return true
	})

	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for v := range values {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}

// StorePodLabels stores a defensive copy of the provided labels.
func (c *Cache) StorePodLabels(namespace, podName string, labels map[string]string) {
	key := cacheKey(namespace, podName)
	if len(labels) == 0 {
		c.podLabels.Delete(key)
		klog.V(5).InfoS("removed pod labels from cache", "pod", key)
		return
	}

	c.podLabels.Store(key, cloneStringMap(labels))
	klog.V(6).InfoS("cached pod labels", "pod", key, "count", len(labels))
}

// DeletePodLabels removes cached pod labels.
func (c *Cache) DeletePodLabels(namespace, podName string) {
	key := cacheKey(namespace, podName)
	c.podLabels.Delete(key)
	klog.V(6).InfoS("deleted pod labels cache entry", "pod", key)
}

// StoreNodeIP registers a node IP; empty IPs are ignored and remove the entry.
func (c *Cache) StoreNodeIP(nodeName, ip string) {
	if ip == "" {
		c.nodeIPs.Delete(nodeName)
		klog.V(5).InfoS("removed node IP due to empty address", "node", nodeName)
		return
	}

	c.nodeIPs.Store(nodeName, ip)
	klog.V(6).InfoS("cached node IP", "node", nodeName, "ip", ip)
}

// DeleteNode removes a node entry from the cache.
func (c *Cache) DeleteNode(nodeName string) {
	c.nodeIPs.Delete(nodeName)
	klog.V(6).InfoS("deleted node IP cache entry", "node", nodeName)
}

func cacheKey(namespace, podName string) string {
	return fmt.Sprintf("%s/%s", namespace, podName)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
