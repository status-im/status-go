package eventlog

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	errNoEventSignature       = errors.New("no event signature")
	errEventSignatureMismatch = errors.New("event signature mismatch")
)

type logParser func(log types.Log) *Event

var parsersMap map[ContractKey]logParser = make(map[ContractKey]logParser)

func getABI(meta *bind.MetaData) *abi.ABI {
	abi, err := meta.GetAbi()
	if err != nil {
		panic(err)
	}
	return abi
}

func unpackLog(contractAbi *abi.ABI, out any, event string, log types.Log) error {
	// Anonymous events are not supported.
	if len(log.Topics) == 0 {
		return errNoEventSignature
	}
	if log.Topics[0] != contractAbi.Events[event].ID {
		return errEventSignatureMismatch
	}
	if len(log.Data) > 0 {
		if err := contractAbi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range contractAbi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}
