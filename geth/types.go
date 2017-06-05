package geth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
)

// SignalEnvelope is a general signal sent upward from node to RN app
type SignalEnvelope struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}

// AccountInfo represents account's info
type AccountInfo struct {
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
	Error    string `json:"error"`
}

// JSONError is wrapper around errors, that are sent upwards
type JSONError struct {
	Error string `json:"error"`
}

// NodeCrashEvent is special kind of error, used to report node crashes
type NodeCrashEvent struct {
	Error string `json:"error"`
}

// AddPeerResult is a JSON returned as a response to AddPeer() request
type AddPeerResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// WhisperMessageEvent is a signal sent on incoming Whisper message
type WhisperMessageEvent struct {
	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
	TTL     int64  `json:"ttl"`
	Hash    string `json:"hash"`
}

// SendTransactionEvent is a signal sent on a send transaction request
type SendTransactionEvent struct {
	ID        string            `json:"id"`
	Args      status.SendTxArgs `json:"args"`
	MessageID string            `json:"message_id"`
}

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string            `json:"id"`
	Args         status.SendTxArgs `json:"args"`
	MessageID    string            `json:"message_id"`
	ErrorMessage string            `json:"error_message"`
	ErrorCode    string            `json:"error_code"`
}

// CompleteTransactionResult is a JSON returned from transaction complete function (used in exposed method)
type CompleteTransactionResult struct {
	ID    string `json:"id"`
	Hash  string `json:"hash"`
	Error string `json:"error"`
}

// RawCompleteTransactionResult is a JSON returned from transaction complete function (used internally)
type RawCompleteTransactionResult struct {
	Hash  common.Hash
	Error error
}

// CompleteTransactionsResult is list of results from CompleteTransactions() (used in exposed method)
type CompleteTransactionsResult struct {
	Results map[string]CompleteTransactionResult `json:"results"`
}

// RawDiscardTransactionResult is list of results from CompleteTransactions() (used internally)
type RawDiscardTransactionResult struct {
	Error error
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

// LocalStorageSetEvent is a signal sent whenever local storage Set method is called
type LocalStorageSetEvent struct {
	ChatID string `json:"chat_id"`
	Data   string `json:"data"`
}

// RPCCall represents RPC call parameters
type RPCCall struct {
	ID     int64
	Method string
	Params []interface{}
}

// SendMessageEvent wraps Jail send signals
type SendMessageEvent struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

// ShowSuggestionsEvent wraps Jail show suggestion signals
type ShowSuggestionsEvent struct {
	ChatID string `json:"chat_id"`
	Markup string `json:"markup"`
}
