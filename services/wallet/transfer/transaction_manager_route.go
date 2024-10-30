package transfer

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/transactions"
)

type BuildRouteExtraParams struct {
	AddressFrom        common.Address
	AddressTo          common.Address
	Username           string
	PublicKey          string
	PackID             *big.Int
	SlippagePercentage float32
}

func (tm *TransactionManager) ClearLocalRouterTransactionsData() {
	tm.routerTransactions = nil
}

func (tm *TransactionManager) ApprovalRequiredForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.routerPath.ProcessorName == pathProcessorName &&
			desc.routerPath.ApprovalRequired {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) ApprovalPlacedForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.routerPath.ProcessorName == pathProcessorName &&
			desc.approvalTxSentHash != (types.Hash{}) {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) TxPlacedForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.routerPath.ProcessorName == pathProcessorName &&
			desc.txSentHash != (types.Hash{}) {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) buildApprovalTxForPath(path *routes.Path, addressFrom common.Address,
	usedNonces map[uint64]int64, signer ethTypes.Signer) (types.Hash, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	data, err := walletCommon.PackApprovalInputData(path.AmountIn.ToInt(), path.ApprovalContractAddress)
	if err != nil {
		return types.Hash{}, err
	}

	addrTo := types.Address(path.FromToken.Address)
	approavalSendArgs := &transactions.SendTxArgs{
		Version: transactions.SendTxArgsVersion1,

		// tx fields
		From:                 types.Address(addressFrom),
		To:                   &addrTo,
		Value:                (*hexutil.Big)(big.NewInt(0)),
		Data:                 data,
		Gas:                  (*hexutil.Uint64)(&path.ApprovalGasAmount),
		MaxFeePerGas:         path.MaxFeesPerGas,
		MaxPriorityFeePerGas: path.ApprovalPriorityFee,
		ValueOut:             (*hexutil.Big)(big.NewInt(0)),

		// additional fields version 1
		FromChainID: path.FromChain.ChainID,
	}
	if path.FromToken != nil {
		approavalSendArgs.FromTokenID = path.FromToken.Symbol
	}

	builtApprovalTx, usedNonce, err := tm.transactor.ValidateAndBuildTransaction(approavalSendArgs.FromChainID, *approavalSendArgs, lastUsedNonce)
	if err != nil {
		return types.Hash{}, err
	}
	approvalTxHash := signer.Hash(builtApprovalTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	tm.routerTransactions = append(tm.routerTransactions, &RouterTransactionDetails{
		routerPath:         path,
		approvalTxArgs:     approavalSendArgs,
		approvalTx:         builtApprovalTx,
		approvalHashToSign: types.Hash(approvalTxHash),
	})

	return types.Hash(approvalTxHash), nil
}

func (tm *TransactionManager) buildTxForPath(path *routes.Path, pathProcessors map[string]pathprocessor.PathProcessor,
	usedNonces map[uint64]int64, signer ethTypes.Signer, params BuildRouteExtraParams) (types.Hash, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	processorInputParams := pathprocessor.ProcessorInputParams{
		FromAddr:  params.AddressFrom,
		ToAddr:    params.AddressTo,
		FromChain: path.FromChain,
		ToChain:   path.ToChain,
		FromToken: path.FromToken,
		ToToken:   path.ToToken,
		AmountIn:  path.AmountIn.ToInt(),
		AmountOut: path.AmountOut.ToInt(),

		Username:  params.Username,
		PublicKey: params.PublicKey,
		PackID:    params.PackID,
	}

	data, err := pathProcessors[path.ProcessorName].PackTxInputData(processorInputParams)
	if err != nil {
		return types.Hash{}, err
	}

	addrTo := types.Address(params.AddressTo)
	sendArgs := &transactions.SendTxArgs{
		Version: transactions.SendTxArgsVersion1,

		// tx fields
		From:                 types.Address(params.AddressFrom),
		To:                   &addrTo,
		Value:                path.AmountIn,
		Data:                 data,
		Gas:                  (*hexutil.Uint64)(&path.TxGasAmount),
		MaxFeePerGas:         path.MaxFeesPerGas,
		MaxPriorityFeePerGas: path.TxPriorityFee,

		// additional fields version 1
		ValueOut:           path.AmountOut,
		FromChainID:        path.FromChain.ChainID,
		ToChainID:          path.ToChain.ChainID,
		SlippagePercentage: params.SlippagePercentage,
	}
	if path.FromToken != nil {
		sendArgs.FromTokenID = path.FromToken.Symbol
		sendArgs.ToContractAddress = types.Address(path.FromToken.Address)

		// special handling for transfer tx if selected token is not ETH
		// TODO: we should fix that in the trasactor, but till then, the best place to handle it is here
		if !path.FromToken.IsNative() {
			sendArgs.Value = (*hexutil.Big)(big.NewInt(0))

			if path.ProcessorName == pathprocessor.ProcessorTransferName ||
				path.ProcessorName == pathprocessor.ProcessorStickersBuyName ||
				path.ProcessorName == pathprocessor.ProcessorENSRegisterName ||
				path.ProcessorName == pathprocessor.ProcessorENSReleaseName ||
				path.ProcessorName == pathprocessor.ProcessorENSPublicKeyName ||
				path.ProcessorName == pathprocessor.ProcessorERC721Name ||
				path.ProcessorName == pathprocessor.ProcessorERC1155Name {
				// TODO: update functions from `TransactorIface` to use `ToContractAddress` (as an address of the contract a transaction should be sent to)
				// and `To` (as the destination address, recipient) of `SendTxArgs` struct appropriately
				toContractAddr := types.Address(path.FromToken.Address)
				sendArgs.To = &toContractAddr
			}
		}
	}
	if path.ToToken != nil {
		sendArgs.ToTokenID = path.ToToken.Symbol
	}

	builtTx, usedNonce, err := pathProcessors[path.ProcessorName].BuildTransactionV2(sendArgs, lastUsedNonce)
	if err != nil {
		return types.Hash{}, err
	}
	txHash := signer.Hash(builtTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	tm.routerTransactions = append(tm.routerTransactions, &RouterTransactionDetails{
		routerPath:   path,
		txArgs:       sendArgs,
		tx:           builtTx,
		txHashToSign: types.Hash(txHash),
	})

	return types.Hash(txHash), nil
}

func (tm *TransactionManager) BuildTransactionsFromRoute(route routes.Route, pathProcessors map[string]pathprocessor.PathProcessor,
	params BuildRouteExtraParams) (*responses.SigningDetails, error) {
	if len(route) == 0 {
		return nil, ErrNoRoute
	}

	accFrom, err := tm.accountsDB.GetAccountByAddress(types.Address(params.AddressFrom))
	if err != nil {
		return nil, err
	}

	keypair, err := tm.accountsDB.GetKeypairByKeyUID(accFrom.KeyUID)
	if err != nil {
		return nil, err
	}

	response := &responses.SigningDetails{
		Address:       accFrom.Address,
		AddressPath:   accFrom.Path,
		KeyUid:        accFrom.KeyUID,
		SignOnKeycard: keypair.MigratedToKeycard(),
	}

	usedNonces := make(map[uint64]int64)
	for _, path := range route {
		signer := ethTypes.NewLondonSigner(big.NewInt(int64(path.FromChain.ChainID)))

		// always check for approval tx first for the path and build it if needed
		if path.ApprovalRequired && !tm.ApprovalPlacedForPath(path.ProcessorName) {
			approvalTxHash, err := tm.buildApprovalTxForPath(path, params.AddressFrom, usedNonces, signer)
			if err != nil {
				return nil, err
			}
			response.Hashes = append(response.Hashes, approvalTxHash)

			// if approval is needed for swap, we cannot build the swap tx before the approval tx is mined
			if path.ProcessorName == pathprocessor.ProcessorSwapParaswapName {
				continue
			}
		}

		// build tx for the path
		txHash, err := tm.buildTxForPath(path, pathProcessors, usedNonces, signer, params)
		if err != nil {
			return nil, err
		}
		response.Hashes = append(response.Hashes, txHash)
	}

	return response, nil
}

func getSignatureForTxHash(txHash string, signatures map[string]SignatureDetails) ([]byte, error) {
	sigDetails, ok := signatures[txHash]
	if !ok {
		err := &errors.ErrorResponse{
			Code:    ErrMissingSignatureForTx.Code,
			Details: fmt.Sprintf(ErrMissingSignatureForTx.Details, txHash),
		}
		return nil, err
	}

	err := sigDetails.Validate()
	if err != nil {
		return nil, err
	}

	rBytes, _ := hex.DecodeString(sigDetails.R)
	sBytes, _ := hex.DecodeString(sigDetails.S)
	vByte := byte(0)
	if sigDetails.V == "01" {
		vByte = 1
	}

	signature := make([]byte, crypto.SignatureLength)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(rBytes):64], sBytes)
	signature[64] = vByte

	return signature, nil
}

func (tm *TransactionManager) ValidateAndAddSignaturesToRouterTransactions(signatures map[string]SignatureDetails) error {
	if len(tm.routerTransactions) == 0 {
		return ErrNoTrsansactionsBeingBuilt
	}

	// check if all transactions have been signed
	for _, desc := range tm.routerTransactions {
		if desc.approvalTx != nil && desc.approvalTxSentHash == (types.Hash{}) {
			sig, err := getSignatureForTxHash(desc.approvalHashToSign.String(), signatures)
			if err != nil {
				return err
			}
			desc.approvalSignature = sig
		}

		if desc.tx != nil && desc.txSentHash == (types.Hash{}) {
			sig, err := getSignatureForTxHash(desc.txHashToSign.String(), signatures)
			if err != nil {
				return err
			}
			desc.txSignature = sig
		}
	}

	return nil
}

func (tm *TransactionManager) SendRouterTransactions(ctx context.Context, multiTx *MultiTransaction) (transactions []*responses.RouterSentTransaction, err error) {
	transactions = make([]*responses.RouterSentTransaction, 0)

	// send transactions
	for _, desc := range tm.routerTransactions {
		if desc.approvalTx != nil && desc.approvalTxSentHash == (types.Hash{}) {
			var approvalTxWithSignature *ethTypes.Transaction
			approvalTxWithSignature, err = tm.transactor.AddSignatureToTransaction(desc.approvalTxArgs.FromChainID, desc.approvalTx, desc.approvalSignature)
			if err != nil {
				return nil, err
			}

			desc.approvalTxSentHash, err = tm.transactor.SendTransactionWithSignature(common.Address(desc.approvalTxArgs.From), desc.approvalTxArgs.FromTokenID, multiTx.ID, approvalTxWithSignature)
			if err != nil {
				return nil, err
			}

			transactions = append(transactions, responses.NewRouterSentTransaction(desc.approvalTxArgs, desc.approvalTxSentHash, true))

			// if approval is needed for swap, then we need to wait for the approval tx to be mined before sending the swap tx
			if desc.routerPath.ProcessorName == pathprocessor.ProcessorSwapParaswapName {
				continue
			}
		}

		if desc.tx != nil && desc.txSentHash == (types.Hash{}) {
			var txWithSignature *ethTypes.Transaction
			txWithSignature, err = tm.transactor.AddSignatureToTransaction(desc.txArgs.FromChainID, desc.tx, desc.txSignature)
			if err != nil {
				return nil, err
			}

			desc.txSentHash, err = tm.transactor.SendTransactionWithSignature(common.Address(desc.txArgs.From), desc.txArgs.FromTokenID, multiTx.ID, txWithSignature)
			if err != nil {
				return nil, err
			}

			transactions = append(transactions, responses.NewRouterSentTransaction(desc.txArgs, desc.txSentHash, false))
		}
	}

	return
}
