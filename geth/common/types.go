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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

// errors
var (
	ErrDeprecatedMethod = errors.New("Method is depricated and will be removed in future release")
)

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
// This struct is based on go-ethereum's type in internal/ethapi/api.go, but we have freedom
// over the exact layout of this struct.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Input    hexutil.Bytes   `json:"input"`
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
