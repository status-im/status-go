package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
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

// StatusBackend implements Status.im service
type StatusBackend struct {
	mu              sync.Mutex
	nodeManager     *node.NodeManager
	accountManager  account.Accounter
	txQueue         *transactions.Manager
	jail            jail.Manager
	connectionState ConnectionState
	log             log.Logger
	newNotification fcm.NotificationConstructor
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")

	backend := StatusBackend{
		nodeManager:     node.NewNodeManager(),
		newNotification: fcm.NewNotification(fcmServerKey),
		log:             log.New("package", "status-go/geth/api.StatusBackend"),
	}
	backend.accountManager = account.NewManager(&backend)

	return &backend
}

// NodeManager returns reference to node manager
func (b *StatusBackend) NodeManager() *node.NodeManager {
	return b.nodeManager
}

// Account get the underlying accounts.Manager under account.Manager
func (b *StatusBackend) Account() (*accounts.Manager, error) {
	node, err := b.nodeManager.Node()
	if err != nil {
		return nil, err
	}
	return node.AccountManager(), nil
}

// AccountKeyStore exposes reference to accounts key store
func (b *StatusBackend) AccountKeyStore() (*keystore.KeyStore, error) {
	am, err := b.Account()
	if err != nil {
		return nil, err
	}

	backends := am.Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		return nil, account.ErrAccountKeyStoreMissing
	}

	keyStore, ok := backends[0].(*keystore.KeyStore)
	if !ok {
		return nil, account.ErrAccountKeyStoreMissing
	}

	return keyStore, nil
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() account.Accounter {
	return b.accountManager
}

// JailManager returns reference to jail
func (b *StatusBackend) JailManager() jail.Manager {
	return b.jail
}

// TxQueueManager returns reference to transactions manager
func (b *StatusBackend) TxQueueManager() *transactions.Manager {
	return b.txQueue
}

// Whisper gets a WhisperService
func (b *StatusBackend) Whisper() (*whisper.Whisper, error) {
	return b.nodeManager.WhisperService()
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

	b.txQueue = transactions.NewManager(b.nodeManager, b.accountManager)
	b.jail = jail.New(b.nodeManager)

	signal.Send(signal.Envelope{Type: signal.EventNodeStarted})
	// tx queue manager should be started after node is started, it depends
	// on rpc client being created
	b.txQueue.Start()
	if err := b.registerHandlers(); err != nil {
		b.log.Error("Handler registration failed", "err", err)
	}
	if err := b.ReselectAccount(); err != nil {
		b.log.Error("Reselect account failed", "err", err)
	}

	b.log.Info("Account reselected")
	signal.Send(signal.Envelope{Type: signal.EventNodeReady})

	return nil
}

// ReselectAccount reselects the previous account if any
func (b *StatusBackend) ReselectAccount() error {
	selectedAccount, err := b.AccountManager().SelectedAccount()
	if err != nil {
		return nil
	}

	w, err := b.Whisper()
	if err != nil {
		return fmt.Errorf("%s: %v", account.ErrWhisperClearIdentitiesFailure, err)
	}

	err = w.SelectKeyPair(selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return account.ErrWhisperIdentityInjectionFailure
	}

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
	b.txQueue.Stop()
	b.jail.Stop()
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
func (b *StatusBackend) SendTransaction(ctx context.Context, args common.SendTxArgs) (hash gethcommon.Hash, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx := common.CreateTransaction(ctx, args)
	if err = b.txQueue.QueueTransaction(tx); err != nil {
		return hash, err
	}
	rst := b.txQueue.WaitForTransaction(tx)
	if rst.Error != nil {
		return hash, rst.Error
	}
	return rst.Hash, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (b *StatusBackend) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return b.txQueue.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (b *StatusBackend) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.TransactionResult {
	return b.txQueue.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (b *StatusBackend) DiscardTransaction(id common.QueuedTxID) error {
	return b.txQueue.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (b *StatusBackend) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	return b.txQueue.DiscardTransactions(ids)
}

// registerHandlers attaches Status callback handlers to running node
func (b *StatusBackend) registerHandlers() error {
	rpcClient := b.nodeManager.RPCClient()
	if rpcClient == nil {
		return node.ErrRPCClient
	}

	rpcClient.RegisterHandler("eth_accounts", func(context.Context, ...interface{}) (interface{}, error) {
		return b.AccountManager().Accounts()
	})
	rpcClient.RegisterHandler("eth_sendTransaction", b.txQueue.SendTransactionRPCHandler)
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

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *StatusBackend) SelectAccount(address, password string) error {
	if err := b.AccountManager().SelectAccount(address, password); err != nil {
		return err
	}
	if err := b.ReselectAccount(); err != nil {
		return err
	}

	return nil
}

// Logout clears whisper identities
func (b *StatusBackend) Logout() error {
	if err := b.DeleteWhisperKeyPairs(); err != nil {
		return err
	}

	return b.AccountManager().Logout()
}

// SetNodeManager Node manager setter
func (b *StatusBackend) SetNodeManager(nodeManager *node.NodeManager) {
	b.nodeManager = nodeManager
}

// DeleteWhisperKeyPairs removes all cryptographic identities known to the node
func (b *StatusBackend) DeleteWhisperKeyPairs() error {
	w, err := b.Whisper()

	if w == nil || err != nil {
		return account.ErrWhisperIdentityInjectionFailure
	}

	if err := w.DeleteKeyPairs(); err != nil {
		return fmt.Errorf("%s: %v", account.ErrWhisperClearIdentitiesFailure, err)
	}
	return nil
}
