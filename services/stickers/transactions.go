package stickers

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

// Prepares a transaction for buying a sticker pack and returns the transaction hash that needs to be signed.
// The transaction will be signed using the SignPreparedTx method or elswhere (on Keycard) and then sent using the SendPreparedTxWithSignature method.
func (api *API) PrepareTxForBuyingStickers(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, packID *bigint.BigInt) (interface{}, error) {
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

	txOpts := txArgs.ToTransactOpts(nil)
	tx, err := snt.ApproveAndCall(
		txOpts,
		stickerMarketAddress,
		packInfo.Price,
		extraData,
	)

	if err != nil {
		return "", err
	}

	api.txSignDetails = &txSigningDetails{
		txType:        transactions.BuyStickerPack,
		chainID:       chainID,
		from:          txOpts.From,
		txBeingSigned: tx,
		packID:        packID,
	}

	signer := ethTypes.NewLondonSigner(new(big.Int).SetUint64(api.txSignDetails.chainID))
	return signer.Hash(api.txSignDetails.txBeingSigned), nil
}

// Signs a transaction using appropriate local keystore file that is decrypted with provided password and returns the signature in hex format.
// The tx that will be signed is the one that was prepared by the last call to PrepareTxForBuyingStickers method.
func (api *API) SignPreparedTx(password string) (interface{}, error) {
	if api.txSignDetails == nil {
		return nil, errors.New("no tx to sign")
	}

	if api.txSignDetails.txType != transactions.BuyStickerPack {
		return nil, errors.New("not supported tx type")
	}

	return utils.SignTx(api.txSignDetails.chainID, api.accountsManager, api.keyStoreDir,
		types.Address(api.txSignDetails.from), password, api.txSignDetails.txBeingSigned)
}

// Sends a transaction using provided signature and returns the transaction hash.
// Provided signature must be in hex format and must be the result of signing the tx that was prepared by the last call to PrepareTxForBuyingStickers method.
func (api *API) SendPreparedTxWithSignature(ctx context.Context, signature string) (string, error) {
	if api.txSignDetails == nil {
		return "", errors.New("no known tx to send to register ens username")
	}

	if api.txSignDetails.txType != transactions.BuyStickerPack {
		return "", errors.New("not supported tx type")
	}

	signature = strings.TrimPrefix(signature, "0x")
	byteSignature, err := hex.DecodeString(signature)
	if err != nil {
		return "", err
	}
	if len(byteSignature) != transactions.ValidSignatureSize {
		return "", transactions.ErrInvalidSignatureSize
	}

	signer := ethTypes.NewLondonSigner(new(big.Int).SetUint64(api.txSignDetails.chainID))
	signedTx, err := api.txSignDetails.txBeingSigned.WithSignature(signer, byteSignature)
	if err != nil {
		return "", err
	}

	backend, err := api.contractMaker.RPCClient.EthClient(api.txSignDetails.chainID)
	if err != nil {
		return "", err
	}

	err = backend.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", err
	}

	err = api.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(api.txSignDetails.chainID),
		signedTx.Hash(),
		api.txSignDetails.from,
		api.txSignDetails.txType,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction for txType ", api.txSignDetails.txType, "error", err)
		return "", err
	}

	if api.txSignDetails.txType == transactions.BuyStickerPack {
		err = api.AddPending(api.txSignDetails.chainID, api.txSignDetails.packID)
		if err != nil {
			log.Warn("Buying stickers pack: transaction successful, but adding failed")
		}
	}

	return signedTx.Hash().String(), nil
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
