package main

import (
	"fmt"

	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	faucetCommand = cli.Command{
		Action: faucetCommandHandler,
		Name:   "faucet",
		Usage:  "Starts faucet node (light node used by faucet service to request Ether)",
		Flags: []cli.Flag{
			HTTPPortFlag,
		},
	}
)

// faucetCommandHandler handles `statusd faucet` command
func faucetCommandHandler(ctx *cli.Context) error {
	config, err := parseFaucetCommandConfig(ctx)
	if err != nil {
		return fmt.Errorf("can not parse config: %v", err)
	}

	fmt.Println("Starting Status Faucet node..")
	if err := statusAPI.StartNode(config); err != nil {
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

// parseFaucetCommandConfig parses incoming CLI options and returns node configuration object
func parseFaucetCommandConfig(ctx *cli.Context) (*params.NodeConfig, error) {
	nodeConfig, err := makeNodeConfig(ctx)
	if err != nil {
		return nil, err
	}

	// select sub-protocols
	nodeConfig.LightEthConfig.Enabled = true
	nodeConfig.WhisperConfig.Enabled = false
	nodeConfig.SwarmConfig.Enabled = false

	// RPC configuration
	nodeConfig.APIModules = "eth"
	nodeConfig.HTTPHost = "0.0.0.0" // allow to connect from anywhere
	nodeConfig.HTTPPort = ctx.Int(HTTPPortFlag.Name)

	// extra options
	nodeConfig.BootClusterConfig.Enabled = true

	return nodeConfig, nil
}
