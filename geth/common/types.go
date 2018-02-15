package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	StartNode(config *params.NodeConfig) error

	// EnsureSync waits until blockchain is synchronized.
	EnsureSync(ctx context.Context) error

	// StopNode stop the running Status node.
	// Stopped node cannot be resumed, one starts a new node instead.
	StopNode() error

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

	// PeerCount returns number of connected peers
	PeerCount() int

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

// TransactionResult is a JSON returned from transaction complete function (used internally)
type TransactionResult struct {
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
	ID      QueuedTxID
	Context context.Context
	Args    SendTxArgs
	Result  chan TransactionResult
}

// SendTxArgs represents the arguments to submit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

// JailCell represents single jail cell, which is basically a JavaScript VM.
// It's designed to be a transparent wrapper around otto.VM's methods.
type JailCell interface {
	// Set a value inside VM.
	Set(string, interface{}) error
	// Get a value from VM.
	Get(string) (otto.Value, error)
	// GetObjectValue returns the given name's otto.Value from the given otto.Value v. Should only be needed in tests.
	GetObjectValue(otto.Value, string) (otto.Value, error)
	// Run an arbitrary JS code. Input maybe string or otto.Script.
	Run(interface{}) (otto.Value, error)
	// Call an arbitrary JS function by name and args.
	Call(item string, this interface{}, args ...interface{}) (otto.Value, error)
	// Stop stops background execution of cell.
	Stop() error
}

// JailManager defines methods for managing jailed environments
type JailManager interface {
	// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
	Call(chatID, this, args string) string

	// CreateCell creates a new jail cell.
	CreateCell(chatID string) (JailCell, error)

	// Parse creates a new jail cell context, with the given chatID as identifier.
	// New context executes provided JavaScript code, right after the initialization.
	// DEPRECATED in favour of CreateAndInitCell.
	Parse(chatID, js string) string

	// CreateAndInitCell creates a new jail cell and initialize it
	// with web3 and other handlers.
	CreateAndInitCell(chatID string, code ...string) string

	// Cell returns an existing instance of JailCell.
	Cell(chatID string) (JailCell, error)

	// Execute allows to run arbitrary JS code within a cell.
	Execute(chatID, code string) string

	// SetBaseJS allows to setup initial JavaScript to be loaded on each jail.CreateAndInitCell().
	SetBaseJS(js string)

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
		buf.WriteString(err.Error() + "\n") // nolint: gas
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
		buf.WriteString(err.Error() + "\n") // nolint: gas
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

type account struct {
	Address  string
	Password string
}

// TestConfig contains shared (among different test packages) parameters
type TestConfig struct {
	Node struct {
		SyncSeconds time.Duration
		HTTPPort    int
		WSPort      int
	}
	Account1 account
	Account2 account
	Account3 account
}

// NotifyResult is a JSON returned from notify message
type NotifyResult struct {
	Status bool   `json:"status"`
	Error  string `json:"error,omitempty"`
}

const passphraseEnvName = "ACCOUNT_PASSWORD"

// LoadTestConfig loads test configuration values from disk
func LoadTestConfig(networkID int) (*TestConfig, error) {
	var testConfig TestConfig

	configData := static.MustAsset("config/test-data.json")
	if err := json.Unmarshal(configData, &testConfig); err != nil {
		return nil, err
	}

	if networkID == params.StatusChainNetworkID {
		accountsData := static.MustAsset("config/status-chain-accounts.json")
		if err := json.Unmarshal(accountsData, &testConfig); err != nil {
			return nil, err
		}
	} else {
		accountsData := static.MustAsset("config/public-chain-accounts.json")
		if err := json.Unmarshal(accountsData, &testConfig); err != nil {
			return nil, err
		}

		pass := os.Getenv(passphraseEnvName)
		testConfig.Account1.Password = pass
		testConfig.Account2.Password = pass
	}

	return &testConfig, nil
}
