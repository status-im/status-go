package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func serve(cCtx *cli.Context) error {
	ctx, cancel := context.WithCancel(cCtx.Context)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	rawLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
	logger = rawLogger.Sugar()

	logger.Info("Running serve command, flags passed:")
	for _, flag := range ServeFlags {
		logger.Infof("-%s %v", flag.Names()[0], cCtx.Value(flag.Names()[0]))
	}

	name := cCtx.String(NameFlag)
	port := cCtx.Int(PortFlag)
	apiModules := cCtx.String(APIModulesFlag)

	cli, err := start(cCtx, name, port, apiModules)
	if err != nil {
		return err
	}
	defer cli.stop()

	// Retrieve for messages
	var wg sync.WaitGroup
	msgCh := make(chan string)

	wg.Add(1)
	go cli.retrieveMessagesLoop(ctx, RetrieveInterval, msgCh, &wg)

	// Send and accept contact request
	dest := cCtx.String(AddFlag)
	if dest != "" {
		err := cli.sendContactRequest(cCtx, dest)
		if err != nil {
			return err
		}
	}

	go func() {
		msgID := <-msgCh
		err = cli.sendContactRequestAcceptance(cCtx, msgID)
		if err != nil {
			logger.Error(err)
			return
		}
	}()

	// Send message if mutual contact exists
	sem := make(chan struct{}, 1)
	wg.Add(1)
	go cli.sendMessageLoop(ctx, SendInterval, &wg, sem, cancel)

	wg.Wait()
	logger.Info("Exiting")

	return nil
}
