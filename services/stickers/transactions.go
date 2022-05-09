package stickers

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/transactions"
)

func (api *API) getSigner(chainID uint64, from types.Address, password string) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		selectedAccount, err := api.accountsManager.VerifyAccountPassword(api.keyStoreDir, from.Hex(), password)
		if err != nil {
			return nil, err
		}
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, selectedAccount.PrivateKey)
	}
}

func (api *API) Buy(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, packID *bigint.BigInt, password string) (string, error) {
	snt, err := api.contractMaker.NewSNT(chainID)
	if err != nil {
		return "", err
	}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: api.ctx, Pending: false}

	packInfo, err := stickerType.GetPackData(callOpts, packID.Int)
	if err != nil {
		return "", err
	}

	stickerMarketABI, err := abi.JSON(strings.NewReader(stickers.StickerMarketABI))
	if err != nil {
		return "", err
	}

	extraData, err := stickerMarketABI.Pack("buyToken", packID.Int, txArgs.From, packInfo.Price)
	if err != nil {
		return "", err
	}

	stickerMarketAddress, err := stickers.StickerMarketContractAddress(chainID)
	if err != nil {
		return "", err
	}

	txOpts := txArgs.ToTransactOpts(api.getSigner(chainID, txArgs.From, password))
	tx, err := snt.ApproveAndCall(
		txOpts,
		stickerMarketAddress,
		packInfo.Price,
		extraData,
	)

	if err != nil {
		return "", err
	}

	err = api.AddPending(chainID, packID)
	if err != nil {
		return "", err
	}

	// TODO: track pending transaction (do this in ENS service too)

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(types.Hash(tx.Hash()))
	return tx.Hash().String(), nil
}

func (api *API) BuyPrepareTxCallMsg(chainID uint64, from types.Address, packID *bigint.BigInt) (ethereum.CallMsg, error) {
	callOpts := &bind.CallOpts{Context: api.ctx, Pending: false}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	packInfo, err := stickerType.GetPackData(callOpts, packID.Int)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	stickerMarketABI, err := abi.JSON(strings.NewReader(stickers.StickerMarketABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	extraData, err := stickerMarketABI.Pack("buyToken", packID.Int, from, packInfo.Price)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	stickerMarketAddress, err := stickers.StickerMarketContractAddress(chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := sntABI.Pack("approveAndCall", stickerMarketAddress, packInfo.Price, extraData)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntAddress, err := snt.ContractAddress(chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	return ethereum.CallMsg{
		From:  common.Address(from),
		To:    &sntAddress,
		Value: big.NewInt(0),
		Data:  data,
	}, nil
}

func (api *API) BuyPrepareTx(ctx context.Context, chainID uint64, from types.Address, packID *bigint.BigInt) (interface{}, error) {
	callMsg, err := api.BuyPrepareTxCallMsg(chainID, from, packID)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

func (api *API) BuyEstimate(ctx context.Context, chainID uint64, from types.Address, packID *bigint.BigInt) (uint64, error) {
	callMsg, err := api.BuyPrepareTxCallMsg(chainID, from, packID)
	if err != nil {
		return 0, err
	}
	ethClient, err := api.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(ctx, callMsg)
}

func (api *API) StickerMarketAddress(ctx context.Context, chainID uint64) (common.Address, error) {
	return stickers.StickerMarketContractAddress(chainID)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
