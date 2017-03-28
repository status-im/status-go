package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: params.DefaultDataDir,
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
		Value: params.DefaultHTTPPort,
	}
	IPCEnabledFlag = cli.BoolFlag{
		Name:  "ipc",
		Usage: "IPC RPC enpoint enabled",
	}
	LogLevelFlag = cli.StringFlag{
		Name:  "log",
		Usage: `Log level, one of: ""ERROR", "WARNING", "INFO", "DEBUG", and "DETAIL"`,
		Value: "INFO",
	}
	TestAccountKey = cli.StringFlag{
		Name:  "accountkey",
		Usage: "Test account PK (will be loaded into accounts cache, and injected to Whisper)",
	}
	TestAccountPasswd = cli.StringFlag{
		Name:  "accountpasswd",
		Usage: "Test account password",
	}
)

func init() {
	// setup the app
	app.Action = statusd
	app.HideVersion = true // separate command prints version
	app.Commands = []cli.Command{
		{
			Action: version,
			Name:   "version",
			Usage:  "Print app version",
		},
	}

	app.Flags = []cli.Flag{
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

// version displays app version
func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(params.DefaultClientIdentifier))
	fmt.Println("Version:", params.Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}

	fmt.Println("Network Id:", ctx.GlobalInt(NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	return nil
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
