package api

import (
	"context"
	"fmt"
	"sync"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/notifications/push/fcm"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/provider"
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
	connectionState ConnectionState
	Provider        *provider.ServiceProvider
	newNotification common.NotificationConstructor
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")
	p := provider.New(node.NewNodeManager())

	backend := StatusBackend{
		Provider:        p,
		newNotification: fcm.NewNotification(fcmServerKey),
	}

	return &backend
}

// NodeManager returns reference to node manager
func (b *StatusBackend) NodeManager() *node.NodeManager {
	return b.Provider.NodeManager()
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() common.AccountManager {
	am, err := b.Provider.AccountManager()
	if err != nil {
		log.Warn(err.Error())
	}
	return am
}

// JailManager returns reference to jail
func (b *StatusBackend) JailManager() jail.Manager {
	return b.Provider.JailManager()
}

// TxQueueManager returns reference to transactions manager
func (b *StatusBackend) TxQueueManager() *transactions.Manager {
	return b.Provider.TxQueueManager()
}

// IsNodeRunning confirm that node is running
func (b *StatusBackend) IsNodeRunning() bool {
	return b.NodeManager().IsNodeRunning()
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
	err = b.NodeManager().StartNode(config)
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
	b.TxQueueManager().Start()
	if err := b.registerHandlers(); err != nil {
		log.Error("Handler registration failed", "err", err)
	}
	if err := b.DeleteWhisperKeyPairs(); err != nil {
		return err
	}

	if err := b.ReselectAccount(); err != nil {
		log.Error("Reselect account failed", "err", err)
	}

	log.Info("Account reselected")
	signal.Send(signal.Envelope{Type: signal.EventNodeReady})
	return nil
}

// ReselectAccount reselects the previous account if any
func (b *StatusBackend) ReselectAccount() error {
	if err := b.AccountManager().ReSelectAccount(); err != nil {
		return err
	}
	if err := b.selectKeyPair(); err != nil {
		return err
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
	b.TxQueueManager().Stop()
	b.JailManager().Stop()
	defer signal.Send(signal.Envelope{Type: signal.EventNodeStopped})
	b.Provider.Reset()
	return b.NodeManager().StopNode()
}

// RestartNode restart running Status node, fails if node is not running
func (b *StatusBackend) RestartNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	config, err := b.NodeManager().NodeConfig()
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
	config, err := b.NodeManager().NodeConfig()
	if err != nil {
		return err
	}
	newcfg := *config
	if err := b.stopNode(); err != nil {
		return err
	}
	// config is cleaned when node is stopped
	if err := b.NodeManager().ResetChainData(&newcfg); err != nil {
		return err
	}
	signal.Send(signal.Envelope{Type: signal.EventChainDataRemoved})
	return b.startNode(&newcfg)
}

// CallRPC executes RPC request on node's in-proc RPC server
func (b *StatusBackend) CallRPC(inputJSON string) string {
	client := b.NodeManager().RPCClient()
	return client.CallRaw(inputJSON)
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(ctx context.Context, args common.SendTxArgs) (hash gethcommon.Hash, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx := common.CreateTransaction(ctx, args)
	if err = b.TxQueueManager().QueueTransaction(tx); err != nil {
		return hash, err
	}
	rst := b.TxQueueManager().WaitForTransaction(tx)
	if rst.Error != nil {
		return hash, rst.Error
	}
	return rst.Hash, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (b *StatusBackend) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return b.TxQueueManager().CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (b *StatusBackend) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.TransactionResult {
	return b.TxQueueManager().CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (b *StatusBackend) DiscardTransaction(id common.QueuedTxID) error {
	return b.TxQueueManager().DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (b *StatusBackend) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	return b.TxQueueManager().DiscardTransactions(ids)
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
	rpcClient.RegisterHandler("eth_sendTransaction", b.TxQueueManager().SendTransactionRPCHandler)
	return nil
}

// ConnectionChange handles network state changes logic.
func (b *StatusBackend) ConnectionChange(state ConnectionState) {
	log.Info("Network state change", "old", b.connectionState, "new", state)
	b.connectionState = state

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}

// AppStateChange handles app state changes (background/foreground).
func (b *StatusBackend) AppStateChange(state AppState) {
	log.Info("App State changed: %s", state)

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
	if err := b.selectKeyPair(); err != nil {
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

// DeleteWhisperKeyPairs removes all cryptographic identities known to the node
func (b *StatusBackend) DeleteWhisperKeyPairs() error {
	w, err := b.Provider.Whisper()
	if w == nil || err != nil {
		return account.ErrWhisperIdentityInjectionFailure
	}

	if err := w.DeleteKeyPairs(); err != nil {
		return fmt.Errorf("%s: %v", account.ErrWhisperClearIdentitiesFailure, err)
	}
	return nil
}

// SelectKeyPair adds the default account cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (b *StatusBackend) selectKeyPair() error {
	selectedAccount, err := b.AccountManager().SelectedAccount()
	if err != nil {
		return err
	}

	w, err := b.Provider.Whisper()
	if err != nil {
		return fmt.Errorf("%s: %v", account.ErrWhisperClearIdentitiesFailure, err)
	}

	err = w.SelectKeyPair(selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return account.ErrWhisperIdentityInjectionFailure
	}
	return nil
}
