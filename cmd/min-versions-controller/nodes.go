package main

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"strconv"
)

// Returns true if all newLabels are present and has the right value in node.Labels
func labelsNeedUpdate(node *corev1.Node, newLabels map[string]string) bool {
	for key, value := range newLabels {
		current, ok := node.Labels[key]
		if !ok {
			return true
		}
		if current != value {
			return true
		}
	}
	return false
}

func buildNodeLabels(containerdVersion string, kubeletVersion string) (map[string]string, error) {
	containerdMajor, containerdMinor, err := getContainerdVersion(containerdVersion)
	if err != nil {
		return nil, err
	}
	kubeletMajor, kubeletMinor, err := getMajorMinor(kubeletVersion)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		ContainerdMajorKey: strconv.FormatUint(containerdMajor, 10),
		ContainerdMinorKey: strconv.FormatUint(containerdMinor, 10),
		KubeletMajorKey:    strconv.FormatUint(kubeletMajor, 10),
		KubeletMinorKey:    strconv.FormatUint(kubeletMinor, 10),
	}, nil
}

func buildGreaterThanTerm(key string, value uint64) v1.NodeSelectorTerm {
	return v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{{
			key,
			v1.NodeSelectorOpGt,
			[]string{strconv.FormatUint(value, 10)},
		}},
	}
}

// This might look a bit magical, essentially this creates a struct of functions that specify
// how data should be listed and watched. Since the Kubernetes client is typed, one of these
// needs to be created per object kind.
func makeNodesListWatch(client *kubernetes.Clientset) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().Nodes().List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Nodes().Watch(context.Background(), options)
		},
	}
}
