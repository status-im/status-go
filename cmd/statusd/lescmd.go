package main

import (
	"fmt"

	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	lesCommand = cli.Command{
		Action: lesCommandHandler,
		Name:   "les",
		Usage:  "Starts Light Ethereum node",
		Flags: []cli.Flag{
			WhisperEnabledFlag,
			SwarmEnabledFlag,
			HTTPEnabledFlag,
			HTTPPortFlag,
			IPCEnabledFlag,
		},
	}
)

// lesCommandHandler handles `statusd les` command
func lesCommandHandler(ctx *cli.Context) error {
	config, err := parseLESCommandConfig(ctx)
	if err != nil {
		return fmt.Errorf("can not parse config: %v", err)
	}

	fmt.Println("Starting Light Status node..")
	if err = statusAPI.StartNode(config); err != nil {
		return err
	}

	// wait till node has been stopped
	node, err := statusAPI.NodeManager().Node()
	if err != nil {
		return nil
	}
	node.Wait()

	return nil
}

// parseLESCommandConfig parses incoming CLI options and returns node configuration object
func parseLESCommandConfig(ctx *cli.Context) (*params.NodeConfig, error) {
	nodeConfig, err := makeNodeConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Enabled sub-protocols
	nodeConfig.LightEthConfig.Enabled = true
	nodeConfig.RPCEnabled = ctx.Bool(HTTPEnabledFlag.Name)
	nodeConfig.WhisperConfig.Enabled = ctx.Bool(WhisperEnabledFlag.Name)
	nodeConfig.SwarmConfig.Enabled = ctx.Bool(SwarmEnabledFlag.Name)

	// RPC configuration
	if !ctx.Bool(HTTPEnabledFlag.Name) {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}
	nodeConfig.HTTPPort = ctx.Int(HTTPPortFlag.Name)
	nodeConfig.IPCEnabled = ctx.Bool(IPCEnabledFlag.Name)

	return nodeConfig, nil
}
