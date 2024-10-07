package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	msignal "github.com/status-im/status-go/signal"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func serve(cCtx *cli.Context) error {
	name := cCtx.String(NameFlag)
	port := cCtx.Int(PortFlag)
	apiModules := cCtx.String(APIModulesFlag)
	telemetryUrl := cCtx.String(TelemetryServerURLFlag)
	interactive := cCtx.Bool(InteractiveFlag)
	dest := cCtx.String(AddFlag)
	keyUID := cCtx.String(KeyUIDFlag)
	isDebugLevel := cCtx.Bool(DebugLevel)
	fleet := cCtx.String(FleetFlag)
	cmdName := cCtx.Command.Name

	logger, err := getSLogger(isDebugLevel)
	if err != nil {
		zap.S().Fatalf("Error initializing logger: %v", err)
	}
	logger.Infof("Running %v command, with:\n%v", cmdName, flagsUsed(cCtx))

	logger = logger.Named(name)

	cli, err := start(StartParams{
		Name:         name,
		Port:         port,
		APIModules:   apiModules,
		TelemetryURL: telemetryUrl,
		KeyUID:       keyUID,
		Fleet:        fleet,
	}, logger)
	if err != nil {
		return err
	}
	defer cli.stop()

	// Using the mobile signal handler to listen for received messages
	// because if we call messenger.RetrieveAll() from different routines we will miss messages in one of them
	// and the retrieve messages loop is started when starting a node, so we needed a different appproach,
	// alternatively we could have implemented another notification mechanism in the messenger, but this signal is already in place
	msignal.SetMobileSignalHandler(msignal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		if err := json.Unmarshal(s, &evt); err != nil {
			logger.Error("unmarshaling event type", zap.Error(err), zap.String("event", string(s)))
			return
		}

		switch evt.Type {
		case msignal.EventNewMessages:
			var ev EventNewMessages
			if err := json.Unmarshal(evt.Event, &ev); err != nil {
				logger.Error("unmarshaling new message event", zap.Error(err), zap.Any("event", evt.Event))
				return
			}
			for _, message := range ev.Messages {
				logger.Infof("message received: %v (ID=%v)", message.Text, message.ID)
				// if request contact, accept it
				if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_SENT {
					if err := cli.sendContactRequestAcceptance(cCtx.Context, message.ID); err != nil {
						logger.Errorf("accepting contact request: %v", err)
						return
					}
				}
			}
		case "local-notifications":
			var ev LocalNotification
			if err := json.Unmarshal(evt.Event, &ev); err != nil {
				logger.Error("unmarshaling local notification event", zap.Error(err), zap.Any("event", evt.Event))
				return
			}
			logger.Infof("local notification: %v, title: %v, id: %v", ev.Category, ev.Title, ev.ID)
		case msignal.EventMesssageDelivered:
			var ev msignal.MessageDeliveredSignal
			if err := json.Unmarshal(evt.Event, &ev); err != nil {
				logger.Error("unmarshaling message delivered event", zap.Error(err), zap.Any("event", evt.Event))
				return
			}
			logger.Infof("message delivered: %v", ev.MessageID)
		default:
			logger.Debugf("received event type '%v'\t%v", evt.Type, string(evt.Event))
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

type EventType struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

type EventNewMessages struct {
	Messages []*common.Message `json:"messages"`
}

type LocalNotification struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

func waitForSigExit() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
