package common

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

// errors
var (
	ErrDeprecatedMethod = errors.New("Method is depricated and will be removed in future release")
)

// SelectedExtKey is a container for currently selected (logged in) account
type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *keystore.Key
	SubAccounts []accounts.Account
}

// Hex dumps address of a given extended key as hex string
func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}

// NodeManager defines expected methods for managing Status node
type NodeManager interface {
	// StartNode start Status node, fails if node is already started
	StartNode(config *params.NodeConfig) (<-chan struct{}, error)

	// StopNode stop the running Status node.
	// Stopped node cannot be resumed, one starts a new node instead.
	StopNode() (<-chan struct{}, error)

	// RestartNode restart running Status node, fails if node is not running
	RestartNode() (<-chan struct{}, error)

	// ResetChainData remove chain data from data directory.
	// Node is stopped, and new node is started, with clean data directory.
	ResetChainData() (<-chan struct{}, error)

	// IsNodeRunning confirm that node is running
	IsNodeRunning() bool

	// NodeConfig returns reference to running node's configuration
	NodeConfig() (*params.NodeConfig, error)

	// Node returns underlying Status node
	Node() (*node.Node, error)

	// PopulateStaticPeers populates node's list of static bootstrap peers
	PopulateStaticPeers() error

	// AddPeer adds URL of static peer
	AddPeer(url string) error

	// LightEthereumService exposes reference to LES service running on top of the node
	LightEthereumService() (*les.LightEthereum, error)

	// WhisperService returns reference to running Whisper service
	WhisperService() (*whisper.Whisper, error)

	// AccountManager returns reference to node's account manager
	AccountManager() (*accounts.Manager, error)

	// AccountKeyStore returns reference to account manager's keystore
	AccountKeyStore() (*keystore.KeyStore, error)

	// RPCClient exposes reference to RPC client connected to the running node
	RPCClient() (*rpc.Client, error)

	// RPCServer exposes reference to running node's in-proc RPC server/handler
	RPCServer() (*rpc.Server, error)
}

// AccountManager defines expected methods for managing Status accounts
type AccountManager interface {
	// CreateAccount creates an internal geth account
	// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
	// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
	// sub-account derivations)
	CreateAccount(password string) (address, pubKey, mnemonic string, err error)

	// CreateChildAccount creates sub-account for an account identified by parent address.
	// CKD#2 is used as root for master accounts (when parentAddress is "").
	// Otherwise (when parentAddress != ""), child is derived directly from parent.
	CreateChildAccount(parentAddress, password string) (address, pubKey string, err error)

	// RecoverAccount re-creates master key using given details.
	// Once master key is re-generated, it is inserted into keystore (if not already there).
	RecoverAccount(password, mnemonic string) (address, pubKey string, err error)

	// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
	// If no error is returned, then account is considered verified.
	VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error)

	// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
	// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
	// all previous identities are removed).
	SelectAccount(address, password string) error

	// ReSelectAccount selects previously selected account, often, after node restart.
	ReSelectAccount() error

	// SelectedAccount returns currently selected account
	SelectedAccount() (*SelectedExtKey, error)

	// Logout clears whisper identities
	Logout() error

	// AccountsListRequestHandler returns handler to process account list request
	AccountsListRequestHandler() func(entities []common.Address) []common.Address

	// AddressToDecryptedAccount tries to load decrypted key for a given account.
	// The running node, has a keystore directory which is loaded on start. Key file
	// for a given address is expected to be in that directory prior to node start.
	AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error)
}

// RPCManager defines expected methods for managing RPC client/server
type RPCManager interface {
	// Call executes RPC request on node's in-proc RPC server
	Call(inputJSON string) string
}

// RawCompleteTransactionResult is a JSON returned from transaction complete function (used internally)
type RawCompleteTransactionResult struct {
	Hash  common.Hash
	Error error
}

// RawDiscardTransactionResult is list of results from CompleteTransactions() (used internally)
type RawDiscardTransactionResult struct {
	Error error
}

// TxQueueManager defines expected methods for managing transaction queue
type TxQueueManager interface {
	// TransactionQueueHandler returns handler that processes incoming tx queue requests
	TransactionQueueHandler() func(queuedTx status.QueuedTx)

	// TransactionReturnHandler returns handler that processes responses from internal tx manager
	TransactionReturnHandler() func(queuedTx *status.QueuedTx, err error)

	// CompleteTransaction instructs backend to complete sending of a given transaction
	CompleteTransaction(id, password string) (common.Hash, error)

	// CompleteTransactions instructs backend to complete sending of multiple transactions
	CompleteTransactions(ids, password string) map[string]RawCompleteTransactionResult

	// DiscardTransaction discards a given transaction from transaction queue
	DiscardTransaction(id string) error

	// DiscardTransactions discards given multiple transactions from transaction queue
	DiscardTransactions(ids string) map[string]RawDiscardTransactionResult
}

// JailCell represents single jail cell, which is basically a JavaScript VM.
type JailCell interface {
	Set(string, interface{}) error
	Get(string) (otto.Value, error)
	Run(string) (otto.Value, error)
	RunOnLoop(string) (otto.Value, error)
}

// JailManager defines methods for managing jailed environments
type JailManager interface {
	// Parse creates a new jail cell context, with the given chatID as identifier.
	// New context executes provided JavaScript code, right after the initialization.
	Parse(chatID string, js string) string

	// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
	// Jail cell is clonned before call is executed i.e. all calls execute w/i their own contexts.
	Call(chatID string, path string, args string) string

	// NewJailCell initializes and returns jail cell
	NewJailCell(id string) (JailCell, error)

	// GetJailCell returns instance of JailCell (which is persisted w/i jail cell) by chatID
	GetJailCell(chatID string) (JailCell, error)

	// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse()
	BaseJS(js string)
}

// APIResponse generic response from API
type APIResponse struct {
	Error string `json:"error"`
}

// AccountInfo represents account's info
type AccountInfo struct {
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
	Error    string `json:"error"`
}

// CompleteTransactionResult is a JSON returned from transaction complete function (used in exposed method)
type CompleteTransactionResult struct {
	ID    string `json:"id"`
	Hash  string `json:"hash"`
	Error string `json:"error"`
}

// CompleteTransactionsResult is list of results from CompleteTransactions() (used in exposed method)
type CompleteTransactionsResult struct {
	Results map[string]CompleteTransactionResult `json:"results"`
}

// DiscardTransactionResult is a JSON returned from transaction discard function
type DiscardTransactionResult struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// DiscardTransactionsResult is a list of results from DiscardTransactions()
type DiscardTransactionsResult struct {
	Results map[string]DiscardTransactionResult `json:"results"`
}

// TestConfig contains shared (among different test packages) parameters
type TestConfig struct {
	Node struct {
		SyncSeconds time.Duration
		HTTPPort    int
		WSPort      int
	}
	Account1 struct {
		Address  string
		Password string
	}
	Account2 struct {
		Address  string
		Password string
	}
}

// LoadTestConfig loads test configuration values from disk
func LoadTestConfig() (*TestConfig, error) {
	var testConfig TestConfig

	configData := string(static.MustAsset("config/test-data.json"))
	if err := json.Unmarshal([]byte(configData), &testConfig); err != nil {
		return nil, err
	}

	return &testConfig, nil
}
