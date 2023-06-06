package activity

import (
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
)

const NoLimitTimestampForPeriod = 0

type Period struct {
	StartTimestamp int64 `json:"startTimestamp"`
	EndTimestamp   int64 `json:"endTimestamp"`
}

type Type int

const (
	SendAT Type = iota
	ReceiveAT
	BuyAT
	SwapAT
	BridgeAT
)

func allActivityTypesFilter() []Type {
	return []Type{}
}

type Status int

const (
	FailedAS Status = iota
	PendingAS
	CompleteAS
	FinalizedAS
)

func allActivityStatusesFilter() []Status {
	return []Status{}
}

type TokenType int

const (
	AssetTT TokenType = iota
	CollectiblesTT
)

type TokenCode string

// Tokens the following rules apply for its members:
// empty member: none is selected
// nil means all
// see allTokensFilter and noTokensFilter
type Tokens struct {
	Assets       []TokenCode          `json:"assets"`
	Collectibles []eth_common.Address `json:"collectibles"`
	EnabledTypes []TokenType          `json:"enabledTypes"`
}

func noAssetsFilter() Tokens {
	return Tokens{[]TokenCode{}, []eth_common.Address{}, []TokenType{CollectiblesTT}}
}

func allTokensFilter() Tokens {
	return Tokens{}
}

func allAddressesFilter() []eth_common.Address {
	return []eth_common.Address{}
}

func allNetworksFilter() []common.ChainID {
	return []common.ChainID{}
}

type Filter struct {
	Period                Period               `json:"period"`
	Types                 []Type               `json:"types"`
	Statuses              []Status             `json:"statuses"`
	Tokens                Tokens               `json:"tokens"`
	CounterpartyAddresses []eth_common.Address `json:"counterpartyAddresses"`
}
