package main

import (
	"context"
	"fmt"
	"github.com/spotahome/kooper/v2/controller"
	"github.com/spotahome/kooper/v2/log"
	"k8s.io/client-go/tools/cache"
)

func RunUntilFailure(handlerFunction controller.HandlerFunc,
	listerWatcher cache.ListerWatcher,
	logger log.Logger) {
	retriever := controller.MustRetrieverFromListerWatcher(listerWatcher)

	config := &controller.Config{
		Name:      "example-controller",
		Handler:   controller.HandlerFunc(handlerFunction),
		Retriever: retriever,
		Logger:    logger,
	}

	ctrl, err := controller.New(config)
	if err != nil {
		panic(fmt.Errorf("could not create controller: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = ctrl.Run(ctx)
	if err != nil {
		panic(fmt.Errorf("controller failed: %w", err))
	}
}
