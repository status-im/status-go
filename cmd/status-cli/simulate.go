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

	// Start Alice and Bob's messengers
	apiModules := cCtx.String(APIModulesFlag)

	alice, err := start(cCtx, "Alice", 0, apiModules)
	if err != nil {
		return err
	}
	defer alice.stop()

	bob, err := start(cCtx, "Bob", 0, apiModules)
	if err != nil {
		return err
	}
	defer bob.stop()

	// Retrieve for messages
	msgCh := make(chan string)
	var wg sync.WaitGroup

	wg.Add(1)
	go alice.retrieveMessagesLoop(ctx, RetrieveInterval, nil, &wg)
	wg.Add(1)
	go bob.retrieveMessagesLoop(ctx, RetrieveInterval, msgCh, &wg)

	// Send contact request from Alice to Bob, bob accept the request
	time.Sleep(WaitingInterval)
	destID := bob.messenger.GetSelfContact().ID
	err = alice.sendContactRequest(cCtx, destID)
	if err != nil {
		return err
	}

	msgID := <-msgCh
	err = bob.sendContactRequestAcceptance(cCtx, msgID)
	if err != nil {
		return err
	}

	// Send DM between alice to bob
	interactive := cCtx.Bool(InteractiveFlag)
	if interactive {
		sem := make(chan struct{}, 1)
		wg.Add(1)
		go alice.sendMessageLoop(ctx, SendInterval, &wg, sem, cancel)
		wg.Add(1)
		go bob.sendMessageLoop(ctx, SendInterval, &wg, sem, cancel)
	} else {
		time.Sleep(WaitingInterval)
		for i := 0; i < cCtx.Int(CountFlag); i++ {
			err = alice.sendDirectMessage(ctx, fmt.Sprintf("message from alice, number: %d", i+1))
			if err != nil {
				return err
			}
			time.Sleep(WaitingInterval)

			err = bob.sendDirectMessage(ctx, fmt.Sprintf("message from bob, number: %d", i+1))
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
