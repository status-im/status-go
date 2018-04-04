package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/notifications/push/fcm"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/transactions"
)

const (
	//todo(jeka): should be removed
	fcmServerKey = "AAAAxwa-r08:APA91bFtMIToDVKGAmVCm76iEXtA4dn9MPvLdYKIZqAlNpLJbd12EgdBI9DSDSXKdqvIAgLodepmRhGVaWvhxnXJzVpE6MoIRuKedDV3kfHSVBhWFqsyoLTwXY4xeufL9Sdzb581U-lx"
)

var (
	// ErrWhisperClearIdentitiesFailure clearing whisper identities has failed.
	ErrWhisperClearIdentitiesFailure = errors.New("failed to clear whisper identities")
	// ErrWhisperIdentityInjectionFailure injecting whisper identities has failed.
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
)

// StatusBackend implements Status.im service
type StatusBackend struct {
	mu              sync.Mutex
	nodeManager     *node.Manager
	accountManager  *account.Manager
	txQueueManager  *transactions.Manager
	jailManager     jail.Manager
	newNotification fcm.NotificationConstructor
	connectionState ConnectionState
	log             log.Logger
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")

	nodeManager := node.NewManager()
	accountManager := account.NewManager(nodeManager)
	txQueueManager := transactions.NewManager(nodeManager)
	jailManager := jail.New(nodeManager)
	notificationManager := fcm.NewNotification(fcmServerKey)

	return &StatusBackend{
		nodeManager:     nodeManager,
		accountManager:  accountManager,
		jailManager:     jailManager,
		txQueueManager:  txQueueManager,
		newNotification: notificationManager,
		log:             log.New("package", "status-go/geth/api.StatusBackend"),
	}
}

// NodeManager returns reference to node manager
func (b *StatusBackend) NodeManager() *node.Manager {
	return b.nodeManager
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() *account.Manager {
	return b.accountManager
}

// JailManager returns reference to jail
func (b *StatusBackend) JailManager() jail.Manager {
	return b.jailManager
}

// TxQueueManager returns reference to transactions manager
func (b *StatusBackend) TxQueueManager() *transactions.Manager {
	return b.txQueueManager
}

// IsNodeRunning confirm that node is running
func (b *StatusBackend) IsNodeRunning() bool {
	return b.nodeManager.IsNodeRunning()
}

// StartNode start Status node, fails if node is already started
func (b *StatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.startNode(config)
}

func (b *StatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
		}
	}()
	err = b.nodeManager.StartNode(config)
	if err != nil {
		switch err.(type) {
		case node.RPCClientError:
			err = fmt.Errorf("%v: %v", node.ErrRPCClient, err)
		case node.EthNodeError:
			err = fmt.Errorf("%v: %v", node.ErrNodeStartFailure, err)
		}
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: err,
			},
		})
		return err
	}
	signal.Send(signal.Envelope{Type: signal.EventNodeStarted})
	// tx queue manager should be started after node is started, it depends
	// on rpc client being created
	b.txQueueManager.Start(config.NetworkID)
	if err := b.registerHandlers(); err != nil {
		b.log.Error("Handler registration failed", "err", err)
	}
	if err := b.ReSelectAccount(); err != nil {
		b.log.Error("Reselect account failed", "err", err)
	}
	b.log.Info("Account reselected")
	signal.Send(signal.Envelope{Type: signal.EventNodeReady})
	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *StatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *StatusBackend) stopNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	b.txQueueManager.Stop()
	b.jailManager.Stop()
	defer signal.Send(signal.Envelope{Type: signal.EventNodeStopped})
	return b.nodeManager.StopNode()
}

// RestartNode restart running Status node, fails if node is not running
func (b *StatusBackend) RestartNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	config, err := b.nodeManager.NodeConfig()
	if err != nil {
		return err
	}
	newcfg := *config
	if err := b.stopNode(); err != nil {
		return err
	}
	return b.startNode(&newcfg)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (b *StatusBackend) ResetChainData() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, err := b.nodeManager.NodeConfig()
	if err != nil {
		return err
	}
	newcfg := *config
	if err := b.stopNode(); err != nil {
		return err
	}
	// config is cleaned when node is stopped
	if err := b.nodeManager.ResetChainData(&newcfg); err != nil {
		return err
	}
	signal.Send(signal.Envelope{Type: signal.EventChainDataRemoved})
	return b.startNode(&newcfg)
}

// CallRPC executes RPC request on node's in-proc RPC server
func (b *StatusBackend) CallRPC(inputJSON string) string {
	client := b.nodeManager.RPCClient()
	return client.CallRaw(inputJSON)
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(ctx context.Context, args transactions.SendTxArgs) (hash gethcommon.Hash, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx := transactions.Create(ctx, args)
	if err = b.txQueueManager.QueueTransaction(tx); err != nil {
		return hash, err
	}
	rst := b.txQueueManager.WaitForTransaction(tx)
	if rst.Error != nil {
		return hash, rst.Error
	}
	return rst.Hash, nil
}

func (b *StatusBackend) getVerifiedAccount(password string) (*account.SelectedExtKey, error) {
	selectedAccount, err := b.accountManager.SelectedAccount()
	if err != nil {
		b.log.Error("failed to get a selected account", "err", err)
		return nil, err
	}
	config, err := b.NodeManager().NodeConfig()
	if err != nil {
		return nil, err
	}
	_, err = b.accountManager.VerifyAccountPassword(config.KeyStoreDir, selectedAccount.Address.String(), password)
	if err != nil {
		b.log.Error("failed to verify account", "account", selectedAccount.Address.String(), "error", err)
		return nil, err
	}
	return selectedAccount, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (b *StatusBackend) CompleteTransaction(id string, password string) (hash gethcommon.Hash, err error) {
	selectedAccount, err := b.getVerifiedAccount(password)
	if err != nil {
		_ = b.txQueueManager.NotifyErrored(id, err)
		return hash, err
	}

	return b.txQueueManager.CompleteTransaction(id, selectedAccount)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (b *StatusBackend) CompleteTransactions(ids []string, password string) map[string]transactions.Result {
	results := make(map[string]transactions.Result)
	for _, txID := range ids {
		txHash, txErr := b.CompleteTransaction(txID, password)
		results[txID] = transactions.Result{
			Hash:  txHash,
			Error: txErr,
		}
	}
	return results
}

// DiscardTransaction discards a given transaction from transaction queue
func (b *StatusBackend) DiscardTransaction(id string) error {
	return b.txQueueManager.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (b *StatusBackend) DiscardTransactions(ids []string) map[string]error {
	results := make(map[string]error)
	for _, txID := range ids {
		err := b.DiscardTransaction(txID)
		if err != nil {
			results[txID] = err
		}
	}

	return results
}

// registerHandlers attaches Status callback handlers to running node
func (b *StatusBackend) registerHandlers() error {
	rpcClient := b.NodeManager().RPCClient()
	if rpcClient == nil {
		return node.ErrRPCClient
	}
	rpcClient.RegisterHandler("eth_accounts", func(context.Context, ...interface{}) (interface{}, error) {
		return b.AccountManager().Accounts()
	})
	rpcClient.RegisterHandler("eth_sendTransaction", b.txQueueManager.SendTransactionRPCHandler)
	return nil
}

// ConnectionChange handles network state changes logic.
func (b *StatusBackend) ConnectionChange(state ConnectionState) {
	b.log.Info("Network state change", "old", b.connectionState, "new", state)
	b.connectionState = state

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}

// AppStateChange handles app state changes (background/foreground).
func (b *StatusBackend) AppStateChange(state AppState) {
	b.log.Info("App State changed.", "new-state", state)

	// TODO: put node in low-power mode if the app is in background (or inactive)
	// and normal mode if the app is in foreground.
}

// Logout clears whisper identities.
func (b *StatusBackend) Logout() error {
	whisperService, err := b.nodeManager.WhisperService()
	if err != nil {
		return err
	}
	err = whisperService.DeleteKeyPairs()
	if err != nil {
		return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
	}

	return b.AccountManager().Logout()
}

// ReSelectAccount selects previously selected account, often, after node restart.
func (b *StatusBackend) ReSelectAccount() error {
	selectedAccount, err := b.AccountManager().SelectedAccount()
	if selectedAccount == nil || err == account.ErrNoAccountSelected {
		return nil
	}
	whisperService, err := b.nodeManager.WhisperService()
	if err != nil {
		return err
	}

	if err := whisperService.SelectKeyPair(selectedAccount.AccountKey.PrivateKey); err != nil {
		return ErrWhisperIdentityInjectionFailure
	}
	return nil
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *StatusBackend) SelectAccount(address, password string) error {
	err := b.accountManager.SelectAccount(address, password)
	if err != nil {
		return err
	}
	acc, err := b.accountManager.SelectedAccount()
	if err != nil {
		return err
	}

	whisperService, err := b.nodeManager.WhisperService()
	if err != nil {
		return err
	}

	err = whisperService.SelectKeyPair(acc.AccountKey.PrivateKey)
	if err != nil {
		return ErrWhisperIdentityInjectionFailure
	}

	return nil
}
