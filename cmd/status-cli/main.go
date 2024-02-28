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

const LightFlag = "light"
const InteractiveFlag = "interactive"

var logger *zap.SugaredLogger

type StatusCLI struct {
	name      string
	messenger *protocol.Messenger
	waku      *wakuv2.Waku
	logger    *zap.SugaredLogger
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
						Name:  LightFlag,
						Usage: "Enable light mode",
					},
					&cli.BoolFlag{
						Name:    InteractiveFlag,
						Aliases: []string{"i"},
						Usage:   "Use interactive mode",
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
					retrieveMessagesLoop(alice, 300*time.Millisecond, cCtx.Done())
					retrieveMessagesLoop(bob, 300*time.Millisecond, cCtx.Done())

					interactive := cCtx.Bool(InteractiveFlag)

					message := "Hello Bob!"
					if interactive {
						alice.logger.Info("Enter message to send to Bob: ")
						_, err := fmt.Scanln(&message)
						if err != nil {
							return err
						}
					}

					err = sendDirectMessage(cCtx, alice, message)
					if err != nil {
						return err
					}

					respond := "Hello Alice!"
					if interactive {
						bob.logger.Info("Enter message to send to Alice: ")
						_, err := fmt.Scanln(&respond)
						if err != nil {
							return err
						}
					}
					err = sendDirectMessage(cCtx, bob, respond)
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
	namedLogger := logger.Named(name)
	namedLogger.Info("starting messager")

	userLogger := setupLogger(name)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := &wakuv2.Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = api.DefaultWakuNodes[api.DefaultFleet]
	config.DiscoveryLimit = 20
	config.LightClient = cCtx.Bool(LightFlag)
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
	namedLogger.Info("messenger started, id: ", id)

	time.Sleep(3 * time.Second)

	data := StatusCLI{
		name:      name,
		messenger: messenger,
		waku:      node,
		logger:    namedLogger,
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
	from.logger.Info("send contact request, contact id: ", destID)
	request := &requests.SendContactRequest{
		ID:      destID,
		Message: "Hello!",
	}
	resp, err := from.messenger.SendContactRequest(cCtx.Context, request)
	from.logger.Info("function SendContactRequest response.messages: ", resp.Messages())
	if err != nil {
		return "", err
	}

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("%s didn't get contact request from %s", to.name, from.name),
	)
	if err != nil {
		return "", err
	}

	msg := protocol.FindFirstByContentType(respTo.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	to.logger.Info("receive contact request response: ", msg.Text)

	return msg.ID, nil
}

func sendContactRequestAcceptance(cCtx *cli.Context, from, to *StatusCLI, msgID string) error {
	from.logger.Info("send contact request acceptance to: ", to.name)
	resp, err := from.messenger.AcceptContactRequest(cCtx.Context, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	from.logger.Info("function AcceptContactRequest response: ", resp.Messages())

	fromContacts := from.messenger.MutualContacts()
	from.logger.Info("contacts number: ", len(fromContacts))

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("%s contact request acceptance not received from %s", to.name, from.name),
	)
	if err != nil {
		return err
	}

	msg := protocol.FindFirstByContentType(respTo.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED)
	to.logger.Info("got message: ", msg.Text)

	toContacts := to.messenger.MutualContacts()
	to.logger.Info("contacts number: ", len(toContacts))

	return nil
}

func sendDirectMessage(cCtx *cli.Context, from *StatusCLI, text string) error {
	chat := from.messenger.Chat(from.messenger.MutualContacts()[0].ID)
	from.logger.Info("chat with contact id: ", chat.ID)

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
	from.logger.Info("function SendChatMessage response.messages: ", resp.Messages())

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

func retrieveMessagesLoop(cli *StatusCLI, tick time.Duration, cancel <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(tick)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				response, err := cli.messenger.RetrieveAll()
				if err != nil {
					cli.logger.Error("failed to retrieve raw messages", "err", err)
					continue
				}
				if response != nil && len(response.Chats()) != 0 {
					for _, chat := range response.Chats() {
						cli.logger.Infof("receive message from: %s\n", chat.LastMessage.Text)
					}
				}
			case <-cancel:
				return
			}
		}
	}()
}
