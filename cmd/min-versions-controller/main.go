package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spotahome/kooper/v2/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"reflect"
)

func makeKubernetesClient() (*kubernetes.Clientset, error) {
	if len(os.Args) > 1 && os.Args[1] == "dev" {
		kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
		kubernetesConfig, err := clientcmd.BuildConfigFromFlags("", kubehome)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(kubernetesConfig)
	}
	kubernetesConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(kubernetesConfig)
}

// The returned HandlerFunc is responding to new or changed Node objects on the cluster. It
// is the entry point to the logic that add labels to each Node containing version information.
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

func makeSimpleMutator(logger *logrus.Entry) SimpleMutator {
	return func(obj metav1.Object) (metav1.Object, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			logger.Warningf("received object is not a Pod")
			return nil, nil
		}
		wantedSelector, err := buildNodeSelector(
			pod.Labels["mwam.com/min-containerd-version"],
			pod.Labels["mwam.com/min-kubelet-version"],
		)
		if err != nil {
			return nil, nil
		}
		currentAffinity := getRequiredAffinity(pod)
		if len(wantedSelector.NodeSelectorTerms) == 0 || reflect.DeepEqual(wantedSelector, currentAffinity) {
			logger.Infof("Pod selector does not need updating for pod '%s'", pod.Name)
			return nil, nil
		}

		logger.Infof("updating affinity selector for pod '%s': %s", pod.Name, wantedSelector)
		replaceAffinityNodeSelector(pod, mergeRequired(currentAffinity, wantedSelector))

		return pod, nil
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
		errorChannel <- RunWehbookFramework(logger, makeSimpleMutator(logger))
	}()

	err = <-errorChannel
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error running app: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
