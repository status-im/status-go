package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func simulate(cCtx *cli.Context) error {
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

	logger.Info("Running simulate command, flags passed:")
	for _, flag := range SimulateFlags {
		logger.Infof("-%s %v", flag.Names()[0], cCtx.Value(flag.Names()[0]))
	}

	// Start messengers
	apiModules := cCtx.String(APIModulesFlag)
	telemetryUrl := cCtx.String(TelemetryServerURLFlag)

	alice, err := start("Alice", 0, apiModules, telemetryUrl)
	if err != nil {
		return err
	}
	defer alice.stop()

	charlie, err := start("Charlie", 0, apiModules, telemetryUrl)
	if err != nil {
		return err
	}
	defer charlie.stop()

	// Retrieve for messages
	msgCh := make(chan string)
	var wg sync.WaitGroup

	wg.Add(1)
	go alice.retrieveMessagesLoop(ctx, RetrieveInterval, nil, &wg)
	wg.Add(1)
	go charlie.retrieveMessagesLoop(ctx, RetrieveInterval, msgCh, &wg)

	// Send contact request from Alice to Charlie, charlie accept the request
	time.Sleep(WaitingInterval)
	destID := charlie.messenger.GetSelfContact().ID
	err = alice.sendContactRequest(ctx, destID)
	if err != nil {
		return err
	}

	msgID := <-msgCh
	err = charlie.sendContactRequestAcceptance(ctx, msgID)
	if err != nil {
		return err
	}
	time.Sleep(WaitingInterval)

	// Send DM between alice to charlie
	interactive := cCtx.Bool(InteractiveFlag)
	if interactive {
		interactiveSendMessageLoop(ctx, alice, charlie)
	} else {
		for i := 0; i < cCtx.Int(CountFlag); i++ {
			err = alice.sendDirectMessage(ctx, fmt.Sprintf("message from alice, number: %d", i+1))
			if err != nil {
				return err
			}
			time.Sleep(WaitingInterval)

			err = charlie.sendDirectMessage(ctx, fmt.Sprintf("message from charlie, number: %d", i+1))
			if err != nil {
				return err
			}
			time.Sleep(WaitingInterval)
		}
		cancel()
	}

	wg.Wait()
	logger.Info("Exiting")

	return nil
}
