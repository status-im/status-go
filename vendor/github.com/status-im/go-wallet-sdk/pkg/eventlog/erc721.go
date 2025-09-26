package eventlog

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
)

const (
	ERC721               ContractKey = "erc721"
	ERC721Approval       EventKey    = "erc721approval"
	ERC721ApprovalForAll EventKey    = "erc721approvalforall"
	ERC721Transfer       EventKey    = "erc721transfer"
)

var ERC721ApprovalID common.Hash = common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
var ERC721ApprovalForAllID common.Hash = common.HexToHash("0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31")
var ERC721TransferID common.Hash = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func init() {
	parsersMap[ERC721] = ParseLogERC721
}

func ParseLogERC721(log types.Log) *Event {
	abi := getABI(erc721.Erc721MetaData)
	event, _ := abi.EventByID(log.Topics[0])
	if event == nil {
		return nil
	}

	ret := &Event{
		ContractKey: ERC721,
		ContractABI: abi,
		ABIEvent:    event,
	}

	switch event.ID {
	case ERC721ApprovalID:
		unpacked := new(erc721.Erc721Approval)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC721Approval
	case ERC721ApprovalForAllID:
		unpacked := new(erc721.Erc721ApprovalForAll)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC721ApprovalForAll
	case ERC721TransferID:
		unpacked := new(erc721.Erc721Transfer)
		err := unpackLog(abi, unpacked, event.Name, log)
		if err != nil {
			return nil
		}
		unpacked.Raw = log
		ret.Unpacked = *unpacked
		ret.EventKey = ERC721Transfer
	default:
		return nil
	}

	return ret
}
