package geth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
)

type AccountInfo struct {
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
	Error    string `json:"error"`
}

type JSONError struct {
	Error string `json:"error"`
}

type AddPeerResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type AddWhisperFilterResult struct {
	Id    int    `json:"id"`
	Error string `json:"error"`
}

type WhisperMessageEvent struct {
	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
	TTL     int64  `json:"ttl"`
	Hash    string `json:"hash"`
}

type SendTransactionEvent struct {
	Id        string            `json:"id"`
	Args      status.SendTxArgs `json:"args"`
	MessageId string            `json:"message_id"`
}

type LocalStorageEvent struct {
	Data string            `json:"data"`
}

type ReturnSendTransactionEvent struct {
	Id           string            `json:"id"`
	Args         status.SendTxArgs `json:"args"`
	MessageId    string            `json:"message_id"`
	ErrorMessage string            `json:"error_message"`
	ErrorCode    string            `json:"error_code"`
}

type CompleteTransactionResult struct {
	Id    string `json:"id"`
	Hash  string `json:"hash"`
	Error string `json:"error"`
}

type RawCompleteTransactionResult struct {
	Hash  common.Hash
	Error error
}

type CompleteTransactionsResult struct {
	Results map[string]CompleteTransactionResult `json:"results"`
}

type RawDiscardTransactionResult struct {
	Error error
}

type DiscardTransactionResult struct {
	Id    string `json:"id"`
	Error string `json:"error"`
}

type DiscardTransactionsResult struct {
	Results map[string]DiscardTransactionResult `json:"results"`
}

type GethEvent struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}

type RPCCall struct {
	Id     int64
	Method string
	Params []interface{}
}
