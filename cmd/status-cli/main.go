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

					enrBootstrap := "enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"
					// enrBootstrap := "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im"

					// start alice node and messager
					fmt.Println("starting alice messager")
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

					// backend := api.NewGethStatusBackend()
					// fmt.Println("============: ", backend.StatusNode().WakuV2Service())

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
					fmt.Println("alice messenger started, id:", aliceId)

					time.Sleep(3 * time.Second)

					// start bob node and messenger
					fmt.Println("starting bob messenger")
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
					fmt.Println("bob messenger started, id:", bobId)

					time.Sleep(3 * time.Second)

					// pkString := hex.EncodeToString(crypto.FromECDSAPub(&recipientKey.PublicKey))
					// chat := protocol.CreateOneToOneChat(pkString, &recipientKey.PublicKey, aliceMessenger.GetTransport())
					// fmt.Println(chat)

					// send contact request
					fmt.Println("send contact request from alice to bob")
					fmt.Println("contact id:", bobId)
					request := &requests.SendContactRequest{
						ID:      bobId,
						Message: "hello!",
					}
					respSendCR, err := aliceMessenger.SendContactRequest(context.Background(), request)
					fmt.Println("SendContactRequest response: ", respSendCR)
					if err != nil {
						fmt.Println(err)
						return err
					}

					// time.Sleep(3 * time.Second)

					respReceiveCR, err := protocol.WaitOnMessengerResponse(
						bobMessenger,
						func(r *protocol.MessengerResponse) bool {
							return len(r.Contacts) == 1 && len(r.Messages()) >= 2
						},
						"bob: no contact request from alice",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}

					msg := protocol.FindFirstByContentType(respReceiveCR.Messages(), protobuf.ChatMessage_CONTACT_REQUEST)
					fmt.Println("Receive Contact Request response: ", msg.Text)

					// accept contact request
					fmt.Println("accept contact request from bob to alice")
					respAccCR, err := bobMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(msg.ID)})
					fmt.Println("AcceptContactRequest response: ", respAccCR)
					if err != nil {
						fmt.Println(err)
						return err
					}

					bobContacts := bobMessenger.MutualContacts()
					fmt.Println("bob has contacts:", len(bobContacts))

					accRespAlice, err := protocol.WaitOnMessengerResponse(aliceMessenger,
						func(r *protocol.MessengerResponse) bool {
							return len(r.Contacts) == 1 && len(r.Messages()) >= 2
						},
						"alice: contact request acceptance not received from bob",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}
					accMsg := protocol.FindFirstByContentType(accRespAlice.Messages(), protobuf.ChatMessage_SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED)
					fmt.Println("alice got message:", accMsg.Text)

					aliceContacts := aliceMessenger.MutualContacts()
					fmt.Println("alice has contacts:", len(aliceContacts))

					// send dm from alice to bob
					// myID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
					aliceChat := aliceMessenger.Chat(aliceContacts[0].ID)
					fmt.Println("alice chat with contact id:", aliceChat.ID)
					fmt.Println("alice chat with contact id:", aliceContacts[0].ID)
					clock, timestamp := aliceChat.NextClockAndTimestamp(aliceMessenger.GetTransport())
					inputMessage := common.NewMessage()
					inputMessage.ChatId = aliceChat.ID
					// inputMessage.LocalChatID = aliceChat.ID
					inputMessage.Text = "hello bob!"
					inputMessage.Clock = clock
					inputMessage.Timestamp = timestamp
					chatResp, err := aliceMessenger.SendChatMessage(context.Background(), inputMessage)
					fmt.Println("alice SendChatMessage response:", chatResp)
					fmt.Println("alice SendChatMessage messages:", chatResp.Messages())
					fmt.Println("alice SendChatMessage message text:", chatResp.Messages()[0].Text)
					if err != nil {
						fmt.Println(err)
						return err
					}

					// bobChat := bobMessenger.Chat(bobContacts[0].ID)
					// fmt.Println("bob chat id:", bobChat.ID)

					// bobChats := bobMessenger.Chats()
					// for _, c := range bobChats {
					// 	fmt.Println("bob chats -> chat id:", c.ID)
					// }

					chatRespBob, err := protocol.WaitOnMessengerResponse(
						bobMessenger,
						func(r *protocol.MessengerResponse) bool { return len(r.Messages()) > 0 },
						"bob: not receive message from alice",
					)
					if err != nil {
						fmt.Println(err)
						return err
					}
					fmt.Println("bob receive message:", chatRespBob.Chats()[0].LastMessage.Text)
					fmt.Println("bob receive message chats:", len(chatRespBob.Chats()))

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
