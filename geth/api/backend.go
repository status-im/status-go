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
	accountManager  common.AccountManager
	txQueueManager  *transactions.Manager
	jailManager     common.JailManager
	newNotification common.NotificationConstructor
	connectionState ConnectionState
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")

	nodeManager := node.NewNodeManager()
	accountManager := account.NewManager(nodeManager)
	txQueueManager := transactions.NewManager(nodeManager, accountManager)
	jailManager := jail.New(nodeManager)
	notificationManager := fcm.NewNotification(fcmServerKey)

	return &StatusBackend{
		nodeManager:     nodeManager,
		accountManager:  accountManager,
		jailManager:     jailManager,
		txQueueManager:  txQueueManager,
		newNotification: notificationManager,
	}
}

// NodeManager returns reference to node manager
func (b *StatusBackend) NodeManager() *node.NodeManager {
	return b.nodeManager
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() common.AccountManager {
	return b.accountManager
}

// JailManager returns reference to jail
func (b *StatusBackend) JailManager() common.JailManager {
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
			err = node.ErrRPCClient
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
	b.txQueueManager.Start()
	if err := b.registerHandlers(); err != nil {
		log.Error("Handler registration failed", "err", err)
	}
	if err := b.accountManager.ReSelectAccount(); err != nil {
		log.Error("Reselect account failed", "err", err)
	}
	log.Info("Account reselected")
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
func (b *StatusBackend) SendTransaction(ctx context.Context, args common.SendTxArgs) (hash gethcommon.Hash, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx := common.CreateTransaction(ctx, args)
	if err = b.txQueueManager.QueueTransaction(tx); err != nil {
		return hash, err
	}
	rst := b.txQueueManager.WaitForTransaction(tx)
	if rst.Error != nil {
		return hash, rst.Error
	}
	return rst.Hash, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (b *StatusBackend) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return b.txQueueManager.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (b *StatusBackend) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.TransactionResult {
	return b.txQueueManager.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (b *StatusBackend) DiscardTransaction(id common.QueuedTxID) error {
	return b.txQueueManager.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (b *StatusBackend) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	return b.txQueueManager.DiscardTransactions(ids)
}

// registerHandlers attaches Status callback handlers to running node
func (b *StatusBackend) registerHandlers() error {
	rpcClient := b.NodeManager().RPCClient()
	if rpcClient == nil {
		return node.ErrRPCClient
	}
	rpcClient.RegisterHandler("eth_accounts", b.accountManager.AccountsRPCHandler())
	rpcClient.RegisterHandler("eth_sendTransaction", b.txQueueManager.SendTransactionRPCHandler)
	return nil
}

// ConnectionChange handles network state changes logic.
func (b *StatusBackend) ConnectionChange(state ConnectionState) {
	log.Info("Network state change", "old", b.connectionState, "new", state)
	b.connectionState = state

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}
