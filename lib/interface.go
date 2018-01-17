// This is a file contains the interface that the package requires
package main

import (
	"context"

	fcm "github.com/NaySoftware/go-fcm"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

type libStatusAPI interface {
	jailAPI
	accountAPI
	transactionAPI
	nodeAPI
	notificationAPI
}

type jailAPI interface {
	JailCall(chatID, this, args string) string
	JailExecute(chatID, code string) string
	SetJailBaseJS(js string)
	JailParse(chatID string, js string) string
	CreateAndInitCell(chatID, js string) string
}

type accountAPI interface {
	CreateAccount(password string) (common.AccountInfo, error)
	CreateChildAccount(parentAddress, password string) (common.AccountInfo, error)
	RecoverAccount(password, mnemonic string) (common.AccountInfo, error)
	VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error)
	SelectAccount(address, password string) error
	Logout() error
}

type transactionAPI interface {
	CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error)
	CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.RawCompleteTransactionResult
	DiscardTransaction(id common.QueuedTxID) error
	DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult
	TxQueueManager() common.TxQueueManager
	SendTransaction(ctx context.Context, args common.SendTxArgs) (gethcommon.Hash, error)
}

type nodeAPI interface {
	NodeManager() common.NodeManager
	ValidateJSONConfig(configJSON string) common.APIDetailedResponse
	StartNodeAsync(config *params.NodeConfig) (<-chan struct{}, error)
	StopNodeAsync() (<-chan struct{}, error)
	RestartNodeAsync() (<-chan struct{}, error)
	ResetChainDataAsync() (<-chan struct{}, error)
	CallRPC(inputJSON string) string
}

type notificationAPI interface {
	Notify(token string) string
	NotifyUsers(message string, payload fcm.NotificationPayload, tokens ...string) error
}
