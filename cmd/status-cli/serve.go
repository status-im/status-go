package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	msignal "github.com/status-im/status-go/signal"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func serve(cCtx *cli.Context, useLastAccount bool) error {
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
	telemetryUrl := cCtx.String(TelemetryServerURLFlag)
	interactive := cCtx.Bool(InteractiveFlag)
	dest := cCtx.String(AddFlag)
	keyUID := cCtx.String(KeyUIDFlag)

	cli, err := start(name, port, apiModules, telemetryUrl, useLastAccount, keyUID)
	if err != nil {
		return err
	}
	defer cli.stop()

	// Using the mobile signal handler to listen for received messages
	// because if we call messenger.RetrieveAll() from different routines we will miss messages in one of them
	// and the retrieve messages loop is started when starting a node, so we needed a different appproach,
	// alternatively we could have implemented another notification mechanism in the messenger, but this signal is already in place
	msignal.SetMobileSignalHandler(msignal.MobileSignalHandler(func(s []byte) {
		var ev MobileSignalEvent
		if err := json.Unmarshal(s, &ev); err != nil {
			logger.Error("unmarshaling signal event", zap.Error(err), zap.String("event", string(s)))
			return
		}

		if ev.Type == msignal.EventNewMessages {
			for _, message := range ev.Event.Messages {
				logger.Infof("message received: %v (ID=%v)", message.Text, message.ID)
				// if request contact, accept it
				if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_SENT {
					if err := cli.sendContactRequestAcceptance(cCtx.Context, message.ID); err != nil {
						logger.Errorf("accepting contact request: %v", err)
						return
					}
				}
			}
		}
	}))

	// Send contact request
	if dest != "" {
		err := cli.sendContactRequest(cCtx.Context, dest)
		if err != nil {
			return err
		}
	}

	// nightly testrunner looks for this log to consider node as started
	logger.Info("retrieve messages...")

	if interactive {
		ctx, cancel := context.WithCancel(cCtx.Context)
		go func() {
			waitForSigExit()
			cancel()
		}()
		interactiveSendMessageLoop(ctx, cli)
	} else {
		waitForSigExit()
	}

	logger.Info("Exiting")

	return nil
}

type MobileSignalEvent struct {
	Type  string `json:"type"`
	Event struct {
		Messages []*common.Message `json:"messages"`
	} `json:"event"`
}

func waitForSigExit() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
