//go:build !gowaku_no_rln
// +build !gowaku_no_rln

package rlngenerate

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

func getMembershipFee(ctx context.Context, rlnContract web3.RLNContract) (*big.Int, error) {
	return rlnContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: ctx})
}

func buildTransactor(ctx context.Context, membershipFee *big.Int, chainID *big.Int) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(options.ETHPrivateKey, chainID)
	if err != nil {
		return nil, err
	}

	auth.Value = membershipFee
	auth.Context = ctx
	auth.GasLimit = options.ETHGasLimit

	var ok bool

	if options.ETHNonce != "" {
		nonce := &big.Int{}
		auth.Nonce, ok = nonce.SetString(options.ETHNonce, 10)
		if !ok {
			return nil, errors.New("invalid nonce value")
		}
	}

	if options.ETHGasFeeCap != "" {
		gasFeeCap := &big.Int{}
		auth.GasFeeCap, ok = gasFeeCap.SetString(options.ETHGasFeeCap, 10)
		if !ok {
			return nil, errors.New("invalid gas fee cap value")
		}
	}

	if options.ETHGasTipCap != "" {
		gasTipCap := &big.Int{}
		auth.GasTipCap, ok = gasTipCap.SetString(options.ETHGasTipCap, 10)
		if !ok {
			return nil, errors.New("invalid gas tip cap value")
		}
	}

	if options.ETHGasPrice != "" {
		gasPrice := &big.Int{}
		auth.GasPrice, ok = gasPrice.SetString(options.ETHGasPrice, 10)
		if !ok {
			return nil, errors.New("invalid gas price value")
		}
	}

	return auth, nil
}

func register(ctx context.Context, web3Config *web3.Config, idComm rln.IDCommitment) (rln.MembershipIndex, error) {
	// check if the contract exists by calling a static function
	membershipFee, err := getMembershipFee(ctx, web3Config.RLNContract)
	if err != nil {
		return 0, err
	}

	auth, err := buildTransactor(ctx, membershipFee, web3Config.ChainID)
	if err != nil {
		return 0, err
	}

	log.Debug("registering an id commitment", zap.Binary("idComm", idComm[:]))

	// registers the idComm  into the membership contract whose address is in rlnPeer.membershipContractAddress
	tx, err := web3Config.RegistryContract.Register(auth, web3Config.RLNContract.StorageIndex, rln.Bytes32ToBigInt(idComm))
	if err != nil {
		return 0, fmt.Errorf("transaction error: %w", err)
	}

	explorerURL := ""
	switch web3Config.ChainID.Int64() {
	case 1:
		explorerURL = "https://etherscan.io"
	case 5:
		explorerURL = "https://goerli.etherscan.io"
	case 11155111:
		explorerURL = "https://sepolia.etherscan.io"
	}

	if explorerURL != "" {
		logger.Info(fmt.Sprintf("transaction broadcasted, find details of your registration transaction in %s/tx/%s", explorerURL, tx.Hash()))
	} else {
		logger.Info("transaction broadcasted.", zap.String("transactionHash", tx.Hash().String()))
	}

	logger.Warn("waiting for transaction to be mined...")

	txReceipt, err := bind.WaitMined(ctx, web3Config.ETHClient, tx)
	if err != nil {
		return 0, fmt.Errorf("transaction error: %w", err)
	}

	if txReceipt.Status != types.ReceiptStatusSuccessful {
		return 0, errors.New("transaction reverted")
	}

	// the receipt topic holds the hash of signature of the raised events
	evt, err := web3Config.RLNContract.ParseMemberRegistered(*txReceipt.Logs[0])
	if err != nil {
		return 0, err
	}

	var eventIDComm rln.IDCommitment = rln.BigIntToBytes32(evt.IdCommitment)

	log.Debug("information extracted from tx log", zap.Uint64("blockNumber", evt.Raw.BlockNumber), logging.HexBytes("idCommitment", eventIDComm[:]), zap.Uint64("index", evt.Index.Uint64()))

	if eventIDComm != idComm {
		return 0, errors.New("invalid id commitment key")
	}

	return rln.MembershipIndex(uint(evt.Index.Int64())), nil
}
