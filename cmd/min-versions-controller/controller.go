package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spotahome/kooper/v2/controller"
	kooperlogrus "github.com/spotahome/kooper/v2/log/logrus"
	"k8s.io/client-go/tools/cache"
)

func RunControllerUntilFailure(
	handlerFunction controller.HandlerFunc,
	listerWatcher cache.ListerWatcher,
	logger *logrus.Entry,
	ctx context.Context,
) error {
	retriever := controller.MustRetrieverFromListerWatcher(listerWatcher)

	config := &controller.Config{
		Name:      "min-versions-controller",
		Handler:   handlerFunction,
		Retriever: retriever,
		Logger:    kooperlogrus.New(logger),
	}

	ctrl, err := controller.New(config)
	if err != nil {
		panic(fmt.Errorf("could not create controller: %w", err))
	}

	err = ctrl.Run(ctx)
	if err != nil {
		return fmt.Errorf("controller failed: %w", err)
	}
	return nil
}
