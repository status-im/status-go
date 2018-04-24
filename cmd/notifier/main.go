package main

import (
	"flag"
	"log"
	"os"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
)

var (
	clusterConfigFile   = flag.String("clusterconfig", "", "Cluster configuration file")
	prodMode            = flag.Bool("production", false, "Whether production settings should be loaded")
	dataDir             = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID           = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby, 777=StatusChain)")
	pushNotificationURI = flag.String("uri", "localhost:3000", "Data directory for the databases and keystore")
)

func main() {
	var n *Notifier
	var node *node.StatusNode

	address := *pushNotificationURI
	if n = New(address); n == nil {
		panic("Couldn't connect to push notification server on " + address)
	}
	defer n.Close()

	if node = statusNode(); node == nil {
		panic("Couldn't setup the node")
	}

	/*
		// TODO(adriacidre) : Subscribe to a specific channel
			var w *whisper.Whisper
			w, err := b.StatusNode().WhisperService()
			if err != nil {
				panic("Couldn't get an instance of whisper service")
			}
	*/

	// TODO(adriacidre) : Remove this example on how to send a notification
	if err := n.Send([]string{os.Getenv("ANDROID_FCM_TOKEN")}, "Hey there!"); err != nil {
		log.Fatalf("An error occured: %v", err)
	}

	if node.GethNode() != nil {
		node.GethNode().Wait()
	}
}
