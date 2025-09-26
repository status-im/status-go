package eventlog

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
)

const (
	ERC1155               ContractKey = "erc1155"
	ERC1155TransferSingle EventKey    = "erc1155transfersingle"
	ERC1155TransferBatch  EventKey    = "erc1155transferbatch"
	ERC1155ApprovalForAll EventKey    = "erc1155approvalforall"
	ERC1155URI            EventKey    = "erc1155uri"
)

var ERC1155TransferSingleID common.Hash = common.HexToHash("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62")
var ERC1155TransferBatchID common.Hash = common.HexToHash("0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb")
var ERC1155ApprovalForAllID common.Hash = common.HexToHash("0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31")
var ERC1155URIID common.Hash = common.HexToHash("0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b")

func init() {
	parsersMap[ERC1155] = ParseLogERC1155
}

func ParseLogERC1155(log types.Log) *Event {
	abi := getABI(erc1155.Erc1155MetaData)
	event, _ := abi.EventByID(log.Topics[0])
	if event == nil {
		return nil
	}

	ret := &Event{
		ContractKey: ERC1155,
		ContractABI: abi,
		ABIEvent:    event,
	}

	switch event.ID {
	case ERC1155TransferSingleID:
		unpacked := new(erc1155.Erc1155TransferSingle)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC1155TransferSingle
	case ERC1155TransferBatchID:
		unpacked := new(erc1155.Erc1155TransferBatch)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC1155TransferBatch
	case ERC1155ApprovalForAllID:
		unpacked := new(erc1155.Erc1155ApprovalForAll)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC1155ApprovalForAll
	case ERC1155URIID:
		unpacked := new(erc1155.Erc1155URI)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC1155URI
	default:
		return nil
	}

	return ret
}
