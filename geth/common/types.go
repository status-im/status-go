package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

// errors
var (
	ErrDeprecatedMethod  = errors.New("Method is depricated and will be removed in future release")
	ErrInvalidSendTxArgs = errors.New("Transaction arguments are invalid (are both 'input' and 'data' fields used?)")
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
	// We keep both "input" and "data" for backward compatibility.
	// "input" is a preferred field.
	// see `vendor/github.com/ethereum/go-ethereum/internal/ethapi/api.go:1107`
	Input hexutil.Bytes `json:"input"`
	Data  hexutil.Bytes `json:"data"`
}

// Valid checks whether this structure is filled in correctly.
func (args SendTxArgs) Valid() bool {
	// if at least one of the fields is empty, it is a valid struct
	if isNilOrEmpty(args.Input) || isNilOrEmpty(args.Data) {
		return true
	}

	// we only allow both fields to present if they have the same data
	return bytes.Equal(args.Input, args.Data)
}

// GetInput returns either Input or Data field's value dependent on what is filled.
func (args SendTxArgs) GetInput() hexutil.Bytes {
	if !isNilOrEmpty(args.Input) {
		return args.Input
	}

	return args.Data
}

func isNilOrEmpty(bytes hexutil.Bytes) bool {
	return bytes == nil || len(bytes) == 0
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
