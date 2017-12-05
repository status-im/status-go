package main

import (
	"context"

	fcm "github.com/NaySoftware/go-fcm"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

type StatusAPI interface {
	NodeManager() common.NodeManager
	AccountManager() common.AccountManager
	JailManager() common.JailManager
	TxQueueManager() common.TxQueueManager
	StartNode(config *params.NodeConfig) error
	StartNodeAsync(config *params.NodeConfig) (<-chan struct{}, error)
	StopNode() error
	StopNodeAsync() (<-chan struct{}, error)
	RestartNode() error
	RestartNodeAsync() (<-chan struct{}, error)
	ResetChainData() error
	ResetChainDataAsync() (<-chan struct{}, error)
	CallRPC(inputJSON string) string
	CreateAccount(password string) common.AccountInfo
	CreateChildAccount(parentAddress, password string) common.AccountInfo
	RecoverAccount(password, mnemonic string) common.AccountInfo
	VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error)
	SelectAccount(address, password string) error
	Logout() error
	SendTransaction(ctx context.Context, args common.SendTxArgs) (gethcommon.Hash, error)
	CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error)
	CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.RawCompleteTransactionResult
	DiscardTransaction(id common.QueuedTxID) error
	DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult
	JailParse(chatID string, js string) string
	CreateAndInitCell(chatID, js string) string
	JailCall(chatID, this, args string) string
	JailExecute(chatID, code string) string
	SetJailBaseJS(js string)
	Notify(token string) string
	NotifyUsers(message string, payload fcm.NotificationPayload, tokens ...string) error
	ValidateJSONConfig(configJSON string) common.APIDetailedResponse
}
