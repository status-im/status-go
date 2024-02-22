package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/google/uuid"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
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

const enrBootstrap = "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im"

type StatusCli struct {
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
				Action: func(cCtx *cli.Context) error {
					fmt.Println("params: ", cCtx.Args().First())

					// Start Alice and Bob's messengers
					alice, err := startMessenger("Alice")
					if err != nil {
						fmt.Println(err)
						return err
					}
					defer stopMessenger(alice)

					bob, err := startMessenger("Bob")
					if err != nil {
						fmt.Println(err)
						return err
					}
					defer stopMessenger(bob)

					aliceMessenger := alice.messenger
					bobMessenger := bob.messenger

					// Send contact request from Alice to Bob, bob accept the request
					msgID, err := sendContactRequest(alice, bob)
					if err != nil {
						fmt.Println(err)
						return err
					}

					err = sendContactRequestAcceptance(bob, alice, msgID)
					if err != nil {
						fmt.Println(err)
						return err
					}

					// Send DM from alice to bob
					aliceChat := aliceMessenger.Chat(alice.messenger.MutualContacts()[0].ID)
					fmt.Println("[Alice] chat with contact id:", aliceChat.ID)

					clock, timestamp := aliceChat.NextClockAndTimestamp(aliceMessenger.GetTransport())
					inputMessage := common.NewMessage()
					inputMessage.ChatId = aliceChat.ID
					inputMessage.LocalChatID = aliceChat.ID
					inputMessage.Clock = clock
					inputMessage.Timestamp = timestamp
					inputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
					inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
					inputMessage.Text = "Hello Bob!"

					chatResp, err := aliceMessenger.SendChatMessage(context.Background(), inputMessage)
					fmt.Println("[Alice] function SendChatMessage response.messages:", chatResp.Messages())
					if err != nil {
						fmt.Println(err)
						return err
					}

					chatRespBob, err := protocol.WaitOnMessengerResponse(
						bobMessenger,
						func(r *protocol.MessengerResponse) bool { return len(r.Messages()) > 0 },
						"[Bob] not receive message from alice",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}
					fmt.Println("[Bob] receive message from alice:", chatRespBob.Chats()[0].LastMessage.Text)

					// Send DM from bob to alice
					bobChat := bobMessenger.Chat(bob.messenger.MutualContacts()[0].ID)
					fmt.Println("[Bob] chat with contact id:", bobChat.ID)

					bobClock, bobTimestamp := bobChat.NextClockAndTimestamp(bobMessenger.GetTransport())
					bobInputMessage := common.NewMessage()
					bobInputMessage.ChatId = bobChat.ID
					bobInputMessage.LocalChatID = bobChat.ID
					bobInputMessage.Clock = bobClock
					bobInputMessage.Timestamp = bobTimestamp
					bobInputMessage.MessageType = protobuf.MessageType_ONE_TO_ONE
					bobInputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
					bobInputMessage.Text = "Hello Alice!"

					bobSendChatResp, err := bobMessenger.SendChatMessage(context.Background(), bobInputMessage)
					fmt.Println("[Bob] function SendChatMessage response.messages:", bobSendChatResp.Messages())
					if err != nil {
						fmt.Println(err)
						return err
					}

					aliceReceiveChatResp, err := protocol.WaitOnMessengerResponse(
						aliceMessenger,
						func(r *protocol.MessengerResponse) bool { return len(r.Messages()) > 0 },
						"[Alice] not receive message from bob",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}
					fmt.Println("[Alice] receive message from bob:", aliceReceiveChatResp.Chats()[0].LastMessage.Text)

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startMessenger(name string) (*StatusCli, error) {
	fmt.Printf("[%s] starting messager\n", name)
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := &wakuv2.Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = []string{enrBootstrap}
	config.DiscoveryLimit = 20
	node, err := wakuv2.New("", "", config, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	err = node.Start()
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

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
		protocol.WithCustomLogger(nil),
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
	fmt.Printf("[%s] messenger started, id: %s\n", name, id)

	time.Sleep(3 * time.Second)

	data := StatusCli{
		name:      name,
		messenger: messenger,
		waku:      node,
	}

	return &data, nil
}

func stopMessenger(cli *StatusCli) {
	err := cli.messenger.Shutdown()
	if err != nil {
		fmt.Println(err)
	}

	err = cli.waku.Stop()
	if err != nil {
		fmt.Println(err)
	}
}

func sendContactRequest(from, to *StatusCli) (string, error) {
	destID := types.EncodeHex(crypto.FromECDSAPub(to.messenger.IdentityPublicKey()))
	fmt.Printf("[%s] send contact request to %s, contact id: %s\n", from.name, to.name, destID)
	request := &requests.SendContactRequest{
		ID:      destID,
		Message: "Hello!",
	}
	resp, err := from.messenger.SendContactRequest(context.Background(), request)
	fmt.Printf("[%s] function SendContactRequest response.messages: %s\n", from.name, resp.Messages())
	if err != nil {
		return "", err
	}

	receiverResp, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("[%s] no contact request from %s", to.name, from.name),
	)
	if err != nil {
		return "", err
	}

	msg := protocol.FindFirstByContentType(receiverResp.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
	fmt.Printf("[%s] receive contact request response: %s\n", to.name, msg.Text)

	return msg.ID, nil
}

func sendContactRequestAcceptance(from, to *StatusCli, msgID string) error {
	fmt.Printf("[%s] send contact request acceptance to %s\n", from.name, to.name)
	resp, err := from.messenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(msgID)})
	if err != nil {
		return err
	}
	fmt.Printf("[%s] function AcceptContactRequest response: %v\n", from.name, resp)

	fromContacts := from.messenger.MutualContacts()
	fmt.Printf("[%s] contacts number: %d\n", from.name, len(fromContacts))

	respTo, err := protocol.WaitOnMessengerResponse(
		to.messenger,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) >= 2
		},
		fmt.Sprintf("[%s] contact request acceptance not received from bob", to.name),
	)
	if err != nil {
		return err
	}

	msg := protocol.FindFirstByContentType(respTo.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED)
	fmt.Printf("[%s] got message: %s\n", to.name, msg.Text)

	toContacts := to.messenger.MutualContacts()
	fmt.Printf("[%s] contacts number: %d\n", to.name, len(toContacts))

	return nil
}
