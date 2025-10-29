package metrics

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// newNodeEventHandler wires the cache updates required for node events.
func newNodeEventHandler(store *Cache) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			node := toNode(obj)
			if node == nil {
				return
			}
			ip := internalNodeIP(node)
			klog.V(4).InfoS("node added/updated", "node", node.Name, "ip", ip)
			store.StoreNodeIP(node.Name, ip)
		},
		UpdateFunc: func(_, newObj any) {
			node := toNode(newObj)
			if node == nil {
				return
			}
			ip := internalNodeIP(node)
			klog.V(5).InfoS("node updated", "node", node.Name, "ip", ip)
			store.StoreNodeIP(node.Name, ip)
		},
		DeleteFunc: func(obj any) {
			node := toNode(obj)
			if node == nil {
				return
			}
			klog.V(4).InfoS("node deleted", "node", node.Name)
			store.DeleteNode(node.Name)
		},
	}
}

// newPodEventHandler wires the cache updates required for pod events.
func newPodEventHandler(store *Cache) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod := toPod(obj)
			if pod == nil {
				return
			}
			klog.V(6).InfoS("pod added", "pod", cacheKey(pod.Namespace, pod.Name))
			store.StorePodLabels(pod.Namespace, pod.Name, pod.Labels)
		},
		UpdateFunc: func(_, newObj any) {
			pod := toPod(newObj)
			if pod == nil {
				return
			}
			klog.V(6).InfoS("pod updated", "pod", cacheKey(pod.Namespace, pod.Name))
			store.StorePodLabels(pod.Namespace, pod.Name, pod.Labels)
		},
		DeleteFunc: func(obj any) {
			pod := toPod(obj)
			if pod == nil {
				return
			}
			klog.V(6).InfoS("pod deleted", "pod", cacheKey(pod.Namespace, pod.Name))
			store.DeletePodLabels(pod.Namespace, pod.Name)
		},
	}
}

func internalNodeIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func toNode(obj any) *corev1.Node {
	switch typed := obj.(type) {
	case *corev1.Node:
		return typed
	case cache.DeletedFinalStateUnknown:
		node, _ := typed.Obj.(*corev1.Node)
		return node
	default:
		return nil
	}
}

func toPod(obj any) *corev1.Pod {
	switch typed := obj.(type) {
	case *corev1.Pod:
		return typed
	case cache.DeletedFinalStateUnknown:
		pod, _ := typed.Obj.(*corev1.Pod)
		return pod
	default:
		return nil
	}
}
