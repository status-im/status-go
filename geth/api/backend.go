package api

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
)

// StatusBackend implements Status.im service
type StatusBackend struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	txQueueManager common.TxQueueManager
	jailManager    common.JailManager
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized")

	nodeManager := node.NewNodeManager()
	accountManager := node.NewAccountManager(nodeManager)
	return &StatusBackend{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		txQueueManager: node.NewTxQueueManager(nodeManager, accountManager),
		jailManager:    jail.New(nodeManager),
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

// IsNodeRunning confirm that node is running
func (m *StatusBackend) IsNodeRunning() bool {
	return m.nodeManager.IsNodeRunning()
}

// StartNode start Status node, fails if node is already started
func (m *StatusBackend) StartNode(config *params.NodeConfig) (<-chan struct{}, error) {
	backendReady := make(chan struct{})
	nodeStarted, err := m.nodeManager.StartNode(config)
	if err != nil {
		return nil, err
	}

	go m.onNodeStart(backendReady, nodeStarted)
	return backendReady, err
}

func (m *StatusBackend) onNodeStart(backendReady chan struct{}, nodeStarted <-chan struct{}) {
	defer close(backendReady)
	<-nodeStarted

	if err := m.registerHandlers(); err != nil {
		log.Error("Handler registration failed", "err", err)
	}
}

// RestartNode restart running Status node, fails if node is not running
func (m *StatusBackend) RestartNode() (<-chan struct{}, error) {
	backendReady := make(chan struct{})
	nodeRestarted, err := m.nodeManager.RestartNode()
	if err != nil {
		return nil, err
	}

	go m.onNodeStart(backendReady, nodeRestarted)
	return backendReady, err
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (m *StatusBackend) StopNode() error {
	return m.nodeManager.StopNode()
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (m *StatusBackend) ResetChainData() (<-chan struct{}, error) {
	backendReady := make(chan struct{})
	nodeRestarted, err := m.nodeManager.ResetChainData()
	if err != nil {
		return nil, err
	}

	go m.onNodeStart(backendReady, nodeRestarted)
	return backendReady, err
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *StatusBackend) CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	return m.accountManager.CreateAccount(password)
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (m *StatusBackend) CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	return m.accountManager.CreateChildAccount(parentAddress, password)
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (m *StatusBackend) RecoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	return m.accountManager.RecoverAccount(password, mnemonic)
}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (m *StatusBackend) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	return m.accountManager.VerifyAccountPassword(keyStoreDir, address, password)
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (m *StatusBackend) SelectAccount(address, password string) error {
	return m.accountManager.SelectAccount(address, password)
}

// ReSelectAccount selects previously selected account, often, after node restart.
func (m *StatusBackend) ReSelectAccount() error {
	return m.accountManager.ReSelectAccount()
}

// Logout clears whisper identities
func (m *StatusBackend) Logout() error {
	return m.accountManager.Logout()
}

// SelectedAccount returns currently selected account
func (m *StatusBackend) SelectedAccount() (*common.SelectedExtKey, error) {
	return m.accountManager.SelectedAccount()
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (m *StatusBackend) CompleteTransaction(id, password string) (gethcommon.Hash, error) {
	return m.txQueueManager.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (m *StatusBackend) CompleteTransactions(ids, password string) map[string]common.RawCompleteTransactionResult {
	return m.txQueueManager.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (m *StatusBackend) DiscardTransaction(id string) error {
	return m.txQueueManager.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (m *StatusBackend) DiscardTransactions(ids string) map[string]common.RawDiscardTransactionResult {
	return m.txQueueManager.DiscardTransactions(ids)
}

// registerHandlers attaches Status callback handlers to running node
func (m *StatusBackend) registerHandlers() error {
	runningNode, err := m.nodeManager.Node()
	if err != nil {
		return err
	}

	var lightEthereum *les.LightEthereum
	if err := runningNode.Service(&lightEthereum); err != nil {
		log.Error("Cannot get light ethereum service", "error", err)
	}

	lightEthereum.StatusBackend.SetAccountsFilterHandler(m.accountManager.AccountsListRequestHandler())
	log.Info("Registered handler", "fn", "AccountsFilterHandler")

	lightEthereum.StatusBackend.SetTransactionQueueHandler(m.txQueueManager.TransactionQueueHandler())
	log.Info("Registered handler", "fn", "TransactionQueueHandler")

	lightEthereum.StatusBackend.SetTransactionReturnHandler(m.txQueueManager.TransactionReturnHandler())
	log.Info("Registered handler", "fn", "TransactionReturnHandler")

	m.ReSelectAccount()
	log.Info("Account reselected")

	node.SendSignal(node.SignalEnvelope{
		Type:  node.EventNodeReady,
		Event: struct{}{},
	})

	return nil
}
