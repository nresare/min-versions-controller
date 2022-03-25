package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spotahome/kooper/v2/controller"
	"github.com/spotahome/kooper/v2/log"
	kooperlogrus "github.com/spotahome/kooper/v2/log/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func makeKubernetesClient() (*kubernetes.Clientset, error) {
	kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
	k8scfg, err := clientcmd.BuildConfigFromFlags("", kubehome)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(k8scfg)
}

func makeObjectSink(logger log.Logger, client *kubernetes.Clientset) controller.HandlerFunc {
	return func(context context.Context, obj runtime.Object) error {
		node := obj.(*corev1.Node)
		info := node.Status.NodeInfo
		newLabels, err := buildLabels(info.ContainerRuntimeVersion, info.KubeletVersion)
		if err != nil {
			return err
		}
		update := needsUpdate(node, newLabels)
		if update {
			for key, value := range newLabels {
				node.Labels[key] = value
			}
			logger.Infof("Node %s needs an update", node.Name)
			client.CoreV1().Nodes().Update(context, node, metav1.UpdateOptions{})

		} else {
			logger.Infof("Node %s does not need an update", node.Name)
		}
		return nil
	}
}

func needsUpdate(node *corev1.Node, newLabels map[string]string) bool {
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
			return client.CoreV1().Nodes().List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Nodes().Watch(context.TODO(), options)
		},
	}
}

func main() {
	logger := kooperlogrus.New(logrus.NewEntry(logrus.New()))
	client, err := makeKubernetesClient()
	if err != nil {
		panic(fmt.Errorf("failed to make client: %w", err))
	}

	RunUntilFailure(makeObjectSink(logger, client), makeNodesListWatch(client), logger)
}
