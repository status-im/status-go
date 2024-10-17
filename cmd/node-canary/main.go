// node-canary tests whether a P2P peer is responding correctly.
package main

import (
	"flag"
	stdlog "log"
	"os"
	"path"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/cmd/node-canary")

var (
	staticEnodeAddr  = flag.String("staticnode", "", "checks if static node talks waku protocol (e.g. enode://abc123@1.2.3.4:30303)")
	minPow           = flag.Float64("waku.pow", params.WakuMinimumPoW, "PoW for messages to be added to queue, in float format")
	ttl              = flag.Int("waku.ttl", params.WakuTTL, "Time to live for messages, in seconds")
	homePath         = flag.String("home-dir", ".", "Home directory where state is stored")
	logLevel         = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile          = flag.String("logfile", "", "Path to the log file")
	logWithoutColors = flag.Bool("log-without-color", false, "Disables log colors")
)

func main() {
	var err error
	var staticParsedNode *enode.Node
	if *staticEnodeAddr != "" {
		staticParsedNode, err = enode.ParseV4(*staticEnodeAddr)
		if err != nil {
			logger.Crit("Invalid static address specified", "staticEnodeAddr", *staticEnodeAddr, "error", err)
			os.Exit(1)
		}
	}

	if staticParsedNode != nil {
		verifyStaticNodeBehavior(staticParsedNode)
		logger.Info("Connected to static node correctly", "address", *staticEnodeAddr)
		os.Exit(0)
	}

	logger.Crit("No address specified")
	os.Exit(1)
}

func init() {
	flag.Parse()

	colors := !(*logWithoutColors)
	if colors {
		colors = terminal.IsTerminal(int(os.Stdin.Fd()))
	}

	if err := logutils.OverrideRootLog(*logLevel != "", *logLevel, logutils.FileOptions{Filename: *logFile}, colors); err != nil {
		stdlog.Fatalf("Error initializing logger: %s", err)
	}
}

func verifyStaticNodeBehavior(staticNode *enode.Node) {
	clientBackend, err := startClientNode()
	if err != nil {
		logger.Error("Node start failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = clientBackend.StopNode() }()

	clientNode := clientBackend.StatusNode()

	// wait for peer to be added to client
	clientErrCh := helpers.WaitForPeerAsync(clientNode.Server(), *staticEnodeAddr, p2p.PeerEventTypeAdd, 5*time.Second)
	err = <-clientErrCh
	if err != nil {
		logger.Error("Error detected while waiting for static peer to be added", "error", err)
		os.Exit(1)
	}

	// wait to check if peer remains connected to client
	clientErrCh = helpers.WaitForPeerAsync(clientNode.Server(), *staticEnodeAddr, p2p.PeerEventTypeDrop, 5*time.Second)
	err = <-clientErrCh
	peers := clientNode.GethNode().Server().Peers()
	if len(peers) != 1 {
		logger.Error("Failed to add static peer", "error", err)
		os.Exit(1)
	}
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig() (*params.NodeConfig, error) {
	err := error(nil)

	workDir := ""
	if path.IsAbs(*homePath) {
		workDir = *homePath
	} else {
		workDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err == nil {
			workDir = path.Join(workDir, *homePath)
		}
	}
	if err != nil {
		return nil, err
	}

	nodeConfig, err := params.NewNodeConfigWithDefaults(path.Join(workDir, ".ethereum"), uint64(params.SepoliaNetworkID))
	if err != nil {
		return nil, err
	}

	if *logLevel != "" {
		nodeConfig.LogLevel = *logLevel
		nodeConfig.LogEnabled = true
	}

	if *logFile != "" {
		nodeConfig.LogFile = *logFile
	}

	nodeConfig.NoDiscovery = true
	nodeConfig.ListenAddr = ""
	if *staticEnodeAddr != "" {
		nodeConfig.ClusterConfig.Enabled = true
		nodeConfig.ClusterConfig.Fleet = params.FleetUndefined
		nodeConfig.ClusterConfig.StaticNodes = []string{
			*staticEnodeAddr,
		}
	}

	return wakuConfig(nodeConfig)
}

// wakuConfig creates node configuration object from flags
func wakuConfig(nodeConfig *params.NodeConfig) (*params.NodeConfig, error) {
	wakuConfig := &nodeConfig.WakuConfig
	wakuConfig.Enabled = true
	wakuConfig.LightClient = true
	wakuConfig.MinimumPoW = *minPow
	wakuConfig.TTL = *ttl

	return nodeConfig, nil
}

func startClientNode() (*api.GethStatusBackend, error) {
	config, err := makeNodeConfig()
	if err != nil {
		return nil, err
	}
	clientBackend := api.NewGethStatusBackend(logutils.ZapLogger())
	err = clientBackend.AccountManager().InitKeystore(config.KeyStoreDir)
	if err != nil {
		return nil, err
	}
	err = clientBackend.StartNode(config)
	if err != nil {
		return nil, err
	}
	return clientBackend, err
}
