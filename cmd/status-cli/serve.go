package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	msignal "github.com/status-im/status-go/signal"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func serve(cCtx *cli.Context) error {
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

	cli, err := start(cCtx, name, port, apiModules, telemetryUrl)
	if err != nil {
		return err
	}
	defer cli.stop()

	// Using the mobile signal handler to listen for received messages
	// because if we call messenger.RetrieveAll() from different routines we will miss messages in one of them
	// and the retrieve messages loop is started when starting a node, so we needed a different appproach,
	// alternatively we could have implemented another notification mechanism in the messenger, but this signal is already in place
	msignal.SetMobileSignalHandler(msignal.MobileSignalHandler(func(s []byte) {
		if strings.Contains(string(s), `"type":"messages.new"`) {
			var resp MobileSignalEvent
			if err := json.Unmarshal(s, &resp); err != nil {
				logger.Errorf("unmarshaling 'messages.new' response: %v", err)
				return
			}

			for _, message := range resp.Event.Messages {
				logger.Infof("message received: %v (ID=%v)", message.Text, message.ID)
				// if request contact, accept it
				if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_SENT {
					if err = cli.sendContactRequestAcceptance(cCtx, message.ID); err != nil {
						logger.Errorf("accepting contact request: %v", err)
						return
					}
				}
			}
		}
	}))

	// Send contact request
	dest := cCtx.String(AddFlag)
	if dest != "" {
		err := cli.sendContactRequest(cCtx, dest)
		if err != nil {
			return err
		}
	}

	// nightly testrunner looks for this log to consider node as started
	logger.Info("retrieve messages...")

	ctx, cancel := context.WithCancel(cCtx.Context)
	go func() {
		// Wait for signal to exit
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	// Send message if mutual contact exists
	cli.sendMessageLoop2(ctx)

	logger.Info("Exiting")

	return nil
}

type MobileSignalEvent struct {
	Type  string `json:"type"`
	Event struct {
		Messages []*common.Message `json:"messages"`
	} `json:"event"`
}
