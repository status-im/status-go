package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/urfave/cli/v2"
)

var logger *zap.SugaredLogger

type StatusCLI struct {
	name      string
	messenger *protocol.Messenger
	waku      *wakuv2.Waku
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "dm",
				Aliases: []string{"d"},
				Usage:   "Send direct message",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "light",
						Usage: "Enable light mode",
					},
				},
				Action: func(cCtx *cli.Context) error {
					rawLogger, err := zap.NewDevelopment()
					if err != nil {
						log.Fatalf("Error initializing logger: %v", err)
					}
					logger = rawLogger.Sugar()

					logger.Info("Flags passed:")
					for _, flag := range cCtx.FlagNames() {
						logger.Infof("  %s: %v\n", flag, cCtx.Value(flag))
					}

					// Start Alice and Bob's messengers
					alice, err := startMessenger(cCtx, "Alice")
					if err != nil {
						return err
					}
					defer stopMessenger(alice)

					bob, err := startMessenger(cCtx, "Bob")
					if err != nil {
						return err
					}
					defer stopMessenger(bob)

					// Send contact request from Alice to Bob, bob accept the request
					msgID, err := sendContactRequest(cCtx, alice, bob)
					if err != nil {
						return err
					}

					err = sendContactRequestAcceptance(cCtx, bob, alice, msgID)
					if err != nil {
						return err
					}

					// Send DM between alice to bob
					err = sendDirectMessage(cCtx, alice, bob, "Hello Bob!")
					if err != nil {
						return err
					}

					err = sendDirectMessage(cCtx, bob, alice, "Hello Alice!")
					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}

func startMessenger(cCtx *cli.Context, name string) (*StatusCLI, error) {
	logger.Infof("[%s] starting messager\n", name)

	userLogger := setupLogger(name)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := &wakuv2.Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = api.DefaultWakuNodes[api.DefaultFleet]
	config.DiscoveryLimit = 20
	config.LightClient = cCtx.Bool("light")
	node, err := wakuv2.New("", "", config, userLogger, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	err = node.Start()
	if err != nil {
		return nil, err
	}

	appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	madb, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
	if err != nil {
		return nil, err
	}
	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")
	opitons := []protocol.Option{
		protocol.WithCustomLogger(userLogger),
		protocol.WithDatabase(appDb),
		protocol.WithWalletDatabase(walletDb),
		protocol.WithMultiAccounts(madb),
		protocol.WithAccount(iai.ToMultiAccount()),
		protocol.WithDatasync(),
		protocol.WithToplevelDatabaseMigrations(),
		protocol.WithBrowserDatabase(nil),
	}
	messenger, err := protocol.NewMessenger(
		fmt.Sprintf("%s-node", strings.ToLower(name)),
		privateKey,
		gethbridge.NewNodeBridge(nil, nil, node),
		uuid.New().String(),
		nil,
		opitons...,
	)
	if err != nil {
		return nil, err
	}

	err = messenger.Init()
	if err != nil {
		return nil, err
	}

	_, err = messenger.Start()
	if err != nil {
		return nil, err
	}

	id := types.EncodeHex(crypto.FromECDSAPub(messenger.IdentityPublicKey()))
	logger.Infof("[%s] messenger started, id: %s\n", name, id)

	time.Sleep(3 * time.Second)

	data := StatusCLI{
		name:      name,
		messenger: messenger,
		waku:      node,
	}

	return &data, nil
}

func stopMessenger(cli *StatusCLI) {
	err := cli.messenger.Shutdown()
	if err != nil {
		logger.Error(err)
	}

	err = cli.waku.Stop()
	if err != nil {
		logger.Error(err)
	}
}

func sendContactRequest(cCtx *cli.Context, from, to *StatusCLI) (string, error) {
	destID := types.EncodeHex(crypto.FromECDSAPub(to.messenger.IdentityPublicKey()))
	logger.Infof("[%s] send contact request to %s, contact id: %s\n", from.name, to.name, destID)
	request := &requests.SendContactRequest{
		ID:      destID,
		Message: "Hello!",
	}
	resp, err := from.messenger.SendContactRequest(cCtx.Context, request)
	logger.Infof("[%s] function SendContactRequest response.messages: %s\n", from.name, resp.Messages())
	if err != nil {
		return "", err
	}

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("[%s] no contact request from %s", to.name, from.name),
	)
	if err != nil {
		return "", err
	}

	msg := protocol.FindFirstByContentType(respTo.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	logger.Infof("[%s] receive contact request response: %s\n", to.name, msg.Text)

	return msg.ID, nil
}

func sendContactRequestAcceptance(cCtx *cli.Context, from, to *StatusCLI, msgID string) error {
	logger.Infof("[%s] send contact request acceptance to %s\n", from.name, to.name)
	resp, err := from.messenger.AcceptContactRequest(cCtx.Context, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	logger.Infof("[%s] function AcceptContactRequest response: %v\n", from.name, resp.Messages())

	fromContacts := from.messenger.MutualContacts()
	logger.Infof("[%s] contacts number: %d\n", from.name, len(fromContacts))

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("[%s] contact request acceptance not received from %s", to.name, from.name),
	)
	if err != nil {
		return err
	}

	msg := protocol.FindFirstByContentType(respTo.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED)
	logger.Infof("[%s] got message: %s\n", to.name, msg.Text)

	toContacts := to.messenger.MutualContacts()
	logger.Infof("[%s] contacts number: %d\n", to.name, len(toContacts))

	return nil
}

func sendDirectMessage(cCtx *cli.Context, from, to *StatusCLI, text string) error {
	chat := from.messenger.Chat(from.messenger.MutualContacts()[0].ID)
	logger.Infof("[%s] chat with contact id: %s\n", from.name, chat.ID)

	clock, timestamp := chat.NextClockAndTimestamp(from.messenger.GetTransport())
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.LocalChatID = chat.ID
	inputMessage.Clock = clock
	inputMessage.Timestamp = timestamp
	inputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = text

	resp, err := from.messenger.SendChatMessage(cCtx.Context, inputMessage)
	if err != nil {
		return err
	}
	logger.Infof("[%s] function SendChatMessage response.messages: %v\n", from.name, resp.Messages())

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool { return len(r.Messages()) > 0 },
		fmt.Sprintf("[%s] not receive message from %s", to.name, from.name),
	)
	if err != nil {
		return err
	}
	logger.Infof("[%s] receive message from %s: %s\n", to.name, from.name, respTo.Chats()[0].LastMessage.Text)

	return nil
}

func setupLogger(file string) *zap.Logger {
	logFile := fmt.Sprintf("%s.log", strings.ToLower(file))
	logSettings := logutils.LogSettings{
		Enabled:         true,
		MobileSystem:    false,
		Level:           "DEBUG",
		File:            logFile,
		MaxSize:         100,
		MaxBackups:      3,
		CompressRotated: true,
	}
	if err := logutils.OverrideRootLogWithConfig(logSettings, false); err != nil {
		logger.Fatalf("Error initializing logger: %v", err)
	}

	newLogger := logutils.ZapLogger()

	return newLogger
}
