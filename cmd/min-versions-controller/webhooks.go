package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const (
	listenPort      = 8080
	certFilePath    = "/etc/webhook-certs/server.crt"
	certKeyFilePath = "/etc/webhook-certs/server.key"
)

func makeMutator(logger *logrus.Entry) kwhmutating.MutatorFunc {
	return func(ctx context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			logger.Warningf("received object is not a Pod")
			return &kwhmutating.MutatorResult{}, nil
		}
		logger.Infof("In webhook: Not modifying Pod '%s'", pod.Name)

		return &kwhmutating.MutatorResult{}, nil
	}
}

func RunWebServer(logger *logrus.Entry, mutatorFunction kwhmutating.MutatorFunc) error {
	logger.Infof("Starting webserver, listening on port %d", listenPort)
	wrappedLogger := kwlogrus.NewLogrus(logger)

	mutator := kwhmutating.MutatorFunc(mutatorFunction)

	webhook, err := kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "safeServiceMonitor",
		Mutator: mutator,
		Logger:  wrappedLogger,
	})
	if err != nil {
		return err
	}

	handler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{
		Webhook: webhook,
		Logger:  wrappedLogger,
	})
	if err != nil {
		return err
	}
	return http.ListenAndServeTLS(
		fmt.Sprintf(":%d", listenPort),
		certFilePath,
		certKeyFilePath,
		handler,
	)
}
