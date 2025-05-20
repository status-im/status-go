package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"math/rand"
	"os"
	"path"
	"time"

	"go.uber.org/zap"

	abi_spec "github.com/status-im/status-go/abi-spec"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	wakuextn "github.com/status-im/status-go/services/wakuv2ext"
)

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix()) * 1000
}

var (
	seedPhrase     = flag.String("seed-phrase", "", "Seed phrase")
	nAddedContacts = flag.Int("added-contacts", 100, "Number of added contacts to create")
	nContacts      = flag.Int("contacts", 100, "Number of contacts to create")
	nPublicChats   = flag.Int("public-chats", 5, "Number of public chats")
	nCommunities   = flag.Int("communities", 5, "Number of communities")
	nMessages      = flag.Int("number-of-messages", 0, "Number of messages for each chat")
	nOneToOneChats = flag.Int("one-to-one-chats", 5, "Number of one to one chats")

	logLevel      = flag.String("log", "DEBUG", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	fetchBackup   = flag.Bool("fetch-backup", false, "Fetch backup")
	proxyUser     = flag.String("proxy-user", os.Getenv("STATUS_BUILD_PROXY_USER"), "Proxy user, defaults to STATUS_BUILD_PROXY_USER env var")
	proxyPassword = flag.String("proxy-password", os.Getenv("STATUS_BUILD_PROXY_PASSWORD"), "Proxy password, defaults to STATUS_BUILD_PROXY_PASSWORD env var")
	dataDir       = flag.String("root-data-dir", getDefaultDataDir(), "Root data directory, use current directory if not specified")
	password      = flag.String("password", "1234567890", "password to encrypt db")
)

func main() {
	if err := logutils.OverrideRootLoggerWithConfig(logutils.LogSettings{
		Enabled:   true,
		Level:     "ERROR",
		Colorized: terminal.IsTerminal(int(os.Stdin.Fd())),
	}); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}

	logger := logutils.ZapLogger()

	flag.Parse()

	logDir := path.Join(*dataDir, "logs")
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		logger.Error("failed to create log directory", zap.Error(err))
		os.Exit(1)
	}

	backend := api.NewGethStatusBackend(logger)
	sha3Password := abi_spec.Sha3WithHexPrefix(*password)
	backend.UpdateRootDataDir(*dataDir)
	err = backend.OpenAccounts()
	if err != nil {
		logger.Error("failed to open accounts", zap.Error(err))
		os.Exit(1)
	}

	restoreAccountRequestTemplate := `
	{
  "customizationColor": "blue",
  "fetchBackup": %t,
  "kdfIterations": 3200,
  "logEnabled": true,
  "logFilePath": "%s",
  "logLevel": "%s",
  "mnemonic": "%s",
  "password": "%s",
  "poktToken": "3ef2018191814b7e1009b8d9",
  "rootDataDir": "%s",
  "rootKeystoreDir": "%s/keystore",
  "statusProxyBlockchainPassword": "%s",
  "statusProxyBlockchainUser": "%s",
  "statusProxyEnabled": true,
  "statusProxyMarketPassword": "%s",
  "statusProxyMarketUser": "%s",
  "statusProxyStageName": "test",
  "testNetworksEnabled": true,
  "verifyENSContractAddress": "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
  "verifyENSURL": "https://eth-archival.rpc.grove.city/v1/3ef2018191814b7e1009b8d9",
  "verifyTransactionChainID": 1,
  "verifyTransactionURL": "https://eth-archival.rpc.grove.city/v1/3ef2018191814b7e1009b8d9",
  "wakuV2EnableMissingMessageVerification": true,
  "wakuV2EnableStoreConfirmationForMessagesSent": false,
  "wakuV2LightClient": true,
  "wakuV2Nameserver": "8.8.8.8"
}`
	restoreAccountRequestJSON := fmt.Sprintf(restoreAccountRequestTemplate,
		*fetchBackup,
		logDir,
		*logLevel,
		*seedPhrase,
		sha3Password,
		*dataDir,
		*dataDir,
		*proxyPassword,
		*proxyUser,
		*proxyPassword,
		*proxyUser,
	)
	restoreAccountRequest := &requests.RestoreAccount{}
	err = json.Unmarshal([]byte(restoreAccountRequestJSON), restoreAccountRequest)
	if err != nil {
		logger.Error("failed to unmarshal restore account request", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("final restoreAccountRequest", zap.Any("restoreAccountRequest", restoreAccountRequest))

	_, err = backend.RestoreAccountAndLogin(restoreAccountRequest)
	if err != nil {
		logger.Error("failed to restore account and login", zap.Error(err))
		os.Exit(1)
	}

	wakuextservice := backend.StatusNode().WakuV2ExtService()
	if wakuextservice == nil {
		logger.Error("wakuext not available")
		return
	}

	wakuext := wakuextn.NewPublicAPI(wakuextservice)

	// This will start the push notification server as well as
	// the config is set to Enabled
	_, err = wakuext.StartMessenger()
	if err != nil {
		logger.Error("failed to start messenger", zap.Error(err))
		return
	}

	logger.Info("Creating added contacts")

	for i := 0; i < *nAddedContacts; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			logger.Error("failed generate key", zap.Error(err))
			return
		}

		keyString := common.PubkeyToHex(&key.PublicKey)
		_, err = wakuext.AddContact(context.Background(), &requests.AddContact{ID: keyString})
		if err != nil {
			logger.Error("failed Add contact", zap.Error(err))
			return
		}
	}

	logger.Info("Creating contacts")

	for i := 0; i < *nContacts; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			return
		}

		contact, err := protocol.BuildContactFromPublicKey(&key.PublicKey)
		if err != nil {
			return
		}

		_, err = wakuext.AddContact(context.Background(), &requests.AddContact{ID: contact.ID})
		if err != nil {
			return
		}
	}

	logger.Info("Creating public chats")

	for i := 0; i < *nPublicChats; i++ {
		chat := protocol.CreatePublicChat(randomString(10), &testTimeSource{})
		chat.SyncedTo = 0
		chat.SyncedFrom = 0

		err = wakuext.SaveChat(context.Background(), chat)
		if err != nil {
			return
		}

		var messages []*common.Message

		for i := 0; i < *nMessages; i++ {
			messages = append(messages, buildMessage(chat, i))

		}

		if len(messages) > 0 {
			if err := wakuext.Messenger().SaveMessages(messages); err != nil {
				return
			}
		}

	}

	logger.Info("Creating communities", zap.Int("num", *nCommunities))
	for i := 0; i < *nCommunities; i++ {
		request := requests.CreateCommunity{
			Name:        randomString(10),
			Description: randomString(30),
			Color:       "#ffffff",
			Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
		}
		_, err = wakuext.CreateCommunity(&request)
		if err != nil {
			logger.Error("failed to create community", zap.Error(err))
			return
		}
	}

	logger.Info("Creating one to one chats")

	for i := 0; i < *nOneToOneChats; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			return
		}

		keyString := common.PubkeyToHex(&key.PublicKey)
		chat := protocol.CreateOneToOneChat(keyString, &key.PublicKey, &testTimeSource{})
		chat.SyncedTo = 0
		chat.SyncedFrom = 0
		err = wakuext.SaveChat(context.Background(), chat)
		if err != nil {
			return
		}
		var messages []*common.Message

		for i := 0; i < *nMessages; i++ {
			messages = append(messages, buildMessage(chat, i))

		}

		if len(messages) > 0 {
			if err := wakuext.Messenger().SaveMessages(messages); err != nil {
				return
			}
		}

	}
}

func getDefaultDataDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func buildMessage(chat *protocol.Chat, count int) *common.Message {
	key, err := crypto.GenerateKey()
	if err != nil {
		logutils.ZapLogger().Error("failed build message", zap.Error(err))
		return nil
	}

	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	clock += uint64(count)
	message := common.NewMessage()
	message.Text = fmt.Sprintf("test message %d", count)
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.From = common.PubkeyToHex(&key.PublicKey)
	data := []byte(uuid.New().String())
	message.ID = types.HexBytes(crypto.Keccak256(data)).String()
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	switch chat.ChatType {
	case protocol.ChatTypePublic, protocol.ChatTypeProfile:
		message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	case protocol.ChatTypeOneToOne:
		message.MessageType = protobuf.MessageType_ONE_TO_ONE
	case protocol.ChatTypePrivateGroupChat:
		message.MessageType = protobuf.MessageType_PRIVATE_GROUP
	}

	_ = message.PrepareContent("")
	return message
}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] // nolint: gosec
	}
	return string(b)
}
