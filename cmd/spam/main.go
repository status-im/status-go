package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	. "github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wakuext"
	"golang.org/x/crypto/ssh/terminal"
	stdlog "log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// All general log messages in this package should be routed through this logger.
var spamLogger = log.New("package", "status-go/cmd/spam")

var backend = api.NewGethStatusBackend()

const PATH_WALLET_ROOT = "m/44'/60'/0'/0"

// EIP1581 Root Key, the extended key from which any whisper key/encryption key can be derived
const PATH_EIP_1581 = "m/43'/60'/1581'"

// BIP44-0 Wallet key, the default wallet key
const PATH_DEFAULT_WALLET = PATH_WALLET_ROOT + "/0"

// EIP1581 Chat Key 0, the default whisper key
const PATH_WHISPER = PATH_EIP_1581 + "/0'/0"

const DEFAULT_NETWORKS = `
[
  {
    "id": "testnet_rpc",
    "etherscan-link": "https://ropsten.etherscan.io/address/",
    "name": "Ropsten with upstream RPC",
    "config": {
      "NetworkId": 3,
      "DataDir": "/ethereum/testnet_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://ropsten.infura.io/v3/f315575765b14720b32382a61a89341a"
      }
    }
  },
  {
    "id": "rinkeby_rpc",
    "etherscan-link": "https://rinkeby.etherscan.io/address/",
    "name": "Rinkeby with upstream RPC",
    "config": {
      "NetworkId": 4,
      "DataDir": "/ethereum/rinkeby_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://rinkeby.infura.io/v3/f315575765b14720b32382a61a89341a"
      }
    }
  },
  {
    "id": "goerli_rpc",
    "etherscan-link": "https://goerli.etherscan.io/address/",
    "name": "Goerli with upstream RPC",
    "config": {
      "NetworkId": 5,
      "DataDir": "/ethereum/goerli_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://goerli.blockscout.com/"
      }
    }
  },
  {
    "id": "mainnet_rpc",
    "etherscan-link": "https://etherscan.io/address/",
    "name": "Mainnet with upstream RPC",
    "config": {
      "NetworkId": 1,
      "DataDir": "/ethereum/mainnet_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://mainnet.infura.io/v3/f315575765b14720b32382a61a89341a"
      }
    }
  },
  {
    "id": "xdai_rpc",
    "name": "xDai Chain",
    "config": {
      "NetworkId": 100,
      "DataDir": "/ethereum/xdai_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://dai.poa.network"
      }
    }
  },
  {
    "id": "poa_rpc",
    "name": "POA Network",
    "config": {
      "NetworkId": 99,
      "DataDir": "/ethereum/poa_rpc",
      "UpstreamConfig": {
        "Enabled": true,
        "URL": "https://core.poa.network"
      }
    }
  }
]
`

var spamMessage = flag.String("m", "for the inefficient of status team>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>0", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
var sleepSecond = flag.Int("s", 30, "-s 30")
var times = flag.Int("t", 1, "-6 1")

func main() {
	//for i:=0;i<10;i++{
	//	fmt.Println(rand.Float64()*10)
	//}
	flag.Parse()

	rand.Seed(time.Now().Unix())

	//fmt.Println(generateMessage())
	doSpam()
}

func generateMessage() string {
	messages := []string{"Oi", "Hi", "Hello", "Hii", "Hola", "Hello everyone!!\nHow are you ??",
		"Ou", "First time here", "Hello guys", "hi...", "Hai friends", "Hai", "SNT -> 20$", "O",
		"✋", "✋🏻", "🖐🏿", "👋🏻", "👄", "💋", "👀", "👁", "👣", "👧", "🤜", "Hi there!", "Sup sup", "Hey!", "Bom dia"}

	message := messages[int(rand.Float64()*float64(len(messages)))]
	//length := int(rand.Float64()*100)
	//for i:=0; i<length;i++{
	//	message = message+" "+ alias.Generate2(uint64(rand.Float64() * 0x7fffffffffffffff))
	//}

	return message
}

func doSpam() {
	nodeConfig, err := params.NewNodeConfigWithDefaultsAndFiles(
		getDefaultDataDir(),
		uint64(1),
		[]params.Option{},
		[]string{"/Users/Franklyn/development/project/from_github/status-go/cmd/spam/waku.json"},
	)
	if err != nil {
		spamLogger.Error(err.Error())
		os.Exit(1)
	}
	installationID := uuid.New().String()
	nodeConfig.ShhextConfig.InstallationID = installationID

	setupLogging(nodeConfig)

	err = backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir)
	if err != nil {
		spamLogger.Error("Failed to init keystore", "error", err)
		return
	}

	//err = backend.StartNode(nodeConfig)
	//if err != nil {
	//	spamLogger.Error("Failed to start node", "error", err)
	//	return
	//}

	fmt.Printf("data dir: %s\n", getDefaultDataDir())

	const pwd = "yyyyyy"
	pathStrings := []string{PATH_WALLET_ROOT, PATH_EIP_1581, PATH_WHISPER, PATH_DEFAULT_WALLET}
	accs, err := backend.AccountManager().
		AccountsGenerator().
		GenerateAndDeriveAddresses(12, 1, "", pathStrings)

	generateAccount := accs[0]
	_, err = backend.AccountManager().
		AccountsGenerator().
		StoreDerivedAccounts(generateAccount.ID, pwd, pathStrings)
	//fmt.Println("Mnemonic => ",generateAccount.Mnemonic)
	name, err := alias.GenerateFromPublicKeyString(generateAccount.Derived[PATH_WHISPER].PublicKey)
	photoPath, err := Identicon(generateAccount.Derived[PATH_WHISPER].PublicKey)

	//networks := json.RawMessage("{}")
	networks := json.RawMessage(DEFAULT_NETWORKS)
	logLevel := "INFO"

	settings := accounts.Settings{
		Mnemonic:          &generateAccount.Mnemonic,
		PublicKey:         generateAccount.Derived[PATH_WHISPER].PublicKey,
		Name:              name,
		Address:           types.HexToAddress(generateAccount.Address),
		EIP1581Address:    types.HexToAddress(generateAccount.Derived[PATH_EIP_1581].Address),
		DappsAddress:      types.HexToAddress(generateAccount.Derived[PATH_DEFAULT_WALLET].Address),
		WalletRootAddress: types.HexToAddress(generateAccount.Derived[PATH_WALLET_ROOT].Address),
		PreviewPrivacy:    true,
		SigningPhrase:     "",
		LogLevel:          &logLevel,
		LatestDerivedPath: 0,
		KeyUID:            generateAccount.KeyUID,
		Networks:          &networks,
		Currency:          "USD",
		PhotoPath:         photoPath,
		WakuEnabled:       true,
		Appearance:        0,
		CurrentNetwork:    "mainnet_rpc",
		InstallationID:    installationID,
	}

	backend.UpdateRootDataDir("/Users/Franklyn/development/project/from_github/status-go/cmd/spam")
	backend.OpenAccounts()
	//backend.SaveAccount(multiAccount)
	//修复DefaultPushNotificationsServers不能序列化存储到db的问题
	nodeConfig.ShhextConfig.DefaultPushNotificationsServers = []*ecdsa.PublicKey{}

	//subAccountInfo, _, err := backend.AccountManager().CreateAccount(pwd)
	//if err != nil {
	//	spamLogger.Error("Failed to create account", "error", err)
	//	return
	//}
	//fmt.Printf("created account. chat address: %s, chat pubkey: %s\n", subAccountInfo.ChatAddress, subAccountInfo.ChatPubKey)

	multiAccount := multiaccounts.Account{
		Name:           name,
		Timestamp:      time.Now().Unix(),
		PhotoPath:      photoPath,
		KeycardPairing: "pairing",
		KeyUID:         generateAccount.KeyUID,
	}

	subAccounts := []accounts.Account{
		{
			PublicKey: types.Hex2Bytes(generateAccount.Derived[PATH_DEFAULT_WALLET].PublicKey),
			Address:   types.HexToAddress(generateAccount.Derived[PATH_DEFAULT_WALLET].Address),
			Color:     "#4360df",
			Wallet:    true,
			Path:      PATH_DEFAULT_WALLET,
			Name:      "Status account",
		},
		{
			PublicKey: types.Hex2Bytes(generateAccount.Derived[PATH_WHISPER].PublicKey),
			Address:   types.HexToAddress(generateAccount.Derived[PATH_WHISPER].Address),
			Name:      name,
			Path:      PATH_WHISPER,
			Chat:      true,
		},
	}
	err = backend.StartNodeWithAccountAndConfig(multiAccount, pwd, settings, nodeConfig, subAccounts)
	if err != nil {
		spamLogger.Error("Failed to StartNodeWithAccount", "error", err)
		return
	}

	/*err = backend.StartWallet()
	if err != nil {
		spamLogger.Error("Failed to start wallet", "error", err)
		return
	}*/

	//loginParams := account.LoginParams{
	//	ChatAddress: types.HexToAddress(generateAccountInfo.ChatAddress),
	//	MainAccount: types.HexToAddress(generateAccountInfo.WalletAddress),
	//	Password:    pwd,
	//}
	//err = backend.SelectAccount(loginParams)
	//if err != nil {
	//	spamLogger.Error("Failed to select account", "error", err)
	//	return
	//}

	//resp,err := backend.CallPrivateRPC(`{"method":"startMessenger","jsonrpc":"2.0"}`)
	//if err != nil {
	//	spamLogger.Error("Failed to startMessenger", "error", err)
	//	return
	//}
	//fmt.Printf("startMessenger response: %s\n",resp)

	statusNode := backend.StatusNode()
	st, err := statusNode.WakuExtService()
	if err != nil {
		spamLogger.Error("Failed to get WakuExtService", "error", err)
		return
	}

	err = st.StartMessenger()
	if err != nil {
		spamLogger.Error("Failed to StartMessenger", "error", err)
		return
	}

	api := st.APIs()[0].Service.(*wakuext.PublicAPI)
	//channels := []string{"introductions","status-keycard","dap-ps","status","status-chinese","statusphere","support","crypto","markets","defi","status-townhall-questions","status-korean","indonesian","status-naija","status-filipino","status-indian","status-spanish","status-turkish","status-brasil","status-russian","chitchat","status-japanese","tech","status-marketing","nimbus-general","embark"}
	//channels := []string{"status-chinese"}
	channels := []string{"status", "status-chinese", "crypto", "support", "markets"}
	chats := make([]Chat, len(channels))
	j := int(rand.Float64() * float64(len(channels)))
	//for i := 0; i < len(chats); i++ {
	for i := 0; i < 1; i++ {
		//chats[i] = CreatePublicChat(channels[i], &testTimeSource{})
		chats[j] = CreatePublicChat(channels[j], &testTimeSource{})
		//err = api.SaveChat(context.TODO(), &chats[i])
		err = api.SaveChat(context.TODO(), &chats[j])
		if err != nil {
			spamLogger.Error("Failed to SaveChat", "error", err)
			return
		}

		//err = api.Join(chats[i])
		//if err != nil {
		//	spamLogger.Error("Failed to join chat", "error", err)
		//	return
		//}

		//resp, err := api.ChatMessages(chats[i].ID,"",20)
		//if err != nil {
		//	spamLogger.Error("Failed to ChatMessages", "error", err)
		//	return
		//}
		//fmt.Println("ChatMessages response =>",resp)
		//chat := chats[i]
		chat := chats[j]
		for i := 0; i < *times; i++ {
			_, err := api.SendChatMessage(context.TODO(), buildTestMessage(chat))
			if err != nil {
				spamLogger.Error("Failed to StartMessenger", "error", err)
				return
			}
			//fmt.Println("SendChatMessage response => ",resp)
		}

		//fmt.Println("done channel:", channels[i])
		fmt.Println("done channel:", channels[j])
	}

	//time.Sleep(time.Duration(*sleepSecond)*time.Second)
	s := int(rand.Float64() * 10)
	fmt.Printf("sleep for %d minutes.\n", s)
	time.Sleep(time.Duration(s) * time.Minute)
	//gethNode := statusNode.GethNode()
	//if gethNode != nil {
	//	// wait till node has been stopped
	//	gethNode.Wait()
	//	if err := sdnotify.Stopping(); err != nil {
	//		spamLogger.Warn("sd_notify STOPPING call failed", "error", err)
	//	}
	//}
}

type testTimeSource struct{}

func (t *testTimeSource) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func buildTestMessage(chat Chat) *Message {
	clock, timestamp := chat.NextClockAndTimestamp(&testTimeSource{})
	message := &Message{}
	//message.Text = *spamMessage
	//message.Text = fmt.Sprintf("%s | %.16f", *spamMessage, rand.Float64())
	message.Text = generateMessage()
	message.ChatId = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.WhisperTimestamp = clock
	message.LocalChatID = chat.ID
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	switch chat.ChatType {
	case ChatTypePublic:
		message.MessageType = protobuf.MessageType_PUBLIC_GROUP
	case ChatTypeOneToOne:
		message.MessageType = protobuf.MessageType_ONE_TO_ONE
	case ChatTypePrivateGroupChat:
		message.MessageType = protobuf.MessageType_PRIVATE_GROUP
	}

	return message
}

func getDefaultDataDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".status-spam-data")
	}
	return "./status-spam-data"
}

func setupLogging(config *params.NodeConfig) {

	colors := terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLogWithConfig(config, colors); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}
}
