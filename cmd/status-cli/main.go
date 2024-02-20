package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol"
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

					// start alice node and messager
					fmt.Println("start alice messager")
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
					fmt.Println(aliceNode)
					if err != nil {
						fmt.Println(err)
						return err
					}

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

					// start bob node and messager
					fmt.Println("start bob messager")
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

					// pkString := hex.EncodeToString(crypto.FromECDSAPub(&recipientKey.PublicKey))
					// chat := protocol.CreateOneToOneChat(pkString, &recipientKey.PublicKey, aliceMessenger.GetTransport())
					// fmt.Println(chat)

					// send contact request
					fmt.Println("send contact request from alice to bob")
					contactId := types.EncodeHex(crypto.FromECDSAPub(bobMessenger.IdentityPublicKey()))
					request := &requests.SendContactRequest{
						ID:      contactId,
						Message: "hello!",
					}
					resp, err := aliceMessenger.SendContactRequest(context.Background(), request)
					fmt.Println("resp", resp)
					if err != nil {
						fmt.Println(err)
						return err
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
