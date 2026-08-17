package transfer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	crypto2 "github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/internal/transactions"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/permit2"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

func (tm *TransactionManager) ClearLocalRouterTransactionsData() {
	tm.routerTransactions = nil
}

func (tm *TransactionManager) ApprovalPlacedForPath(pathProcessorName string) bool {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.ProcessorName == pathProcessorName && desc.IsApprovalPlaced() {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) anySwapPath(pred func(*wallettypes.RouterTransactionDetails) bool) bool {
	for _, desc := range tm.routerTransactions {
		if walletCommon.IsProcessorSwap(desc.RouterPath.ProcessorName) && pred(desc) {
			return true
		}
	}
	return false
}

func (tm *TransactionManager) ApprovalRequiredForSwap() bool {
	return tm.anySwapPath(func(desc *wallettypes.RouterTransactionDetails) bool { return desc.RouterPath.ApprovalRequired })
}

func (tm *TransactionManager) ApprovalPlacedForSwap() bool {
	return tm.anySwapPath(func(desc *wallettypes.RouterTransactionDetails) bool { return desc.IsApprovalPlaced() })
}

func (tm *TransactionManager) TxPlacedForSwap() bool {
	return tm.anySwapPath(func(desc *wallettypes.RouterTransactionDetails) bool { return desc.IsTxPlaced() })
}

func (tm *TransactionManager) getOrInitDetailsForPath(path *routes.Path) *wallettypes.RouterTransactionDetails {
	for _, desc := range tm.routerTransactions {
		if desc.RouterPath.PathIdentity() == path.PathIdentity() {
			if walletCommon.IsProcessorSwap(desc.RouterPath.ProcessorName) {
				// since the path is re-evaluated for swap after approval tx is placed we need to use the latest path
				carryOverPermitSignature(desc.RouterPath, path)
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

// carryOverPermitSignature moves an already-collected permit signature onto the
// re-evaluated path. Route lookups hand back deep copies, so the path seen in the second
// signing phase isn't the object the signature was attached to and the user would be asked
// to sign the same permit again. Only carried over when both digests match, a differing
// one means the old signature is void.
func carryOverPermitSignature(from, to *routes.Path) {
	if from.PermitDetails == nil || to.PermitDetails == nil || len(from.PermitDetails.Signature) == 0 {
		return
	}

	fromDigest, err := from.PermitDetails.Digest()
	if err != nil {
		return
	}
	toDigest, err := to.PermitDetails.Digest()
	if err != nil || fromDigest != toDigest {
		return
	}

	to.PermitDetails.Signature = from.PermitDetails.Signature
}

func buildApprovalTxForPath(transactor transactions.TransactorIface, path *routes.Path, addressFrom common.Address,
	usedNonces map[uint64]int64, signer ethTypes.Signer) (*wallettypes.TransactionData, error) {
	lastUsedNonce := int64(-1)
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		lastUsedNonce = nonce
	}

	addrTo := types2.Address(path.FromToken.Address)
	approavalSendArgs := &wallettypes.SendTxArgs{
		Version: wallettypes.SendTxArgsVersion1,

		// tx fields
		From:     types2.Address(addressFrom),
		To:       &addrTo,
		Value:    (*hexutil.Big)(big.NewInt(0)),
		Data:     path.ApprovalPackedData,
		Nonce:    path.ApprovalTxNonce,
		Gas:      (*hexutil.Uint64)(&path.ApprovalGasAmount),
		ValueOut: (*hexutil.Big)(big.NewInt(0)),

		// additional fields version 1
		FromChainID: path.FromChain.ChainID,
		FromToken:   path.FromToken,
	}

	// set appropriate fields based on EIP-1559 compatibility of the chai
	if !path.FromChain.EIP1559Enabled {
		approavalSendArgs.GasPrice = path.ApprovalGasPrice
	} else {
		approavalSendArgs.MaxFeePerGas = path.ApprovalMaxFeesPerGas
		approavalSendArgs.MaxPriorityFeePerGas = path.ApprovalPriorityFee
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
		HashToSign: types2.Hash(approvalTxHash),
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
		From:  types2.Address(processorInputParams.FromAddr),
		Value: path.AmountIn,
		Data:  path.TxPackedData,
		Nonce: path.TxNonce,
		Gas:   (*hexutil.Uint64)(&path.TxGasAmount),

		// additional fields version 1
		FromToken:          path.FromToken,
		ToToken:            path.ToToken,
		ValueIn:            path.AmountIn,
		ValueOut:           path.AmountOut,
		FromChainID:        path.FromChain.ChainID,
		ToChainID:          path.ToChain.ChainID,
		SlippagePercentage: processorInputParams.SlippagePercentage,

		// Carries the signed permit to the processor, which wraps the swap calldata in the
		// matching Permit2Proxy call.
		PermitDetails: path.PermitDetails,
	}

	if !path.FromChain.EIP1559Enabled {
		sendArgs.GasPrice = path.TxGasPrice
	} else {
		sendArgs.MaxFeePerGas = path.TxMaxFeesPerGas
		sendArgs.MaxPriorityFeePerGas = path.TxPriorityFee
	}

	isContractDeployment := path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployCollectiblesName ||
		path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployAssetsName
	if !isContractDeployment {
		addrTo := types2.Address(processorInputParams.ToAddr)
		sendArgs.To = &addrTo
	}

	if path.FromToken != nil {
		sendArgs.ToContractAddress = types2.Address(path.FromToken.Address)

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
				toContractAddr := types2.Address(path.FromToken.Address)
				sendArgs.To = &toContractAddr
			}
		} else if path.ProcessorName == pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName || // special handling for community related txs, tokenID for those txs is ETH
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityMintTokensName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityRemoteBurnName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunityBurnName ||
			path.ProcessorName == pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName {
			toContractAddr := types2.Address(*path.UsedContractAddress)
			sendArgs.To = &toContractAddr
			sendArgs.ToContractAddress = toContractAddr
		}
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
		HashToSign: types2.Hash(txHash),
	}, nil
}

func (tm *TransactionManager) BuildTransactionsFromRoute(route routes.Route, pathProcessors map[string]pathprocessor.PathProcessor,
	processorInputParams *pathprocessor.ProcessorInputParams) (*responses.SigningDetails, uint64, uint64, error) {
	if len(route) == 0 {
		return nil, 0, 0, ErrNoRoute
	}

	accFrom, err := tm.accountsDB.GetAccountByAddress(types2.Address(processorInputParams.FromAddr))
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
		SignOnKeycard: keypair.MigratedToColdWallet(),
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
			response.AddTxHash(txDetails.ApprovalTxData.HashToSign)

			// if approval is needed for swap, we cannot build the swap tx before the approval tx is mined
			if walletCommon.IsProcessorSwap(path.ProcessorName) {
				continue
			}
		}

		// The permit signature goes into the calldata, so the tx hash doesn't exist until
		// it's signed. Ask for the digest now and build the tx in the second phase.
		if path.PermitDetails != nil && len(path.PermitDetails.Signature) == 0 {
			digest, permitErr := path.PermitDetails.Digest()
			if permitErr != nil {
				return nil, path.FromChain.ChainID, path.ToChain.ChainID, permitErr
			}
			typedData, permitErr := marshalPermitTypedData(path.PermitDetails)
			if permitErr != nil {
				return nil, path.FromChain.ChainID, path.ToChain.ChainID, permitErr
			}
			response.AddPermitHash(types2.Hash(digest), typedData)
			continue
		}

		// build tx for the path
		txDetails.TxData, err = buildTxForPath(path, pathProcessors, usedNonces, signer, processorInputParams)
		if err != nil {
			return nil, path.FromChain.ChainID, path.ToChain.ChainID, err
		}
		response.AddTxHash(txDetails.TxData.HashToSign)
	}

	return response, 0, 0, nil
}

// marshalPermitTypedData renders the EIP-712 payload the client shows the user before
// they sign.
func marshalPermitTypedData(details *permit2.Details) (string, error) {
	typedData, err := details.TypedData()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(typedData)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// AddPermitSignatures attaches client-provided permit signatures to the paths waiting on
// them, matched by digest. Returns the number of paths that got one.
func (tm *TransactionManager) AddPermitSignatures(signatures map[string]requests.SignatureDetails) (int, error) {
	if len(tm.routerTransactions) == 0 {
		return 0, ErrNoTrsansactionsBeingBuilt
	}

	applied := 0
	for _, desc := range tm.routerTransactions {
		details := desc.RouterPath.PermitDetails
		if details == nil || len(details.Signature) > 0 {
			continue
		}

		digest, err := details.Digest()
		if err != nil {
			return applied, err
		}

		signature, err := signatureFor(types2.Hash(digest).String(), signatures, recoveryOffsetEIP712)
		if err != nil {
			return applied, err
		}
		details.Signature = signature
		applied++
	}

	return applied, nil
}

// recoveryOffset is the convention the consumer of a signature expects for its recovery
// byte. Transactions carry the raw parity the signer applies EIP-155 to; EIP-712 permits
// (Permit2 and ERC20Permit) expect the 27/28 form.
type recoveryOffset byte

const (
	recoveryOffsetRaw    recoveryOffset = 0
	recoveryOffsetEIP712 recoveryOffset = 27
)

// signatureFor assembles the 65-byte signature the client produced for the given key, a
// transaction hash or a permit digest. r and s are right-aligned in their 32-byte halves.
func signatureFor(key string, signatures map[string]requests.SignatureDetails, offset recoveryOffset) ([]byte, error) {
	sigDetails, ok := signatures[key]
	if !ok {
		return nil, &errors.ErrorResponse{
			Code:    ErrMissingSignatureForTx.Code,
			Details: fmt.Sprintf(ErrMissingSignatureForTx.Details, key),
		}
	}
	if err := sigDetails.Validate(); err != nil {
		return nil, err
	}

	// Validate only checks string lengths, so malformed hex reaches this point: a partial
	// decode would be padded into a well-formed but wrong signature.
	rBytes, err := hex.DecodeString(sigDetails.R)
	if err != nil {
		return nil, err
	}
	sBytes, err := hex.DecodeString(sigDetails.S)
	if err != nil {
		return nil, err
	}

	signature := make([]byte, crypto2.SignatureLength)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)
	signature[64] = byte(offset)
	if sigDetails.V == "01" {
		signature[64]++
	}

	return signature, nil
}

func validateAndAddSignature(txData *wallettypes.TransactionData, signatures map[string]requests.SignatureDetails) error {
	if txData != nil && !txData.IsTxPlaced() {
		var err error
		txData.Signature, err = signatureFor(txData.HashToSign.String(), signatures, recoveryOffsetRaw)
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
	isApproval bool) (*responses.RouterSentTransaction, error) {
	var txWithSignature *ethTypes.Transaction
	var err error

	txWithSignature, err = transactor.AddSignatureToTransaction(txData.TxArgs.FromChainID, txData.Tx, txData.Signature)
	if err != nil {
		return nil, err
	}
	txData.Tx = txWithSignature

	txData.SentHash, err = transactor.SendTransactionWithSignature(txData.TxArgs, txWithSignature)
	if err != nil {
		return nil, err
	}

	if txWithSignature.To() == nil {
		toAddr := crypto2.CreateAddress(txData.TxArgs.From, txData.Tx.Nonce())
		txData.TxArgs.To = &toAddr
	}

	return responses.NewRouterSentTransaction(txData.TxArgs, txData.SentHash, isApproval), nil
}

func (tm *TransactionManager) SendRouterTransactions(ctx context.Context) (transactions []*responses.RouterSentTransaction, fromChainID uint64, toChainID uint64, err error) {
	transactions = make([]*responses.RouterSentTransaction, 0)

	// send transactions
	for _, desc := range tm.routerTransactions {
		fromChainID = desc.RouterPath.FromChain.ChainID
		toChainID = desc.RouterPath.ToChain.ChainID
		if desc.ApprovalTxData != nil && !desc.IsApprovalPlaced() {
			var response *responses.RouterSentTransaction
			response, err = addSignatureAndSendTransaction(tm.transactor, desc.ApprovalTxData, true)
			if err != nil {
				return
			}

			transactions = append(transactions, response)

			// if approval is needed for swap, then we need to wait for the approval tx to be mined before sending the swap tx
			if walletCommon.IsProcessorSwap(desc.RouterPath.ProcessorName) {
				continue
			}
		}

		if desc.TxData != nil && !desc.IsTxPlaced() {
			var response *responses.RouterSentTransaction
			response, err = addSignatureAndSendTransaction(tm.transactor, desc.TxData, false)
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
