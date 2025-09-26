package transferdetector

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/pkg/pubsub"
)

func (c *Controller) sendEventTransferDetectionStarted(chainID uint64, accounts []common.Address, fromBlock uint64, toBlock uint64) {
	pubsub.Publish(c.publisher, EventTransferDetectionStarted{
		ChainID:   chainID,
		Accounts:  accounts,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
	})
}

func (c *Controller) sendEventTransferDetectionFinished(chainID uint64, accounts []common.Address, fromBlock uint64, toBlock uint64, events []eventlog.Event) {
	pubsub.Publish(c.publisher, EventTransferDetectionFinished{
		ChainID:   chainID,
		Accounts:  accounts,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Events:    events,
	})
}

func (c *Controller) sendEventTransferDetectionError(chainID uint64, accounts []common.Address, fromBlock uint64, toBlock uint64, err error) {
	pubsub.Publish(c.publisher, EventTransferDetectionError{
		ChainID:   chainID,
		Accounts:  accounts,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Error:     err,
	})
}
