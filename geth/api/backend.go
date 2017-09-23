package api

import (
	"context"
	"sync"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/txqueue"
)

// StatusBackend implements Status.im service
type StatusBackend struct {
	sync.Mutex
	nodeReady      chan struct{} // channel to wait for when node is fully ready
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	txQueueManager common.TxQueueManager
	jailManager    common.JailManager
	// TODO(oskarth): notifer here
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")

	nodeManager := node.NewNodeManager()
	accountManager := account.NewManager(nodeManager)
	txQueueManager := txqueue.NewManager(nodeManager, accountManager)
	jailManager := jail.New(nodeManager)

	return &StatusBackend{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		jailManager:    jailManager,
		txQueueManager: txQueueManager,
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

// TxQueueManager returns reference to jail
func (m *StatusBackend) TxQueueManager() common.TxQueueManager {
	return m.txQueueManager
}

// IsNodeRunning confirm that node is running
func (m *StatusBackend) IsNodeRunning() bool {
	return m.nodeManager.IsNodeRunning()
}

// StartNode start Status node, fails if node is already started
func (m *StatusBackend) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if m.nodeReady != nil {
		return nil, node.ErrNodeExists
	}

	nodeStarted, err := m.nodeManager.StartNode(config)
	if err != nil {
		return nil, err
	}

	m.txQueueManager.Start()

	m.nodeReady = make(chan struct{}, 1)
	go m.onNodeStart(nodeStarted, m.nodeReady) // waits on nodeStarted, writes to backendReady

	return m.nodeReady, err
}

// onNodeStart does everything required to prepare backend
func (m *StatusBackend) onNodeStart(nodeStarted <-chan struct{}, backendReady chan struct{}) {
	<-nodeStarted

	if err := m.registerHandlers(); err != nil {
		log.Error("Handler registration failed", "err", err)
	}

	m.accountManager.ReSelectAccount()
	log.Info("Account reselected")

	close(backendReady)
	signal.Send(signal.Envelope{
		Type:  signal.EventNodeReady,
		Event: struct{}{},
	})
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *StatusBackend) StopNode() (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if m.nodeReady == nil {
		return nil, node.ErrNoRunningNode
	}
	<-m.nodeReady

	nodeStopped, err := m.nodeManager.StopNode()
	if err != nil {
		return nil, err
	}

	m.txQueueManager.Stop()

	backendStopped := make(chan struct{}, 1)
	go func() {
		<-nodeStopped
		m.Lock()
		m.nodeReady = nil
		m.Unlock()
		close(backendStopped)
	}()

	return backendStopped, nil
}

// RestartNode restart running Status node, fails if node is not running
func (m *StatusBackend) RestartNode() (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if m.nodeReady == nil {
		return nil, node.ErrNoRunningNode
	}
	<-m.nodeReady

	nodeRestarted, err := m.nodeManager.RestartNode()
	if err != nil {
		return nil, err
	}

	m.nodeReady = make(chan struct{}, 1)
	go m.onNodeStart(nodeRestarted, m.nodeReady) // waits on nodeRestarted, writes to backendReady

	return m.nodeReady, err
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *StatusBackend) ResetChainData() (<-chan struct{}, error) {
	m.Lock()
	defer m.Unlock()

	if m.nodeReady == nil {
		return nil, node.ErrNoRunningNode
	}
	<-m.nodeReady

	nodeReset, err := m.nodeManager.ResetChainData()
	if err != nil {
		return nil, err
	}

	m.nodeReady = make(chan struct{}, 1)
	go m.onNodeStart(nodeReset, m.nodeReady) // waits on nodeReset, writes to backendReady

	return m.nodeReady, err
}

// CallRPC executes RPC request on node's in-proc RPC server
func (m *StatusBackend) CallRPC(inputJSON string) string {
	client := m.nodeManager.RPCClient()
	return client.CallRaw(inputJSON)
}

// SendTransaction creates a new transaction and waits until it's complete.
func (m *StatusBackend) SendTransaction(ctx context.Context, args common.SendTxArgs) (gethcommon.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tx := m.txQueueManager.CreateTransaction(ctx, args)

	if err := m.txQueueManager.QueueTransaction(tx); err != nil {
		return gethcommon.Hash{}, err
	}

	if err := m.txQueueManager.WaitForTransaction(tx); err != nil {
		return gethcommon.Hash{}, err
	}

	return tx.Hash, nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (m *StatusBackend) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return m.txQueueManager.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (m *StatusBackend) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.RawCompleteTransactionResult {
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
	rpcClient.RegisterHandler("eth_accounts", m.accountManager.AccountsRPCHandler())
	rpcClient.RegisterHandler("eth_sendTransaction", m.txQueueManager.SendTransactionRPCHandler)

	m.txQueueManager.SetTransactionQueueHandler(m.txQueueManager.TransactionQueueHandler())
	log.Info("Registered handler", "fn", "TransactionQueueHandler")

	m.txQueueManager.SetTransactionReturnHandler(m.txQueueManager.TransactionReturnHandler())
	log.Info("Registered handler", "fn", "TransactionReturnHandler")

	return nil
}
