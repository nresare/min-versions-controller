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

func makeObjectSink(logger log.Logger) controller.HandlerFunc {
	return func(context context.Context, obj runtime.Object) error {
		node := obj.(*corev1.Node)
		info := node.Status.NodeInfo
		logger.Infof("Node event: containerd: %s kubelet: %s", info.ContainerRuntimeVersion, info.KubeletVersion)
		return nil
	}
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

	RunUntilFailure(makeObjectSink(logger), makeNodesListWatch(client), logger)
}
