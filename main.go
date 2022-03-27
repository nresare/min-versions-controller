package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spotahome/kooper/v2/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"reflect"
)

func makeKubernetesClient() (*kubernetes.Clientset, error) {
	kubernetesConfig, err := rest.InClusterConfig()
	if err != nil {
		kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
		kubernetesConfig, err = clientcmd.BuildConfigFromFlags("", kubehome)
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(kubernetesConfig)
}

func makeNodeHandler(logger *logrus.Entry, client *kubernetes.Clientset) controller.HandlerFunc {
	return func(context context.Context, obj runtime.Object) error {
		node := obj.(*corev1.Node)
		info := node.Status.NodeInfo
		newNodeLabels, err := buildNodeLabels(info.ContainerRuntimeVersion, info.KubeletVersion)
		if err != nil {
			return err
		}
		update := labelsNeedUpdate(node, newNodeLabels)
		if update {
			for key, value := range newNodeLabels {
				node.Labels[key] = value
			}
			logger.Infof("Node '%s' is not the desired state. Updating", node.Name)
			_, err = client.CoreV1().Nodes().Update(context, node, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			logger.Infof("Node '%s' does not need an update", node.Name)
		}
		return nil
	}
}

func getRequiredAffinity(pod *corev1.Pod) *corev1.NodeSelector {
	affinity := pod.Spec.Affinity
	if affinity == nil {
		return nil
	}
	nodeAffinity := affinity.NodeAffinity
	if nodeAffinity == nil {
		return nil
	}
	return nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
}

func in(value corev1.NodeSelectorTerm, slice []corev1.NodeSelectorTerm) bool {
	for _, item := range slice {
		if reflect.DeepEqual(item, value) {
			return true
		}
	}
	return false
}

func mergeRequired(first *corev1.NodeSelector, second *corev1.NodeSelector) *corev1.NodeSelector {
	var finalTerms []corev1.NodeSelectorTerm
	if first != nil {
		copy(finalTerms, first.NodeSelectorTerms)
	}

	for _, term := range second.NodeSelectorTerms {
		if !in(term, finalTerms) {
			finalTerms = append(finalTerms, term)
		}
	}
	return &corev1.NodeSelector{NodeSelectorTerms: finalTerms}
}

func replaceAffinityNodeSelector(pod *corev1.Pod, nodeSelector *corev1.NodeSelector) {
	affinity := pod.Spec.Affinity
	if affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution:  nodeSelector,
				PreferredDuringSchedulingIgnoredDuringExecution: nil,
			},
		}
		return
	}
	nodeAffinity := affinity.NodeAffinity
	if nodeAffinity == nil {
		affinity.NodeAffinity = &corev1.NodeAffinity{nodeSelector, nil}
		return
	}
	nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodeSelector
}

func makePodHandler(logger *logrus.Entry, client *kubernetes.Clientset) controller.HandlerFunc {
	return func(context context.Context, obj runtime.Object) error {
		pod := obj.(*corev1.Pod)

		wantedSelector, err := buildNodeSelector(
			pod.Labels["mwam.com/min-containerd-version"],
			pod.Labels["mwam.com/min-kubelet-version"],
		)
		if err != nil {
			return err
		}
		currentAffinity := getRequiredAffinity(pod)
		if len(wantedSelector.NodeSelectorTerms) == 0 || reflect.DeepEqual(wantedSelector, currentAffinity) {
			logger.Infof("Pod selector does not need updating for pod '%s'", pod.Name)
			return nil
		}

		replaceAffinityNodeSelector(pod, mergeRequired(currentAffinity, wantedSelector))

		logger.Infof("Updating node affinity selector for pod '%s'", pod.Name)
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = wantedSelector
		_, err = client.CoreV1().Pods(pod.Namespace).Update(context, pod, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		return nil
	}
}

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

func makePodsListWatch(client *kubernetes.Clientset) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().Pods("").List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Pods("").Watch(context.Background(), options)
		},
	}
}

func main() {
	logger := logrus.NewEntry(logrus.New())
	client, err := makeKubernetesClient()
	if err != nil {
		panic(fmt.Errorf("failed to make client: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorChannel := make(chan error)
	go func() {
		errorChannel <- RunControllerUntilFailure(makeNodeHandler(logger, client), makeNodesListWatch(client), logger, ctx)
	}()

	go func() {
		errorChannel <- RunControllerUntilFailure(makePodHandler(logger, client), makePodsListWatch(client), logger, ctx)
	}()

	err = <-errorChannel
	if err != nil {
		panic(err)
	}
}
