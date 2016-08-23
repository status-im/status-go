package main

import (
	"github.com/ethereum/go-ethereum/les"
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
	Hash string         `json:"hash"`
	Args les.SendTxArgs `json:"args"`
}

type CompleteTransactionResult struct {
	Hash  string `json:"hash"`
	Error string `json:"error"`
}

type GethEvent struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}
