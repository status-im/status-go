package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/node"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
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
	RPCClient() *rpc.Client
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

	// Accounts returns handler to process account list request
	Accounts() ([]common.Address, error)

	// AccountsRPCHandler returns RPC wrapper for Accounts()
	AccountsRPCHandler() rpc.Handler

	// AddressToDecryptedAccount tries to load decrypted key for a given account.
	// The running node, has a keystore directory which is loaded on start. Key file
	// for a given address is expected to be in that directory prior to node start.
	AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error)
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

// QueuedTxID queued transaction identifier
type QueuedTxID string

// QueuedTx holds enough information to complete the queued transaction.
type QueuedTx struct {
	ID         QueuedTxID
	Hash       common.Hash
	Context    context.Context
	Args       SendTxArgs
	InProgress bool // true if transaction is being sent
	Done       chan struct{}
	Discard    chan struct{}
	Err        error
}

// SendTxArgs represents the arguments to submit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Big    `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
}

// EnqueuedTxHandler is a function that receives queued/pending transactions, when they get queued
type EnqueuedTxHandler func(*QueuedTx)

// EnqueuedTxReturnHandler is a function that receives response when tx is complete (both on success and error)
type EnqueuedTxReturnHandler func(*QueuedTx, error)

// TxQueue is a queue of transactions.
type TxQueue interface {
	// Remove removes a transaction from the queue.
	Remove(id QueuedTxID)

	// Reset resets the state of the queue.
	Reset()

	// Count returns a number of transactions in the queue.
	Count() int

	// Has returns true if a transaction is in the queue.
	Has(id QueuedTxID) bool
}

// TxQueueManager defines expected methods for managing transaction queue
type TxQueueManager interface {
	// Start starts accepting new transaction in the queue.
	Start()

	// Stop stops accepting new transactions in the queue.
	Stop()

	// TransactionQueue returns a transaction queue.
	TransactionQueue() TxQueue

	// CreateTransactoin creates a new transaction.
	CreateTransaction(ctx context.Context, args SendTxArgs) *QueuedTx

	// QueueTransaction adds a new transaction to the queue.
	QueueTransaction(tx *QueuedTx) error

	// WaitForTransactions blocks until transaction is completed, discarded or timed out.
	WaitForTransaction(tx *QueuedTx) error

	// NotifyOnQueuedTxReturn notifies a handler when a transaction returns.
	NotifyOnQueuedTxReturn(queuedTx *QueuedTx, err error)

	// TransactionQueueHandler returns handler that processes incoming tx queue requests
	TransactionQueueHandler() func(queuedTx *QueuedTx)

	// TODO(adam): might be not needed
	SetTransactionQueueHandler(fn EnqueuedTxHandler)

	// TODO(adam): might be not needed
	SetTransactionReturnHandler(fn EnqueuedTxReturnHandler)

	SendTransactionRPCHandler(ctx context.Context, args ...interface{}) (interface{}, error)

	// TransactionReturnHandler returns handler that processes responses from internal tx manager
	TransactionReturnHandler() func(queuedTx *QueuedTx, err error)

	// CompleteTransaction instructs backend to complete sending of a given transaction
	CompleteTransaction(id QueuedTxID, password string) (common.Hash, error)

	// CompleteTransactions instructs backend to complete sending of multiple transactions
	CompleteTransactions(ids []QueuedTxID, password string) map[QueuedTxID]RawCompleteTransactionResult

	// DiscardTransaction discards a given transaction from transaction queue
	DiscardTransaction(id QueuedTxID) error

	// DiscardTransactions discards given multiple transactions from transaction queue
	DiscardTransactions(ids []QueuedTxID) map[QueuedTxID]RawDiscardTransactionResult
}

// JailCell represents single jail cell, which is basically a JavaScript VM.
// It's designed to be a transparent wrapper around otto.VM's methods.
type JailCell interface {
	// Set a value inside VM.
	Set(string, interface{}) error
	// Get a value from VM.
	Get(string) (otto.Value, error)
	// Run an arbitrary JS code. Input maybe string or otto.Script.
	Run(interface{}) (otto.Value, error)
	// Call an arbitrary JS function by name and args.
	Call(item string, this interface{}, args ...interface{}) (otto.Value, error)
	// Stop stops background execution of cell.
	Stop()
}

// JailManager defines methods for managing jailed environments
type JailManager interface {
	// Parse creates a new jail cell context, with the given chatID as identifier.
	// New context executes provided JavaScript code, right after the initialization.
	Parse(chatID, js string) string

	// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
	Call(chatID, this, args string) string

	// NewCell initializes and returns a new jail cell.
	NewCell(chatID string) (JailCell, error)

	// Cell returns an existing instance of JailCell.
	Cell(chatID string) (JailCell, error)

	// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse()
	BaseJS(js string)

	// Stop stops all background activity of jail
	Stop()
}

// APIResponse generic response from API
type APIResponse struct {
	Error string `json:"error"`
}

// APIDetailedResponse represents a generic response
// with possible errors.
type APIDetailedResponse struct {
	Status      bool            `json:"status"`
	Message     string          `json:"message,omitempty"`
	FieldErrors []APIFieldError `json:"field_errors,omitempty"`
}

func (r APIDetailedResponse) Error() string {
	buf := bytes.NewBufferString("")

	for _, err := range r.FieldErrors {
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}

	return strings.TrimSpace(buf.String())
}

// APIFieldError represents a set of errors
// related to a parameter.
type APIFieldError struct {
	Parameter string     `json:"parameter,omitempty"`
	Errors    []APIError `json:"errors"`
}

func (e APIFieldError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	buf := bytes.NewBufferString(fmt.Sprintf("Parameter: %s\n", e.Parameter))

	for _, err := range e.Errors {
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}

	return strings.TrimSpace(buf.String())
}

// APIError represents a single error.
type APIError struct {
	Message string `json:"message"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("message=%s", e.Message)
}

// AccountInfo represents account's info
type AccountInfo struct {
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
	Error    string `json:"error"`
}

// StopRPCCallError defines a error type specific for killing a execution process.
type StopRPCCallError struct {
	Err error
}

// Error returns the internal error associated with the critical error.
func (c StopRPCCallError) Error() string {
	return c.Err.Error()
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
