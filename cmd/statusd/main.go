package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
	app        = makeApp(gitCommit)
	statusAPI  = api.NewStatusAPI()
)

var (
	// ProdModeFlag is whether we need dev or production settings
	ProdModeFlag = cli.BoolFlag{
		Name:  "production",
		Usage: "Whether production settings should be loaded",
	}

	// NodeKeyFileFlag is a node key file to be used as node's private key
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file (private key)",
	}

	// DataDirFlag defines data directory for the node
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: params.DataDir,
	}

	// NetworkIDFlag defines network ID
	NetworkIDFlag = cli.IntFlag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby)",
		Value: params.RopstenNetworkID,
	}

	// WhisperEnabledFlag flags whether Whisper is enabled or not
	WhisperEnabledFlag = cli.BoolFlag{
		Name:  "shh",
		Usage: "SHH protocol enabled",
	}

	// SwarmEnabledFlag flags whether Swarm is enabled or not
	SwarmEnabledFlag = cli.BoolFlag{
		Name:  "swarm",
		Usage: "Swarm protocol enabled",
	}

	// HTTPEnabledFlag defines whether HTTP RPC endpoint should be opened or not
	HTTPEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "HTTP RPC enpoint enabled (default: false)",
	}

	// HTTPPortFlag defines HTTP RPC port to use (if HTTP RPC is enabled)
	HTTPPortFlag = cli.IntFlag{
		Name:  "httpport",
		Usage: "HTTP RPC server's listening port",
		Value: params.HTTPPort,
	}

	// IPCEnabledFlag flags whether IPC is enabled or not
	IPCEnabledFlag = cli.BoolFlag{
		Name:  "ipc",
		Usage: "IPC RPC enpoint enabled",
	}

	// LogLevelFlag defines a log reporting level
	LogLevelFlag = cli.StringFlag{
		Name:  "log",
		Usage: `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`,
		Value: "",
	}

	// LogFileFlag defines a log filename
	LogFileFlag = cli.StringFlag{
		Name:  "logfile",
		Usage: `Path to the log file`,
		Value: "",
	}
)

func init() {
	// setup the app
	app.Action = cli.ShowAppHelp
	app.HideVersion = true // separate command prints version
	app.Commands = []cli.Command{
		versionCommand,
		lesCommand,
		wnodeCommand,
	}
	app.Flags = []cli.Flag{
		ProdModeFlag,
		NodeKeyFileFlag,
		DataDirFlag,
		NetworkIDFlag,
		LogLevelFlag,
		LogFileFlag,
	}
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makeApp creates an app with sane defaults.
func makeApp(gitCommit string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = params.Version
	if gitCommit != "" {
		app.Version += "-" + gitCommit[:8]
	}
	app.Usage = "CLI for Status nodes management"
	return app
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
