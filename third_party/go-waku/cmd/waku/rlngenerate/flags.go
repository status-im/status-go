//go:build !gowaku_no_rln
// +build !gowaku_no_rln

package rlngenerate

import (
	cli "github.com/urfave/cli/v2"
	wcli "github.com/waku-org/go-waku/waku/cliutils"
)

var flags = []cli.Flag{
	&cli.PathFlag{
		Name:        "cred-path",
		Usage:       "RLN relay membership credentials file",
		Value:       "./rlnKeystore.json",
		Destination: &options.CredentialsPath,
	},
	&cli.StringFlag{
		Name:        "cred-password",
		Value:       "password",
		Usage:       "Password for encrypting RLN credentials",
		Destination: &options.CredentialsPassword,
	},
	&cli.GenericFlag{
		Name:  "eth-account-private-key",
		Usage: "Ethereum  account private key used for registering in member contract",
		Value: &wcli.PrivateKeyValue{
			Value: &options.ETHPrivateKey,
		},
	},
	&cli.StringFlag{
		Name:        "eth-client-address",
		Usage:       "Ethereum testnet client address",
		Value:       "ws://localhost:8545",
		Destination: &options.ETHClientAddress,
	},
	&cli.GenericFlag{
		Name:  "eth-contract-address",
		Usage: "Address of membership contract",
		Value: &wcli.AddressValue{
			Value: &options.MembershipContractAddress,
		},
	},
	&cli.StringFlag{
		Name:        "eth-nonce",
		Value:       "",
		Usage:       "Set an specific ETH transaction nonce. Leave empty to calculate the nonce automatically",
		Destination: &options.ETHNonce,
	},
	&cli.Uint64Flag{
		Name:        "eth-gas-limit",
		Value:       0,
		Usage:       "Gas limit to set for the transaction execution (0 = estimate)",
		Destination: &options.ETHGasLimit,
	},
	&cli.StringFlag{
		Name:        "eth-gas-price",
		Value:       "",
		Usage:       "Gas price  in wei to use for the transaction execution (empty = gas price oracle)",
		Destination: &options.ETHGasPrice,
	},
	&cli.StringFlag{
		Name:        "eth-gas-fee-cap",
		Value:       "",
		Usage:       "Gas fee cap in wei to use for the 1559 transaction execution (empty = gas price oracle)",
		Destination: &options.ETHGasFeeCap,
	},
	&cli.StringFlag{
		Name:        "eth-gas-tip-cap",
		Value:       "",
		Usage:       "Gas priority fee cap in wei to use for the 1559 transaction execution (empty = gas price oracle)",
		Destination: &options.ETHGasTipCap,
	},
}
