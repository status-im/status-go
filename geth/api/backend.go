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
	"github.com/status-im/status-go/geth/notification/fcm"
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
func (m *StatusBackend) NodeManager() common.NodeManager {
	return m.nodeManager
}

// AccountManager returns reference to account manager
func (m *StatusBackend) AccountManager() common.AccountManager {
	return m.accountManager
}

// JailManager returns reference to jail
func (m *StatusBackend) JailManager() common.JailManager {
	return m.jailManager
}

// TxQueueManager returns reference to transactions manager
func (m *StatusBackend) TxQueueManager() *transactions.Manager {
	return m.txQueueManager
}

// IsNodeRunning confirm that node is running
func (m *StatusBackend) IsNodeRunning() bool {
	return m.nodeManager.IsNodeRunning()
}

// StartNode start Status node, fails if node is already started
func (m *StatusBackend) StartNode(config *params.NodeConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startNode(config)
}

func (m *StatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
		}
	}()
	err = m.nodeManager.StartNode(config)
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
	m.txQueueManager.Start()
	if err := m.registerHandlers(); err != nil {
		log.Error("Handler registration failed", "err", err)
	}
	if err := m.accountManager.ReSelectAccount(); err != nil {
		log.Error("Reselect account failed", "err", err)
	}
	log.Info("Account reselected")
	signal.Send(signal.Envelope{Type: signal.EventNodeReady})
	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *StatusBackend) StopNode() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopNode()
}

func (m *StatusBackend) stopNode() error {
	if !m.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	m.txQueueManager.Stop()
	m.jailManager.Stop()
	return m.nodeManager.StopNode()
}

// RestartNode restart running Status node, fails if node is not running
func (m *StatusBackend) RestartNode() error {
	if !m.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	config, err := m.nodeManager.NodeConfig()
	if err != nil {
		return err
	}
	newcfg := *config
	if err := m.stopNode(); err != nil {
		return err
	}
	return m.startNode(&newcfg)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *StatusBackend) ResetChainData() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	config, err := m.nodeManager.NodeConfig()
	if err != nil {
		return err
	}
	newcfg := *config
	if err := m.stopNode(); err != nil {
		return err
	}
	if err := m.ResetChainData(); err != nil {
		return err
	}
	signal.Send(signal.Envelope{Type: signal.EventChainDataRemoved})
	return m.startNode(&newcfg)
}

// CallRPC executes RPC request on node's in-proc RPC server
func (m *StatusBackend) CallRPC(inputJSON string) string {
	client := m.nodeManager.RPCClient()
	return client.CallRaw(inputJSON)
}

// SendTransaction creates a new transaction and waits until it's complete.
func (m *StatusBackend) SendTransaction(ctx context.Context, args common.SendTxArgs) (hash gethcommon.Hash, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx := common.CreateTransaction(ctx, args)
	if err = m.txQueueManager.QueueTransaction(tx); err != nil {
		return hash, err
	}
	rst := m.txQueueManager.WaitForTransaction(tx)
	if rst.Error != nil {
		return hash, rst.Error
	}
	return rst.Hash, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (m *StatusBackend) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return m.txQueueManager.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (m *StatusBackend) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.TransactionResult {
	return m.txQueueManager.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (m *StatusBackend) DiscardTransaction(id common.QueuedTxID) error {
	return m.txQueueManager.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (m *StatusBackend) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	return m.txQueueManager.DiscardTransactions(ids)
}

// registerHandlers attaches Status callback handlers to running node
func (m *StatusBackend) registerHandlers() error {
	rpcClient := m.NodeManager().RPCClient()
	if rpcClient == nil {
		return node.ErrRPCClient
	}
	rpcClient.RegisterHandler("eth_accounts", m.accountManager.AccountsRPCHandler())
	rpcClient.RegisterHandler("eth_sendTransaction", m.txQueueManager.SendTransactionRPCHandler)
	return nil
}
