package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
)

var (
	prodMode    = flag.Bool("production", false, "Whether production settings should be loaded")
	nodeKeyFile = flag.String("nodekey", "", "P2P node key file (private key)")
	dataDir     = flag.String("datadir", "wnode-status-data", "Data directory for the databases and keystore")
	networkID   = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby)")
	httpEnabled = flag.Bool("http", false, "HTTP RPC enpoint enabled (default: false)")
	httpPort    = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled  = flag.Bool("ipc", false, "IPC RPC enpoint enabled")

	// wnode specific flags
	echo           = flag.Bool("echo", true, "Echo mode, prints some arguments for diagnostics")
	bootstrap      = flag.Bool("bootstrap", true, "Don't actively connect to peers, wait for incoming connections")
	notify         = flag.Bool("notify", false, "Node is capable of sending Push Notifications")
	forward        = flag.Bool("forward", false, "Only forward messages, neither send nor decrypt messages")
	mailserver     = flag.Bool("mailserver", false, "Delivers expired messages on demand")
	identity       = flag.String("identity", "", "Protocol identity file (private key used for asymmetric encryption)")
	password       = flag.String("password", "", "Password file (password is used for symmetric encryption)")
	port           = flag.Int("port", params.WhisperPort, "Whisper node's listening port")
	pow            = flag.Float64("pow", params.WhisperMinimumPoW, "PoW for messages to be added to queue, in float format")
	ttl            = flag.Int("ttl", params.WhisperTTL, "Time to live for messages, in seconds")
	injectAccounts = flag.Bool("injectaccounts", true, "Whether test account should be injected or not")
	firebaseAuth   = flag.String("firebaseauth", "", "FCM Authorization Key used for sending Push Notifications")
)

func main() {
	flag.Parse()

	config, err := makeNodeConfig()
	if err != nil {
		log.Fatalf("Making config failed: %v", err)
		return
	}

	printHeader(config)

	if *injectAccounts {
		LoadTestAccounts(config.DataDir)
	}

	backend := api.NewStatusBackend()
	started, err := backend.StartNode(config)
	if err != nil {
		log.Fatalf("Node start failed: %v", err)
		return
	}

	// wait till node is started
	<-started

	if *injectAccounts {
		InjectTestAccounts(backend.NodeManager())
	}

	// wait till node has been stopped
	node, err := backend.NodeManager().Node()
	if err != nil {
		log.Fatalf("Getting node failed: %v", err)
		return
	}

	node.Wait()
}

// printHeader prints command header
func printHeader(config *params.NodeConfig) {
	fmt.Println("Starting Whisper/5 node..")
	if config.WhisperConfig.EchoMode {
		fmt.Printf("Whisper Config: %s\n", config.WhisperConfig)
	}
}
