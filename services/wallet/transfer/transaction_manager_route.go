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

func (tm *TransactionManager) getOrInitDetailsForPath(path *routes.Path) *RouterTransactionDetails {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.ID() == path.ID() {
			return desc
		}
	}

	newDetails := &RouterTransactionDetails{
		RouterPath: path,
	}
	tm.routerTransactions = append(tm.routerTransactions, newDetails)

	return newDetails
}

func buildApprovalTxForPath(transactor transactions.TransactorIface, path *routes.Path, addressFrom common.Address,
	usedNonces map[uint64]int64, signer ethTypes.Signer) (*TransactionData, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	data, err := walletCommon.PackApprovalInputData(path.AmountIn.ToInt(), path.ApprovalContractAddress)
	if err != nil {
		return nil, err
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

	builtApprovalTx, usedNonce, err := transactor.ValidateAndBuildTransaction(approavalSendArgs.FromChainID, *approavalSendArgs, lastUsedNonce)
	if err != nil {
		return nil, err
	}
	approvalTxHash := signer.Hash(builtApprovalTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	return &TransactionData{
		TxArgs:     approavalSendArgs,
		Tx:         builtApprovalTx,
		HashToSign: types.Hash(approvalTxHash),
	}, nil
}

func buildTxForPath(transactor transactions.TransactorIface, path *routes.Path, pathProcessors map[string]pathprocessor.PathProcessor,
	usedNonces map[uint64]int64, signer ethTypes.Signer, params BuildRouteExtraParams) (*TransactionData, error) {
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
		return nil, err
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
		return nil, err
	}
	txHash := signer.Hash(builtTx)
	usedNonces[path.FromChain.ChainID] = int64(usedNonce)

	return &TransactionData{
		TxArgs:     sendArgs,
		Tx:         builtTx,
		HashToSign: types.Hash(txHash),
	}, nil
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

		txDetails := tm.getOrInitDetailsForPath(path)

		// always check for approval tx first for the path and build it if needed
		if path.ApprovalRequired && !tm.ApprovalPlacedForPath(path.ProcessorName) {
			txDetails.ApprovalTxData, err = buildApprovalTxForPath(tm.transactor, path, params.AddressFrom, usedNonces, signer)
			if err != nil {
				return nil, err
			}
			response.Hashes = append(response.Hashes, txDetails.ApprovalTxData.HashToSign)

			// if approval is needed for swap, we cannot build the swap tx before the approval tx is mined
			if path.ProcessorName == pathprocessor.ProcessorSwapParaswapName {
				continue
			}
		}

		// build tx for the path
		txDetails.TxData, err = buildTxForPath(tm.transactor, path, pathProcessors, usedNonces, signer, params)
		if err != nil {
			return nil, err
		}
		response.Hashes = append(response.Hashes, txDetails.TxData.HashToSign)
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

func validateAndAddSignature(txData *TransactionData, signatures map[string]SignatureDetails) error {
	if txData != nil && !txData.IsTxPlaced() {
		var err error
		txData.Signature, err = getSignatureForTxHash(txData.HashToSign.String(), signatures)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tm *TransactionManager) ValidateAndAddSignaturesToRouterTransactions(signatures map[string]SignatureDetails) error {
	if len(tm.routerTransactions) == 0 {
		return ErrNoTrsansactionsBeingBuilt
	}

	// check if all transactions have been signed
	var err error
	for _, desc := range tm.routerTransactions {
		err = validateAndAddSignature(desc.ApprovalTxData, signatures)
		if err != nil {
			return err
		}

		err = validateAndAddSignature(desc.TxData, signatures)
		if err != nil {
			return err
		}
	}

	return nil
}

func addSignatureAndSendTransaction(
	transactor transactions.TransactorIface,
	txData *TransactionData,
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

	return responses.NewRouterSentTransaction(txData.TxArgs, txData.SentHash, isApproval), nil
}

func (tm *TransactionManager) SendRouterTransactions(ctx context.Context, multiTx *MultiTransaction) (transactions []*responses.RouterSentTransaction, err error) {
	transactions = make([]*responses.RouterSentTransaction, 0)

	// send transactions
	for _, desc := range tm.routerTransactions {
		if desc.ApprovalTxData != nil && !desc.IsApprovalPlaced() {
			var response *responses.RouterSentTransaction
			response, err = addSignatureAndSendTransaction(tm.transactor, desc.ApprovalTxData, multiTx.ID, true)
			if err != nil {
				return
			}

			transactions = append(transactions, response)

			// if approval is needed for swap, then we need to wait for the approval tx to be mined before sending the swap tx
			if desc.RouterPath.ProcessorName == pathprocessor.ProcessorSwapParaswapName {
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

func (tm *TransactionManager) GetRouterTransactions() []*RouterTransactionDetails {
	return tm.routerTransactions
}
