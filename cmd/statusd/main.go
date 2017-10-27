package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
	statusAPI  = api.NewStatusAPI()
)

var (
	profMode       = flag.Bool("production", false, "Whether production settings should be loaded")
	nodeKeyFile    = flag.String("nodekey", "", "P2P node key file (private key)")
	dataDir        = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID      = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby)")
	whisperEnabled = flag.Bool("shh", false, "SHH protocol enabled")
	swarmEnabled   = flag.Bool("swarm", false, "Swarm protocol enabled")
	httpEnabled    = flag.Bool("http", false, "HTTP RPC enpoint enabled (default: false)")
	httpPort       = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled     = flag.Bool("ipc", false, "IPC RPC enpoint enabled")
	logLevel       = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile        = flag.String("logfile", "", "Path to the log file")
)

func main() {
	flag.Parse()
}

// version returns string representing binary version plus
// git comming hash, if present.
func version(gitCommit string) string {
	version = params.Version
	if gitCommit != "" {
		version += "-" + gitCommit[:8]
	}
	return version
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig(ctx *cli.Context) (*params.NodeConfig, error) {
	nodeConfig, err := params.NewNodeConfig(
		ctx.GlobalString(DataDirFlag.Name),
		ctx.GlobalUint64(NetworkIDFlag.Name),
		!ctx.GlobalBool(ProdModeFlag.Name))
	if err != nil {
		return nil, err
	}

	nodeConfig.NodeKeyFile = ctx.GlobalString(NodeKeyFileFlag.Name)

	if logLevel := ctx.GlobalString(LogLevelFlag.Name); logLevel != "" {
		nodeConfig.LogLevel = logLevel
	}
	if logFile := ctx.GlobalString(LogFileFlag.Name); logFile != "" {
		nodeConfig.LogFile = logFile
	}

	return nodeConfig, nil
}

// printNodeConfig prints node config
func printNodeConfig(ctx *cli.Context) {
	nodeConfig, err := makeNodeConfig(ctx)
	if err != nil {
		fmt.Printf("Loaded Config: failed (err: %v)", err)
		return
	}
	nodeConfig.LightEthConfig.Genesis = "SKIP"
	fmt.Println("Loaded Config: ", nodeConfig)
}
