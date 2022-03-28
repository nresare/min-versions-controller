package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const (
	listenPort      = 8080
	certFilePath    = "/etc/webhook-certs/server.crt"
	certKeyFilePath = "/etc/webhook-certs/server.key"
)

// SimpleMutator is a simplification of the MutatorFunc api. It receives
// an object and either returns nil which indicates that the object should
// not be changed, or a changed object.
type SimpleMutator func(obj metav1.Object) (metav1.Object, error)

func RunWehbookFramework(logger *logrus.Entry, simpleMutator SimpleMutator) error {
	logger.Infof("Starting webserver, listening on port %d", listenPort)
	wrappedLogger := kwlogrus.NewLogrus(logger)

	m := func(
		ctx context.Context,
		ar *kwhmodel.AdmissionReview,
		obj metav1.Object,
	) (*kwhmutating.MutatorResult, error) {
		obj, err := simpleMutator(obj)
		if err != nil {
			return nil, err
		}
		return &kwhmutating.MutatorResult{MutatedObject: obj}, nil
	}

	webhook, err := kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "safeServiceMonitor",
		Mutator: kwhmutating.MutatorFunc(m),
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
