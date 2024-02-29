package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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
const NameFlag = "name"
const AddFlag = "add"

const RetrieveInterval = 300 * time.Millisecond
const SendInterval = 1 * time.Second
const WaitingInterval = 5 * time.Second

var CommonFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    LightFlag,
		Aliases: []string{"l"},
		Usage:   "Enable light mode",
	},
}

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
				Flags: append([]cli.Flag{
					&cli.BoolFlag{
						Name:    InteractiveFlag,
						Aliases: []string{"i"},
						Usage:   "Use interactive mode",
					},
				}, CommonFlags...),
				Action: func(cCtx *cli.Context) error {
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

					// Retrieve for messages
					msgCh := make(chan string)
					msgCh2 := make(chan string)
					var wg sync.WaitGroup

					wg.Add(1)
					go retrieveMessagesLoop(alice, RetrieveInterval, msgCh, ctx, &wg)
					wg.Add(1)
					go retrieveMessagesLoop(bob, RetrieveInterval, msgCh2, ctx, &wg)

					// Send contact request from Alice to Bob, bob accept the request
					time.Sleep(WaitingInterval)
					destId := types.EncodeHex(crypto.FromECDSAPub(bob.messenger.IdentityPublicKey()))
					err = sendContactRequest(cCtx, alice, destId)
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
						go sendMessageLoop(alice, SendInterval, ctx, &wg, sem, cancel)
						wg.Add(1)
						go sendMessageLoop(bob, SendInterval, ctx, &wg, sem, cancel)
					} else {
						time.Sleep(WaitingInterval)
						for i := 0; i < 3; i++ {
							if len(bob.messenger.MutualContacts()) == 0 {
								continue
							}
							err = sendDirectMessage(alice, "Hello Bob, I'm Alice!", ctx)
							if err != nil {
								return err
							}
							time.Sleep(WaitingInterval)

							err = sendDirectMessage(bob, "Hello Alice, I'm Bob!", ctx)
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
				},
			},
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Start a server to send and receive messages",
				Flags: append([]cli.Flag{
					&cli.StringFlag{
						Name:    NameFlag,
						Aliases: []string{"n"},
						Value:   "Alice",
						Usage:   "Name of the user",
					},
					&cli.StringFlag{
						Name:    AddFlag,
						Aliases: []string{"a"},
						Usage:   "Add a friend with the public key",
					},
				}, CommonFlags...),
				Action: func(cCtx *cli.Context) error {
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
					for _, flag := range cCtx.FlagNames() {
						logger.Infof("  %s: %v\n", flag, cCtx.Value(flag))
					}

					name := cCtx.String(NameFlag)

					// Start Alice and Bob's messengers
					alice, err := startMessenger(cCtx, name)
					if err != nil {
						return err
					}
					defer stopMessenger(alice)

					// Retrieve for messages
					var wg sync.WaitGroup
					msgCh := make(chan string)
					contactCh := make(chan int)

					wg.Add(1)
					go retrieveMessagesLoop(alice, RetrieveInterval, msgCh, ctx, &wg)

					// Send contact request from Alice to Bob, bob accept the request
					dest := cCtx.String(AddFlag)
					if dest != "" {
						err := sendContactRequest(cCtx, alice, dest)
						if err != nil {
							return err
						}
					}

					go func() {
						msgID := <-msgCh
						err = sendContactRequestAcceptance(cCtx, alice, msgID)
						if err != nil {
							logger.Error(err)
							return
						}
					}()

					// Send message if mutual contact exists
					<-contactCh
					sem := make(chan struct{}, 1)
					wg.Add(1)
					go sendMessageLoop(alice, SendInterval, ctx, &wg, sem, cancel)

					wg.Wait()
					logger.Info("Exiting")

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
	namedLogger.Info("messenger started, public key: ", id)

	time.Sleep(WaitingInterval)

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

func sendContactRequest(cCtx *cli.Context, from *StatusCLI, toId string) error {
	from.logger.Info("send contact request, contact public key: ", toId)
	request := &requests.SendContactRequest{
		ID:      toId,
		Message: "Hello!",
	}
	resp, err := from.messenger.SendContactRequest(cCtx.Context, request)
	from.logger.Info("function SendContactRequest response.messages: ", resp.Messages())
	if err != nil {
		return err
	}

	return nil
}

func sendContactRequestAcceptance(cCtx *cli.Context, from *StatusCLI, msgID string) error {
	from.logger.Info("accept contact request, message ID: ", msgID)
	resp, err := from.messenger.AcceptContactRequest(cCtx.Context, &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	from.logger.Info("function AcceptContactRequest response: ", resp.Messages())

	fromContacts := from.messenger.MutualContacts()
	from.logger.Info("contacts number: ", len(fromContacts))

	return nil
}

func sendDirectMessage(from *StatusCLI, text string, ctx context.Context) error {
	chat := from.messenger.Chat(from.messenger.MutualContacts()[0].ID)
	from.logger.Infof("send message (%s) to contact: %s", text, chat.ID)

	clock, timestamp := chat.NextClockAndTimestamp(from.messenger.GetTransport())
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.LocalChatID = chat.ID
	inputMessage.Clock = clock
	inputMessage.Timestamp = timestamp
	inputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = text

	_, err := from.messenger.SendChatMessage(ctx, inputMessage)
	if err != nil {
		return err
	}

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

func retrieveMessagesLoop(cli *StatusCLI, tick time.Duration, msgCh chan string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	cli.logger.Info("retrieve messages...")

	for {
		select {
		case <-ticker.C:
			response, err := cli.messenger.RetrieveAll()
			if err != nil {
				cli.logger.Error("failed to retrieve raw messages", "err", err)
				continue
			}
			if response != nil && len(response.Messages()) != 0 {
				for _, message := range response.Messages() {
					cli.logger.Info("receive message: ", message.Text)
					if message.ContentType == protobuf.ChatMessage_CONTACT_REQUEST {
						msgCh <- message.ID
					}
					if message.ContentType == protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED {
						toContacts := cli.messenger.MutualContacts()
						cli.logger.Info("contacts number: ", len(toContacts))
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendMessageLoop(cli *StatusCLI, tick time.Duration, ctx context.Context, wg *sync.WaitGroup, sem chan struct{}, cancel context.CancelFunc) {
	defer wg.Done()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ticker.C:
			if len(cli.messenger.MutualContacts()) == 0 {
				continue
			}
			sem <- struct{}{}
			cli.logger.Info("Enter your message to send: (type 'quit' or 'q' to exit)")
			message, err := reader.ReadString('\n')
			if err != nil {
				<-sem
				cli.logger.Error("failed to read input", err)
				continue
			}

			message = strings.TrimSpace(message)
			if message == "quit" || message == "q" || strings.Contains(message, "\x03") {
				cancel()
				<-sem
				return
			}
			if message == "" {
				<-sem
				continue
			}

			err = sendDirectMessage(cli, message, ctx)
			time.Sleep(WaitingInterval)
			<-sem
			if err != nil {
				cli.logger.Error("failed to send direct message: ", err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
