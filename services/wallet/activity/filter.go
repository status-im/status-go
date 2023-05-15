package activity

import "github.com/ethereum/go-ethereum/common"

type Period struct {
	// 0 means no limit
	StartTimestamp int64 `json:"startTimestamp"`
	EndTimestamp   int64 `json:"endTimestamp"`
}

type Type int

const (
	AllAT Type = iota
	SendAT
	ReceiveAT
	BuyAT
	SwapAT
	BridgeAT
)

type Status int

const (
	AllAS Status = iota
	FailedAS
	PendingAS
	CompleteAS
	FinalizedAS
)

type TokenType int

const (
	AllTT TokenType = iota
	AssetTT
	CollectiblesTT
)

type Filter struct {
	Period                Period           `json:"period"`
	Types                 []Type           `json:"types"`
	Statuses              []Status         `json:"statuses"`
	TokenTypes            []TokenType      `json:"tokenTypes"`
	CounterpartyAddresses []common.Address `json:"counterpartyAddresses"`
}
