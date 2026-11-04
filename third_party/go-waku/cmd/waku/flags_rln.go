//go:build !gowaku_no_rln
// +build !gowaku_no_rln

package main

import (
	cli "github.com/urfave/cli/v2"
	wcli "github.com/waku-org/go-waku/waku/cliutils"
)

func rlnFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "rln-relay",
			Value:       false,
			Usage:       "Enable spam protection through rln-relay",
			Destination: &options.RLNRelay.Enable,
		},
		&cli.GenericFlag{
			Name:  "rln-relay-cred-index",
			Usage: "the index of the onchain commitment to use",
			Value: &wcli.OptionalUint{
				Value: &options.RLNRelay.MembershipIndex,
			},
		},
		&cli.BoolFlag{
			Name:        "rln-relay-dynamic",
			Usage:       "Enable waku-rln-relay with on-chain dynamic group management",
			Destination: &options.RLNRelay.Dynamic,
		},
		&cli.PathFlag{
			Name:        "rln-relay-cred-path",
			Usage:       "RLN relay membership credentials file",
			Value:       "",
			Destination: &options.RLNRelay.CredentialsPath,
		},
		&cli.StringFlag{
			Name:        "rln-relay-cred-password",
			Value:       "",
			Usage:       "Password for encrypting RLN credentials",
			Destination: &options.RLNRelay.CredentialsPassword,
		},
		&cli.StringFlag{
			Name:        "rln-relay-tree-path",
			Value:       "",
			Usage:       "Path to the RLN merkle tree sled db (https://github.com/spacejam/sled)",
			Destination: &options.RLNRelay.TreePath,
		},
		&cli.StringFlag{
			Name:        "rln-relay-eth-client-address",
			Usage:       "Ethereum testnet client address",
			Value:       "ws://localhost:8545",
			Destination: &options.RLNRelay.ETHClientAddress,
		},
		&cli.GenericFlag{
			Name:  "rln-relay-eth-contract-address",
			Usage: "Address of membership contract",
			Value: &wcli.AddressValue{
				Value: &options.RLNRelay.MembershipContractAddress,
			},
		},
	}
}
