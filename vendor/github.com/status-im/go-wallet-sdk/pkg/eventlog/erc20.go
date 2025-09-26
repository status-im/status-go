package eventlog

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
)

const (
	ERC20         ContractKey = "erc20"
	ERC20Approval EventKey    = "erc20approval"
	ERC20Transfer EventKey    = "erc20transfer"
)

var ERC20ApprovalID common.Hash = common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
var ERC20TransferID common.Hash = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func init() {
	parsersMap[ERC20] = ParseLogERC20
}

func ParseLogERC20(log types.Log) *Event {
	abi := getABI(erc20.Erc20MetaData)
	event, _ := abi.EventByID(log.Topics[0])
	if event == nil {
		return nil
	}

	ret := &Event{
		ContractKey: ERC20,
		ContractABI: abi,
		ABIEvent:    event,
	}

	switch event.ID {
	case ERC20ApprovalID:
		unpacked := new(erc20.Erc20Approval)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC20Approval
	case ERC20TransferID:
		unpacked := new(erc20.Erc20Transfer)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC20Transfer
	default:
		return nil
	}

	return ret
}
