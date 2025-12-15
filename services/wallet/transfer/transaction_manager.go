package transfer

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type TransactionManager struct {
	gethManager    *accsmanagement.AccountsManager
	transactor     transactions.TransactorIface
	config         *params.NodeConfig
	accountsDB     accounts.AccountsStorage
	pendingTracker *pendingtxtracker.PendingTxTracker
	eventFeed      *event.Feed

	// used in a new approach
	routerTransactions []*wallettypes.RouterTransactionDetails
}

func NewTransactionManager(
	gethManager *accsmanagement.AccountsManager,
	transactor transactions.TransactorIface,
	config *params.NodeConfig,
	accountsDB accounts.AccountsStorage,
	pendingTxManager *pendingtxtracker.PendingTxTracker,
	eventFeed *event.Feed,
) *TransactionManager {
	return &TransactionManager{
		gethManager:    gethManager,
		transactor:     transactor,
		config:         config,
		accountsDB:     accountsDB,
		pendingTracker: pendingTxManager,
		eventFeed:      eventFeed,
	}
}

var (
	emptyHash = common.Hash{}
)

type TxResponse struct {
	KeyUID        string                 `json:"keyUid,omitempty"`
	Address       types.Address          `json:"address,omitempty"`
	AddressPath   string                 `json:"addressPath,omitempty"`
	SignOnKeycard bool                   `json:"signOnKeycard,omitempty"`
	ChainID       uint64                 `json:"chainId,omitempty"`
	MessageToSign interface{}            `json:"messageToSign,omitempty"`
	TxArgs        wallettypes.SendTxArgs `json:"txArgs,omitempty"`
	RawTx         string                 `json:"rawTx,omitempty"`
	TxHash        common.Hash            `json:"txHash,omitempty"`
}

func (tm *TransactionManager) SignMessage(message types.HexBytes, privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("account or private key is nil")
	}

	signature, err := crypto.Sign(message[:], privateKey)

	return types.EncodeHex(signature), err
}

func (tm *TransactionManager) BuildTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs) (response *TxResponse, err error) {
	account, err := tm.accountsDB.GetAccountByAddress(sendArgs.From)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve account: %w", err)
	}

	kp, err := tm.accountsDB.GetKeypairByKeyUID(account.KeyUID)
	if err != nil {
		return nil, err
	}

	txBeingSigned, _, err := tm.transactor.ValidateAndBuildTransaction(chainID, sendArgs, -1)
	if err != nil {
		return nil, err
	}

	// Set potential missing fields that were added while building the transaction
	if sendArgs.Value == nil {
		value := hexutil.Big(*txBeingSigned.Value())
		sendArgs.Value = &value
	}
	if sendArgs.Nonce == nil {
		nonce := hexutil.Uint64(txBeingSigned.Nonce())
		sendArgs.Nonce = &nonce
	}
	if sendArgs.Gas == nil {
		gas := hexutil.Uint64(txBeingSigned.Gas())
		sendArgs.Gas = &gas
	}
	if sendArgs.GasPrice == nil {
		gasPrice := hexutil.Big(*txBeingSigned.GasPrice())
		sendArgs.GasPrice = &gasPrice
	}

	if sendArgs.IsDynamicFeeTx() {
		if sendArgs.MaxPriorityFeePerGas == nil {
			maxPriorityFeePerGas := hexutil.Big(*txBeingSigned.GasTipCap())
			sendArgs.MaxPriorityFeePerGas = &maxPriorityFeePerGas
		}
		if sendArgs.MaxFeePerGas == nil {
			maxFeePerGas := hexutil.Big(*txBeingSigned.GasFeeCap())
			sendArgs.MaxFeePerGas = &maxFeePerGas
		}
	}

	signer := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))

	return &TxResponse{
		KeyUID:        account.KeyUID,
		Address:       account.Address,
		AddressPath:   account.Path,
		SignOnKeycard: kp.MigratedToKeycard(),
		ChainID:       chainID,
		MessageToSign: signer.Hash(txBeingSigned),
		TxArgs:        sendArgs,
	}, nil
}

func (tm *TransactionManager) BuildRawTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs, signature []byte) (response *TxResponse, err error) {
	tx, err := tm.transactor.BuildTransactionWithSignature(chainID, sendArgs, signature)
	if err != nil {
		return nil, err
	}

	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &TxResponse{
		ChainID: chainID,
		TxArgs:  sendArgs,
		RawTx:   types.EncodeHex(data),
		TxHash:  tx.Hash(),
	}, nil
}

func (tm *TransactionManager) SendTransactionWithSignature(chainID uint64, sendArgs wallettypes.SendTxArgs, signature []byte) (types.Hash, error) {
	txWithSignature, err := tm.transactor.BuildTransactionWithSignature(chainID, sendArgs, signature)
	if err != nil {
		return types.Hash{}, err
	}

	return tm.transactor.SendTransactionWithSignature(&sendArgs, txWithSignature)
}
