package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "dm",
				Aliases: []string{"d"},
				Usage:   "Send direct message",
				Action: func(cCtx *cli.Context) error {
					fmt.Println("params: ", cCtx.Args().First())

					// TODO test sharding
					// enrBootstrap := "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im"
					enrBootstrap := "enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"

					// Start alice node and messager

					fmt.Println("[Alice] starting messager")
					alicePrivKey, err := crypto.GenerateKey()
					if err != nil {
						fmt.Println(err)
						return err
					}

					config := &wakuv2.Config{}
					config.EnableDiscV5 = true
					config.DiscV5BootstrapNodes = []string{enrBootstrap}
					config.DiscoveryLimit = 20
					aliceNode, err := wakuv2.New("", "", config, nil, nil, nil, nil, nil)
					if err != nil {
						fmt.Println(err)
						return err
					}

					aliceNode.Start()
					defer aliceNode.Stop()

					time.Sleep(3 * time.Second)

					appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
					if err != nil {
						fmt.Println(err)
						return err
					}
					walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
					if err != nil {
						return err
					}
					madb, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
					if err != nil {
						return err
					}
					acc := generator.NewAccount(alicePrivKey, nil)
					iai := acc.ToIdentifiedAccountInfo("")
					aliceOptions := []protocol.Option{
						protocol.WithCustomLogger(nil),
						protocol.WithDatabase(appDb),
						protocol.WithWalletDatabase(walletDb),
						protocol.WithMultiAccounts(madb),
						protocol.WithAccount(iai.ToMultiAccount()),
						protocol.WithDatasync(),
						protocol.WithToplevelDatabaseMigrations(),
						protocol.WithBrowserDatabase(nil),
					}
					aliceMessenger, err := protocol.NewMessenger(
						"alice-node",
						alicePrivKey,
						gethbridge.NewNodeBridge(nil, nil, aliceNode),
						uuid.New().String(),
						nil,
						aliceOptions...,
					)
					if err != nil {
						fmt.Println(err)
						return err
					}

					err = aliceMessenger.Init()
					if err != nil {
						fmt.Println(err)
						return err
					}

					aliceMessenger.Start()
					defer aliceMessenger.Shutdown()

					aliceId := types.EncodeHex(crypto.FromECDSAPub(aliceMessenger.IdentityPublicKey()))
					fmt.Println("[Alice] messenger started, id:", aliceId)

					time.Sleep(3 * time.Second)

					// Start bob node and messenger

					fmt.Println("[Bob] starting messenger")
					bobPrivKey, err := crypto.GenerateKey()
					if err != nil {
						fmt.Println(err)
						return err
					}

					bobConfig := &wakuv2.Config{}
					bobConfig.EnableDiscV5 = true
					bobConfig.DiscV5BootstrapNodes = []string{enrBootstrap}
					bobConfig.DiscoveryLimit = 20
					bobNode, err := wakuv2.New("", "", bobConfig, nil, nil, nil, nil, nil)
					bobNode.Start()
					defer bobNode.Stop()
					time.Sleep(3 * time.Second)
					if err != nil {
						fmt.Println(err)
						return err
					}

					appDb2, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
					if err != nil {
						fmt.Println(err)
						return err
					}
					walletDb2, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
					if err != nil {
						return err
					}
					madb2, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
					if err != nil {
						return err
					}
					acc2 := generator.NewAccount(bobPrivKey, nil)
					iai2 := acc2.ToIdentifiedAccountInfo("")
					bobOptions := []protocol.Option{
						protocol.WithCustomLogger(nil),
						protocol.WithDatabase(appDb2),
						protocol.WithWalletDatabase(walletDb2),
						protocol.WithMultiAccounts(madb2),
						protocol.WithAccount(iai2.ToMultiAccount()),
						protocol.WithDatasync(),
						protocol.WithToplevelDatabaseMigrations(),
						protocol.WithBrowserDatabase(nil),
					}
					bobMessenger, err := protocol.NewMessenger(
						"bob-node",
						bobPrivKey,
						gethbridge.NewNodeBridge(nil, nil, bobNode),
						uuid.New().String(),
						nil,
						bobOptions...,
					)
					if err != nil {
						fmt.Println(err)
						return err
					}

					err = bobMessenger.Init()
					if err != nil {
						fmt.Println(err)
						return err
					}

					bobMessenger.Start()
					defer bobMessenger.Shutdown()

					bobId := types.EncodeHex(crypto.FromECDSAPub(bobMessenger.IdentityPublicKey()))
					fmt.Println("[Bob] messenger started, id:", bobId)

					time.Sleep(3 * time.Second)

					// Send contact request from Alice to Bob

					fmt.Println("[Alice] send contact request to bob, contact id:", bobId)
					request := &requests.SendContactRequest{
						ID:      bobId,
						Message: "Hello!",
					}
					respSendCR, err := aliceMessenger.SendContactRequest(context.Background(), request)
					fmt.Println("[Alice] function SendContactRequest response: ", respSendCR)
					if err != nil {
						fmt.Println(err)
						return err
					}

					respReceiveCR, err := protocol.WaitOnMessengerResponse(
						bobMessenger,
						func(r *protocol.MessengerResponse) bool {
							return len(r.Contacts) == 1 && len(r.Messages()) >= 2
						},
						"[Bob] no contact request from alice",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}

					msg := protocol.FindFirstByContentType(respReceiveCR.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
					fmt.Println("[Bob] receive contact request response: ", msg.Text)

					// Bob accept contact request

					fmt.Println("[Bob] send contact request acceptance to alice")
					respAccCR, err := bobMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(msg.ID)})
					fmt.Println("[Bob] function AcceptContactRequest response: ", respAccCR)
					if err != nil {
						fmt.Println(err)
						return err
					}

					bobContacts := bobMessenger.MutualContacts()
					fmt.Println("[Bob] contacts number:", len(bobContacts))

					accRespAlice, err := protocol.WaitOnMessengerResponse(aliceMessenger,
						func(r *protocol.MessengerResponse) bool {
							return len(r.Contacts) == 1 && len(r.Messages()) >= 2
						},
						"[Alice] contact request acceptance not received from bob",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}
					accMsg := protocol.FindFirstByContentType(accRespAlice.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED)
					fmt.Println("[Alice] got message:", accMsg.Text)

					aliceContacts := aliceMessenger.MutualContacts()
					fmt.Println("[Alice] contacts number:", len(aliceContacts))

					// Send DM from alice to bob

					aliceChat := aliceMessenger.Chat(aliceContacts[0].ID)
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

					bobChat := bobMessenger.Chat(bobContacts[0].ID)
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
