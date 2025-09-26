package eventlog

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContractKey string
type EventKey string

type Event struct {
	ContractKey ContractKey
	ContractABI *abi.ABI
	EventKey    EventKey
	ABIEvent    *abi.Event
	Unpacked    any
}

func ParseLog(log types.Log) []Event {
	ret := make([]Event, 0)

	for _, parser := range parsersMap {
		event := parser(log)
		if event == nil {
			continue
		}
		ret = append(ret, *event)

	}
	return ret
}
