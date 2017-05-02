package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	WhisperEchoModeFlag = cli.BoolTFlag{
		Name:  "echo",
		Usage: "Echo mode, prints some arguments for diagnostics (default: true)",
	}
	WhisperBootstrapNodeFlag = cli.BoolTFlag{
		Name:  "bootstrap",
		Usage: "Don't actively connect to peers, wait for incoming connections (default: true)",
	}
	WhisperNotificationServerNodeFlag = cli.BoolFlag{
		Name:  "notify",
		Usage: "Node is capable of sending Push Notifications",
	}
	WhisperForwarderNodeFlag = cli.BoolFlag{
		Name:  "forward",
		Usage: "Only forward messages, neither send nor decrypt messages",
	}
	WhisperMailserverNodeFlag = cli.BoolFlag{
		Name:  "mailserver",
		Usage: "Delivers expired messages on demand",
	}
	WhisperIdentityFile = cli.StringFlag{
		Name:  "identity",
		Usage: "Protocol identity file (private key used for asymetric encryption)",
	}
	WhisperPasswordFile = cli.StringFlag{
		Name:  "password",
		Usage: "Password file (password is used for symmetric encryption)",
	}
	WhisperPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Whisper node's listening port",
		Value: params.WhisperPort,
	}
	WhisperPoWFlag = cli.Float64Flag{
		Name:  "pow",
		Usage: "PoW for messages to be added to queue, in float format",
		Value: params.WhisperMinimumPoW,
	}
	WhisperTTLFlag = cli.IntFlag{
		Name:  "ttl",
		Usage: "Time to live for messages, in seconds",
		Value: params.WhisperTTL,
	}
	WhisperInjectTestAccounts = cli.BoolTFlag{
		Name:  "injectaccounts",
		Usage: "Whether test account should be injected or not (default: true)",
	}
	FirebaseAuthorizationKey = cli.StringFlag{
		Name:  "firebaseauth",
		Usage: "FCM Authorization Key used for sending Push Notifications",
	}
)

var (
	wnodeCommand = cli.Command{
		Action: wnode,
		Name:   "wnode",
		Usage:  "Starts Whisper/5 node",
		Flags: []cli.Flag{
			WhisperEchoModeFlag,
			WhisperBootstrapNodeFlag,
			WhisperNotificationServerNodeFlag,
			WhisperForwarderNodeFlag,
			WhisperMailserverNodeFlag,
			WhisperIdentityFile,
			WhisperPasswordFile,
			WhisperPoWFlag,
			WhisperPortFlag,
			WhisperTTLFlag,
			WhisperInjectTestAccounts,
			FirebaseAuthorizationKey,
		},
	}
)

// version displays app version
func wnode(ctx *cli.Context) error {
	config, err := makeWhisperNodeConfig(ctx)
	if err != nil {
		return fmt.Errorf("can not parse config: %v", err)
	}

	wnodePrintHeader(config)

	// inject test accounts
	if ctx.BoolT(WhisperInjectTestAccounts.Name) {
		geth.ImportTestAccount(filepath.Join(config.DataDir, "keystore"), "test-account1.pk")
		geth.ImportTestAccount(filepath.Join(config.DataDir, "keystore"), "test-account2.pk")
	}

	if err := geth.CreateAndRunNode(config); err != nil {
		return err
	}

	// inject test accounts into Whisper
	if ctx.BoolT(WhisperInjectTestAccounts.Name) {
		testConfig, _ := geth.LoadTestConfig()
		geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password)
	}

	// wait till node has been stopped
	geth.NodeManagerInstance().Node().GethStack().Wait()

	return nil
}

func wnodePrintHeader(nodeConfig *params.NodeConfig) {
	fmt.Println("Starting Whisper/5 node..")

	whisperConfig := nodeConfig.WhisperConfig

	if whisperConfig.EchoMode {
		fmt.Printf("Whisper Config: %s\n", whisperConfig)
	}
}

// makeWhisperNodeConfig parses incoming CLI options and returns node configuration object
func makeWhisperNodeConfig(ctx *cli.Context) (*params.NodeConfig, error) {
	nodeConfig, err := makeNodeConfig(ctx)
	if err != nil {
		return nil, err
	}

	whisperConfig := nodeConfig.WhisperConfig

	whisperConfig.Enabled = true
	whisperConfig.IdentityFile = ctx.String(WhisperIdentityFile.Name)
	whisperConfig.PasswordFile = ctx.String(WhisperPasswordFile.Name)
	whisperConfig.EchoMode = ctx.BoolT(WhisperEchoModeFlag.Name)
	whisperConfig.BootstrapNode = ctx.BoolT(WhisperBootstrapNodeFlag.Name)
	whisperConfig.ForwarderNode = ctx.Bool(WhisperForwarderNodeFlag.Name)
	whisperConfig.NotificationServerNode = ctx.Bool(WhisperNotificationServerNodeFlag.Name)
	whisperConfig.MailServerNode = ctx.Bool(WhisperMailserverNodeFlag.Name)
	whisperConfig.Port = ctx.Int(WhisperPortFlag.Name)
	whisperConfig.TTL = ctx.Int(WhisperTTLFlag.Name)
	whisperConfig.MinimumPoW = ctx.Float64(WhisperPoWFlag.Name)

	if whisperConfig.MailServerNode && len(whisperConfig.PasswordFile) == 0 {
		return nil, errors.New("mail server requires --password to be specified")
	}

	if whisperConfig.NotificationServerNode && len(whisperConfig.IdentityFile) == 0 {
		return nil, errors.New("notification server requires either --identity file to be specified")
	}

	if len(whisperConfig.PasswordFile) > 0 { // make sure that we can load password file
		if whisperConfig.PasswordFile, err = filepath.Abs(whisperConfig.PasswordFile); err != nil {
			return nil, err
		}
		if _, err := whisperConfig.ReadPasswordFile(); err != nil {
			return nil, err
		}
	}

	if len(whisperConfig.IdentityFile) > 0 { // make sure that we can load identity file
		if whisperConfig.IdentityFile, err = filepath.Abs(whisperConfig.IdentityFile); err != nil {
			return nil, err
		}
		if _, err := whisperConfig.ReadIdentityFile(); err != nil {
			return nil, err
		}
	}

	firebaseConfig := whisperConfig.FirebaseConfig
	firebaseConfig.AuthorizationKeyFile = ctx.String(FirebaseAuthorizationKey.Name)
	if len(firebaseConfig.AuthorizationKeyFile) > 0 { // make sure authorization key can be loaded
		if firebaseConfig.AuthorizationKeyFile, err = filepath.Abs(firebaseConfig.AuthorizationKeyFile); err != nil {
			return nil, err
		}
		if _, err := firebaseConfig.ReadAuthorizationKeyFile(); err != nil {
			return nil, err
		}
	}

	return nodeConfig, nil
}
