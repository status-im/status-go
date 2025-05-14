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
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

func (tm *TransactionManager) ClearLocalRouterTransactionsData() {
	tm.routerTransactions = nil
}

func (tm *TransactionManager) ApprovalRequiredForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.ProcessorName == pathProcessorName &&
			desc.RouterPath.ApprovalRequired {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) ApprovalPlacedForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.ProcessorName == pathProcessorName && desc.IsApprovalPlaced() {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) TxPlacedForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.ProcessorName == pathProcessorName && desc.IsTxPlaced() {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) getOrInitDetailsForPath(path *routes.Path) *wallettypes.RouterTransactionDetails {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.PathIdentity() == path.PathIdentity() {
			if desc.RouterPath.ProcessorName == pathProcessorCommon.ProcessorSwapParaswapName {
				// since the path is re-evaluated for swap after approval tx is placed we need to use the latest path
				desc.RouterPath = path
			}
			return desc
		}
	}

	newDetails := &wallettypes.RouterTransactionDetails{
		RouterPath: path,
	}
	tm.routerTransactions = append(tm.routerTransactions, newDetails)

	return newDetails
}

func buildApprovalTxForPath(transactor transactions.TransactorIface, path *routes.Path, addressFrom common.Address,
	usedNonces map[uint64]int64, signer ethTypes.Signer) (*wallettypes.TransactionData, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	addrTo := types.Address(path.FromToken.Address)
	approavalSendArgs := &wallettypes.SendTxArgs{
		Version: wallettypes.SendTxArgsVersion1,

		// tx fields
		From:                 types.Address(addressFrom),
		To:                   &addrTo,
		Value:                (*hexutil.Big)(big.NewInt(0)),
		Data:                 path.ApprovalPackedData,
		Nonce:                path.ApprovalTxNonce,
		Gas:                  (*hexutil.Uint64)(&path.ApprovalGasAmount),
		MaxFeePerGas:         path.ApprovalMaxFeesPerGas,
		MaxPriorityFeePerGas: path.ApprovalPriorityFee,
		ValueOut:             (*hexutil.Big)(big.NewInt(0)),

		// additional fields version 1
		FromChainID: path.FromChain.ChainID,
	}
	if path.FromToken != nil {
		approavalSendArgs.FromTokenID = path.FromToken.Symbol
	}

	builtApprovalTx, usedNonce, err := transactor.ValidateAndBuildTransaction(approavalSendArgs.FromChainID, *approavalSendArgs, lastUsedNonce)
	if err != nil {
		return nil, err
	}
	approvalTxHash := signer.Hash(builtApprovalTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	return &wallettypes.TransactionData{
		TxArgs:     approavalSendArgs,
		Tx:         builtApprovalTx,
		HashToSign: types.Hash(approvalTxHash),
	}, nil
}

func buildTxForPath(path *routes.Path, pathProcessors map[string]pathprocessor.PathProcessor,
	usedNonces map[uint64]int64, signer ethTypes.Signer, processorInputParams *pathprocessor.ProcessorInputParams) (*wallettypes.TransactionData, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	sendArgs := &wallettypes.SendTxArgs{
		Version: wallettypes.SendTxArgsVersion1,

		// tx fields
		From:                 types.Address(processorInputParams.FromAddr),
		Value:                path.AmountIn,
		Data:                 path.TxPackedData,
		Nonce:                path.TxNonce,
		Gas:                  (*hexutil.Uint64)(&path.TxGasAmount),
		MaxFeePerGas:         path.TxMaxFeesPerGas,
		MaxPriorityFeePerGas: path.TxPriorityFee,

		// additional fields version 1
		ValueIn:            path.AmountIn,
		ValueOut:           path.AmountOut,
		FromChainID:        path.FromChain.ChainID,
		ToChainID:          path.ToChain.ChainID,
		SlippagePercentage: processorInputParams.SlippagePercentage,
	}

	isContractDeployment := path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployCollectiblesName ||
		path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployAssetsName
	if !isContractDeployment {
		addrTo := types.Address(processorInputParams.ToAddr)
		sendArgs.To = &addrTo
	}

	if path.FromToken != nil {
		sendArgs.FromTokenID = path.FromToken.Symbol
		sendArgs.ToContractAddress = types.Address(path.FromToken.Address)

		// special handling for transfer tx if selected token is not ETH
		// TODO: we should fix that in the trasactor, but till then, the best place to handle it is here
		if !path.FromToken.IsNative() {
			sendArgs.Value = (*hexutil.Big)(big.NewInt(0))

			if path.ProcessorName == pathProcessorCommon.ProcessorTransferName ||
				path.ProcessorName == pathProcessorCommon.ProcessorStickersBuyName ||
				path.ProcessorName == pathProcessorCommon.ProcessorENSRegisterName ||
				path.ProcessorName == pathProcessorCommon.ProcessorENSReleaseName ||
				path.ProcessorName == pathProcessorCommon.ProcessorENSPublicKeyName ||
				path.ProcessorName == pathProcessorCommon.ProcessorERC721Name ||
				path.ProcessorName == pathProcessorCommon.ProcessorERC1155Name {
				// TODO: update functions from `TransactorIface` to use `ToContractAddress` (as an address of the contract a transaction should be sent to)
				// and `To` (as the destination address, recipient) of `SendTxArgs` struct appropriately
				toContractAddr := types.Address(path.FromToken.Address)
				sendArgs.To = &toContractAddr
			}
		} else if path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName || // special handling for community related txs, tokenID for those txs is ETH
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityMintTokensName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityRemoteBurnName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityBurnName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName {
			toContractAddr := types.Address(*path.UsedContractAddress)
			sendArgs.To = &toContractAddr
			sendArgs.ToContractAddress = toContractAddr
		}
	}
	if path.ToToken != nil {
		sendArgs.ToTokenID = path.ToToken.Symbol
	}

	builtTx, usedNonce, err := pathProcessors[path.ProcessorName].BuildTransactionV2(sendArgs, lastUsedNonce)
	if err != nil {
		return nil, err
	}
	txHash := signer.Hash(builtTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	return &wallettypes.TransactionData{
		TxArgs:     sendArgs,
		Tx:         builtTx,
		HashToSign: types.Hash(txHash),
	}, nil
}

func (tm *TransactionManager) BuildTransactionsFromRoute(route routes.Route, pathProcessors map[string]pathprocessor.PathProcessor,
	processorInputParams *pathprocessor.ProcessorInputParams) (*responses.SigningDetails, uint64, uint64, error) {
	if len(route) == 0 {
		return nil, 0, 0, ErrNoRoute
	}

	accFrom, err := tm.accountsDB.GetAccountByAddress(types.Address(processorInputParams.FromAddr))
	if err != nil {
		return nil, 0, 0, err
	}

	keypair, err := tm.accountsDB.GetKeypairByKeyUID(accFrom.KeyUID)
	if err != nil {
		return nil, 0, 0, err
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

		txDetails := tm.getOrInitDetailsForPath(path)

		// always check for approval tx first for the path and build it if needed
		if path.ApprovalRequired && !tm.ApprovalPlacedForPath(path.ProcessorName) {
			txDetails.ApprovalTxData, err = buildApprovalTxForPath(tm.transactor, path, processorInputParams.FromAddr, usedNonces, signer)
			if err != nil {
				return nil, path.FromChain.ChainID, path.ToChain.ChainID, err
			}
			response.Hashes = append(response.Hashes, txDetails.ApprovalTxData.HashToSign)

			// if approval is needed for swap, we cannot build the swap tx before the approval tx is mined
			if path.ProcessorName == pathProcessorCommon.ProcessorSwapParaswapName {
				continue
			}
		}

		// build tx for the path
		txDetails.TxData, err = buildTxForPath(path, pathProcessors, usedNonces, signer, processorInputParams)
		if err != nil {
			return nil, path.FromChain.ChainID, path.ToChain.ChainID, err
		}
		response.Hashes = append(response.Hashes, txDetails.TxData.HashToSign)
	}

	return response, 0, 0, nil
}

func getSignatureForTxHash(txHash string, signatures map[string]requests.SignatureDetails) ([]byte, error) {
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

func validateAndAddSignature(txData *wallettypes.TransactionData, signatures map[string]requests.SignatureDetails) error {
	if txData != nil && !txData.IsTxPlaced() {
		var err error
		txData.Signature, err = getSignatureForTxHash(txData.HashToSign.String(), signatures)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tm *TransactionManager) ValidateAndAddSignaturesToRouterTransactions(signatures map[string]requests.SignatureDetails) (uint64, uint64, error) {
	if len(tm.routerTransactions) == 0 {
		return 0, 0, ErrNoTrsansactionsBeingBuilt
	}

	// check if all transactions have been signed
	var err error
	for _, desc := range tm.routerTransactions {
		err = validateAndAddSignature(desc.ApprovalTxData, signatures)
		if err != nil {
			return desc.RouterPath.FromChain.ChainID, desc.RouterPath.ToChain.ChainID, err
		}

		err = validateAndAddSignature(desc.TxData, signatures)
		if err != nil {
			return desc.RouterPath.FromChain.ChainID, desc.RouterPath.ToChain.ChainID, err
		}
	}

	return 0, 0, nil
}

func addSignatureAndSendTransaction(
	transactor transactions.TransactorIface,
	txData *wallettypes.TransactionData,
	multiTransactionID walletCommon.MultiTransactionIDType,
	isApproval bool) (*responses.RouterSentTransaction, error) {
	var txWithSignature *ethTypes.Transaction
	var err error

	txWithSignature, err = transactor.AddSignatureToTransaction(txData.TxArgs.FromChainID, txData.Tx, txData.Signature)
	if err != nil {
		return nil, err
	}
	txData.Tx = txWithSignature

	txData.SentHash, err = transactor.SendTransactionWithSignature(common.Address(txData.TxArgs.From), txData.TxArgs.FromTokenID, multiTransactionID, txWithSignature)
	if err != nil {
		return nil, err
	}

	txData.TxArgs.MultiTransactionID = multiTransactionID

	if txWithSignature.To() == nil {
		toAddr := crypto.CreateAddress(txData.TxArgs.From, txData.Tx.Nonce())
		txData.TxArgs.To = &toAddr
	}

	return responses.NewRouterSentTransaction(txData.TxArgs, txData.SentHash, isApproval), nil
}

func (tm *TransactionManager) SendRouterTransactions(ctx context.Context, multiTx *MultiTransaction) (transactions []*responses.RouterSentTransaction, fromChainID uint64, toChainID uint64, err error) {
	transactions = make([]*responses.RouterSentTransaction, 0)

	// send transactions
	for _, desc := range tm.routerTransactions {
		fromChainID = desc.RouterPath.FromChain.ChainID
		toChainID = desc.RouterPath.ToChain.ChainID
		if desc.ApprovalTxData != nil && !desc.IsApprovalPlaced() {
			var response *responses.RouterSentTransaction
			response, err = addSignatureAndSendTransaction(tm.transactor, desc.ApprovalTxData, multiTx.ID, true)
			if err != nil {
				return
			}

			transactions = append(transactions, response)

			// if approval is needed for swap, then we need to wait for the approval tx to be mined before sending the swap tx
			if desc.RouterPath.ProcessorName == pathProcessorCommon.ProcessorSwapParaswapName {
				continue
			}
		}

		if desc.TxData != nil && !desc.IsTxPlaced() {
			var response *responses.RouterSentTransaction
			response, err = addSignatureAndSendTransaction(tm.transactor, desc.TxData, multiTx.ID, false)
			if err != nil {
				return
			}

			transactions = append(transactions, response)
		}
	}

	return
}

func (tm *TransactionManager) GetRouterTransactions() []*wallettypes.RouterTransactionDetails {
	return tm.routerTransactions
}
