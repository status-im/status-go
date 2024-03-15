package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/eth-node/types"

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

	logger.Info("Running dm command, flags passed:")
	for _, flag := range DmFlags {
		logger.Infof("-%s %v", flag.Names()[0], cCtx.Value(flag.Names()[0]))
	}

	// Start Alice and Bob's messengers
	alice, err := startMessenger(cCtx, "Alice", 0)
	if err != nil {
		return err
	}
	defer stopMessenger(alice)

	bob, err := startMessenger(cCtx, "Bob", 0)
	if err != nil {
		return err
	}
	defer stopMessenger(bob)

	// Retrieve for messages
	msgCh := make(chan string)
	var wg sync.WaitGroup

	wg.Add(1)
	go retrieveMessagesLoop(ctx, alice, RetrieveInterval, nil, &wg)
	wg.Add(1)
	go retrieveMessagesLoop(ctx, bob, RetrieveInterval, msgCh, &wg)

	// Send contact request from Alice to Bob, bob accept the request
	time.Sleep(WaitingInterval)
	destID := types.EncodeHex(crypto.FromECDSAPub(bob.messenger.IdentityPublicKey()))
	err = sendContactRequest(cCtx, alice, destID)
	if err != nil {
		return err
	}

	msgID := <-msgCh
	err = sendContactRequestAcceptance(cCtx, bob, msgID)
	if err != nil {
		return err
	}

	// Send DM between alice to bob
	interactive := cCtx.Bool(InteractiveFlag)
	if interactive {
		sem := make(chan struct{}, 1)
		wg.Add(1)
		go sendMessageLoop(ctx, alice, SendInterval, &wg, sem, cancel)
		wg.Add(1)
		go sendMessageLoop(ctx, bob, SendInterval, &wg, sem, cancel)
	} else {
		time.Sleep(WaitingInterval)
		for i := 0; i < cCtx.Int(CountFlag); i++ {
			err = sendDirectMessage(ctx, alice, "hello bob :)")
			if err != nil {
				return err
			}
			time.Sleep(WaitingInterval)

			err = sendDirectMessage(ctx, bob, "hello Alice ~")
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
