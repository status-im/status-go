//go:build !gowaku_no_rln
// +build !gowaku_no_rln

package rlngenerate

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	cli "github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

var options Options
var logger = utils.Logger().Named("rln-credentials")

// Command generates a key file used to generate the node's peerID, encrypted with an optional password
var Command = cli.Command{
	Name:  "generate-rln-credentials",
	Usage: "Generate credentials for usage with RLN",
	Action: func(cCtx *cli.Context) error {
		if options.ETHPrivateKey == nil {
			err := errors.New("a private key must be specified")
			logger.Error("validating option flags", zap.Error(err))
			return cli.Exit(err, 1)
		}

		err := execute(context.Background())
		if err != nil {
			logger.Error("registering RLN credentials", zap.Error(err))
			return cli.Exit(err, 1)
		}

		return nil
	},
	Flags: flags,
}

func execute(ctx context.Context) error {
	rlnInstance, err := rln.NewRLN()
	if err != nil {
		return err
	}

	web3Config, err := web3.BuildConfig(ctx, options.ETHClientAddress, options.MembershipContractAddress)
	if err != nil {
		return err
	}

	// prepare rln membership key pair
	logger.Info("generating rln credential")
	identityCredential, err := rlnInstance.MembershipKeyGen()
	if err != nil {
		return err
	}

	// register the rln-relay peer to the membership contract
	membershipIndex, err := register(ctx, web3Config, identityCredential.IDCommitment)
	if err != nil {
		return err
	}

	// TODO: clean private key from memory

	err = persistCredentials(identityCredential, membershipIndex, web3Config.ChainID)
	if err != nil {
		return err
	}

	if logger.Level() == zap.DebugLevel {
		logger.Info("registered credentials into the membership contract",
			logging.HexBytes("IDCommitment", identityCredential.IDCommitment[:]),
			logging.HexBytes("IDNullifier", identityCredential.IDNullifier[:]),
			logging.HexBytes("IDSecretHash", identityCredential.IDSecretHash[:]),
			logging.HexBytes("IDTrapDoor", identityCredential.IDTrapdoor[:]),
			zap.Uint("index", membershipIndex),
		)
	} else {
		logger.Info("registered credentials into the membership contract", logging.HexBytes("idCommitment", identityCredential.IDCommitment[:]), zap.Uint("index", membershipIndex))
	}

	web3Config.ETHClient.Close()

	return nil
}

func persistCredentials(identityCredential *rln.IdentityCredential, treeIndex rln.MembershipIndex, chainID *big.Int) error {
	appKeystore, err := keystore.New(options.CredentialsPath, dynamic.RLNAppInfo, logger)
	if err != nil {
		return err
	}

	membershipCredential := keystore.MembershipCredentials{
		IdentityCredential:     identityCredential,
		TreeIndex:              treeIndex,
		MembershipContractInfo: keystore.NewMembershipContractInfo(chainID, options.MembershipContractAddress),
	}

	err = appKeystore.AddMembershipCredentials(membershipCredential, options.CredentialsPassword)
	if err != nil {
		return fmt.Errorf("failed to persist credentials: %w", err)
	}

	logger.Info("persisted credentials succesfully")

	return nil
}
