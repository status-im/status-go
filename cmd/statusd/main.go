package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
	app        = makeApp(gitCommit)
)

var (
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file (private key)",
	}
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: params.DataDir,
	}
	NetworkIdFlag = cli.IntFlag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten)",
		Value: params.TestNetworkId,
	}
	LightEthEnabledFlag = cli.BoolFlag{
		Name:  "les",
		Usage: "LES protocol enabled",
	}
	WhisperEnabledFlag = cli.BoolFlag{
		Name:  "shh",
		Usage: "SHH protocol enabled",
	}
	SwarmEnabledFlag = cli.BoolFlag{
		Name:  "swarm",
		Usage: "Swarm protocol enabled",
	}
	HTTPEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "HTTP RPC enpoint enabled",
	}
	HTTPPortFlag = cli.IntFlag{
		Name:  "httpport",
		Usage: "HTTP RPC server's listening port",
		Value: params.HTTPPort,
	}
	IPCEnabledFlag = cli.BoolFlag{
		Name:  "ipc",
		Usage: "IPC RPC enpoint enabled",
	}
	LogLevelFlag = cli.StringFlag{
		Name:  "log",
		Usage: `Log level, one of: ""ERROR", "WARNING", "INFO", "DEBUG", and "TRACE"`,
		Value: "INFO",
	}
)

func init() {
	// setup the app
	app.Action = statusd
	app.HideVersion = true // separate command prints version
	app.Commands = []cli.Command{
		versionCommand,
		wnodeCommand,
	}

	app.Flags = []cli.Flag{
		NodeKeyFileFlag,
		DataDirFlag,
		NetworkIdFlag,
		LightEthEnabledFlag,
		WhisperEnabledFlag,
		SwarmEnabledFlag,
		HTTPEnabledFlag,
		HTTPPortFlag,
		IPCEnabledFlag,
		LogLevelFlag,
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

// statusd runs Status node
func statusd(ctx *cli.Context) error {
	config, err := makeNodeConfig(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not parse config: %v", err)
		return err
	}

	if err := geth.CreateAndRunNode(config); err != nil {
		return err
	}

	// wait till node has been stopped
	geth.NodeManagerInstance().Node().GethStack().Wait()

	return nil
}

// makeNodeConfig parses incoming CLI options and returns node configuration object
func makeNodeConfig(ctx *cli.Context) (*params.NodeConfig, error) {
	nodeConfig, err := params.NewNodeConfig(ctx.GlobalString(DataDirFlag.Name), ctx.GlobalInt(NetworkIdFlag.Name))
	if err != nil {
		return nil, err
	}

	nodeConfig.NodeKeyFile = ctx.GlobalString(NodeKeyFileFlag.Name)
	if !ctx.GlobalBool(HTTPEnabledFlag.Name) {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}
	nodeConfig.IPCEnabled = ctx.GlobalBool(IPCEnabledFlag.Name)
	nodeConfig.LightEthConfig.Enabled = ctx.GlobalBool(LightEthEnabledFlag.Name)
	nodeConfig.WhisperConfig.Enabled = ctx.GlobalBool(WhisperEnabledFlag.Name)
	nodeConfig.SwarmConfig.Enabled = ctx.GlobalBool(SwarmEnabledFlag.Name)
	nodeConfig.HTTPPort = ctx.GlobalInt(HTTPPortFlag.Name)

	if logLevel := ctx.GlobalString(LogLevelFlag.Name); len(logLevel) > 0 {
		nodeConfig.LogEnabled = true
		nodeConfig.LogLevel = logLevel
	}

	return nodeConfig, nil
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
	app.Usage = "Status CLI"
	return app
}
