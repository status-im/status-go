package main

import (
	"flag"
	"time"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
)

var (
	clusterConfigFile   = flag.String("clusterconfig", "", "Cluster configuration file")
	prodMode            = flag.Bool("production", false, "Whether production settings should be loaded")
	dataDir             = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID           = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby, 777=StatusChain)")
	pushNotificationURI = flag.String("uri", "localhost:3000", "Push notification uri")
	discoveryTopic      = flag.String("topic", "notifier", "Discovery topic")
)

func main() {
	var n *Notifier
	var node *node.StatusNode

	flag.Parse()

	address := *pushNotificationURI
	if n = New(address); n == nil {
		panic("Couldn't connect to push notification server on " + address)
	}
	defer n.Close()

	if node = statusNode(); node == nil {
		panic("Couldn't setup the node")
	}

	t := *discoveryTopic
	m := NewMessenger(node, t, 5*time.Second)
	if m == nil {
		panic("Error while creating the PN server")
	}
	if err := m.BroadcastAvailability(); err != nil {
		panic(err)
	}
	// TODO (adriacidre) : uncomment this
	// go m.ManageRegistrations()

	if node.GethNode() != nil {
		node.GethNode().Wait()
	}
}
