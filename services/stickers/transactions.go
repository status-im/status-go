package stickers

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/transactions"
)

func (api *API) getSigner(chainID uint64, from types.Address, password string) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		selectedAccount, err := api.accountsManager.VerifyAccountPassword(api.config.KeyStoreDir, from.Hex(), password)
		if err != nil {
			return nil, err
		}
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, selectedAccount.PrivateKey)
	}
}

func (api *API) Buy(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, packID *big.Int, password string) (string, error) {
	err := api.AddPending(chainID, packID.Uint64())
	if err != nil {
		return "", err
	}

	snt, err := api.contractMaker.NewSNT(chainID)
	if err != nil {
		return "", err
	}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: api.ctx, Pending: false}

	packInfo, err := stickerType.GetPackData(callOpts, packID)
	if err != nil {
		return "", err
	}

	stickerMarketABI, err := abi.JSON(strings.NewReader(stickers.StickerMarketABI))
	if err != nil {
		return "", err
	}

	extraData, err := stickerMarketABI.Pack("buyToken", packID, packInfo.Price)
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

	// TODO: track pending transaction (do this in ENS service too)

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(types.Hash(tx.Hash()))
	return tx.Hash().String(), nil
}

func (api *API) BuyEstimate(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, packID *big.Int) (uint64, error) {
	callOpts := &bind.CallOpts{Context: api.ctx, Pending: false}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return 0, err
	}

	packInfo, err := stickerType.GetPackData(callOpts, packID)
	if err != nil {
		return 0, err
	}

	stickerMarketABI, err := abi.JSON(strings.NewReader(stickers.StickerMarketABI))
	if err != nil {
		return 0, err
	}

	extraData, err := stickerMarketABI.Pack("buyToken", packID, packInfo.Price)
	if err != nil {
		return 0, err
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return 0, err
	}

	stickerMarketAddress, err := stickers.StickerMarketContractAddress(chainID)
	if err != nil {
		return 0, err
	}

	data, err := sntABI.Pack("approveAndCall", stickerMarketAddress, packInfo.Price, extraData)
	if err != nil {
		return 0, err
	}

	ethClient, err := api.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	sntAddress, err := snt.ContractAddress(chainID)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(ctx, ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    &sntAddress,
		Value: big.NewInt(0),
		Data:  data,
	})
}
