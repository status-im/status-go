package thirdparty

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
)

//go:generate mockgen -package=mock_thirdparty -source=activity_types.go -destination=mock/activity_types.go

type ActivityProvider interface {
	ChainProvider
}

type Order string

const (
	OldToNew Order = "oldToNew"
	NewToOld Order = "newToOld"
)

type Direction string

const (
	Both     Direction = "both"
	Incoming Direction = "incoming"
	Outgoing Direction = "outgoing"
)

func (d Direction) IncludesIncoming() bool {
	return d == Incoming || d == Both
}

func (d Direction) IncludesOutgoing() bool {
	return d == Outgoing || d == Both
}

type ActivityFetchParameters struct {
	FromBlock *rpc.BlockNumber `json:"fromBlock"`
	ToBlock   *rpc.BlockNumber `json:"toBlock"`
	Address   common.Address   `json:"address"`
	Order     Order            `json:"order" validate:"required,oneof=oldToNew newToOld"`
	Direction Direction        `json:"direction" validate:"required,oneof=both incoming outgoing"`
}

type ActivityEntry struct {
	Timestamp       int64           `json:"timestamp"`
	ActivityType    ac.Type         `json:"activityType"`
	AmountOut       *hexutil.Big    `json:"amountOut,omitempty"`
	AmountIn        *hexutil.Big    `json:"amountIn,omitempty"`
	TokenOut        *ac.Token       `json:"tokenOut,omitempty"`
	TokenIn         *ac.Token       `json:"tokenIn,omitempty"`
	Sender          common.Address  `json:"sender"`
	Recipient       *common.Address `json:"recipient,omitempty"`
	ChainIDOut      *uint64         `json:"chainIdOut,omitempty"`
	ChainIDIn       *uint64         `json:"chainIdIn,omitempty"`
	ContractAddress *common.Address `json:"contractAddress,omitempty"`
	TxHash          common.Hash     `json:"txHash"`
	BlockNumber     *hexutil.Big    `json:"blockNumber"`
}

type ActivityEntryContainer ItemsContainer[ActivityEntry]

type ActivityFetcher interface {
	ActivityProvider
	FetchActivity(ctx context.Context, chainID uint64, parameters ActivityFetchParameters, cursor string, limit int) (ActivityEntryContainer, error)
}
