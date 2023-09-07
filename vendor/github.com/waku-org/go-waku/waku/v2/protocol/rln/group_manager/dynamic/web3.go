package dynamic

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"go.uber.org/zap"
)

// the types of inputs to this handler matches the MemberRegistered event/proc defined in the MembershipContract interface
type RegistrationEventHandler = func(*DynamicGroupManager, []*contracts.RLNMemberRegistered) error

// HandleGroupUpdates mounts the supplied handler for the registration events emitting from the membership contract
// It connects to the eth client, subscribes to the `MemberRegistered` event emitted from the `MembershipContract`
// and collects all the events, for every received event, it calls the `handler`
func (gm *DynamicGroupManager) HandleGroupUpdates(ctx context.Context, handler RegistrationEventHandler) error {
	fromBlock := gm.web3Config.RLNContract.DeployedBlockNumber
	metadata, err := gm.GetMetadata()
	if err != nil {
		gm.log.Warn("could not load last processed block from metadata. Starting onchain sync from deployment block", zap.Error(err), zap.Uint64("deploymentBlock", gm.web3Config.RLNContract.DeployedBlockNumber))
	} else {
		if gm.web3Config.ChainID.Cmp(metadata.ChainID) != 0 {
			return errors.New("persisted data: chain id mismatch")
		}

		if !bytes.Equal(gm.web3Config.RegistryContract.Address.Bytes(), metadata.ContractAddress.Bytes()) {
			return errors.New("persisted data: contract address mismatch")
		}

		fromBlock = metadata.LastProcessedBlock
		gm.log.Info("resuming onchain sync", zap.Uint64("fromBlock", fromBlock))
	}

	gm.rootTracker.SetValidRootsPerBlock(metadata.ValidRootsPerBlock)

	err = gm.loadOldEvents(ctx, fromBlock, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	gm.wg.Add(1)
	go gm.watchNewEvents(ctx, handler, gm.log, errCh)
	return <-errCh
}

func (gm *DynamicGroupManager) loadOldEvents(ctx context.Context, fromBlock uint64, handler RegistrationEventHandler) error {
	events, err := gm.getEvents(ctx, fromBlock, nil)
	if err != nil {
		return err
	}
	return handler(gm, events)
}

func (gm *DynamicGroupManager) watchNewEvents(ctx context.Context, handler RegistrationEventHandler, log *zap.Logger, errCh chan<- error) {
	defer gm.wg.Done()

	// Watch for new events
	firstErr := true
	headerCh := make(chan *types.Header)
	subs := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		s, err := gm.web3Config.ETHClient.SubscribeNewHead(ctx, headerCh)
		if err != nil {
			if err == rpc.ErrNotificationsUnsupported {
				err = errors.New("notifications not supported. The node must support websockets")
			}
			if firstErr {
				errCh <- err
			}
			gm.log.Error("subscribing to rln events", zap.Error(err))
		}
		firstErr = false
		close(errCh)
		return s, err
	})

	defer subs.Unsubscribe()
	defer close(headerCh)

	for {
		select {
		case h := <-headerCh:
			blk := h.Number.Uint64()
			events, err := gm.getEvents(ctx, blk, &blk)
			if err != nil {
				gm.log.Error("obtaining rln events", zap.Error(err))
			}

			err = handler(gm, events)
			if err != nil {
				gm.log.Error("processing rln log", zap.Error(err))
			}
		case <-ctx.Done():
			return
		case err := <-subs.Err():
			if err != nil {
				gm.log.Error("watching new events", zap.Error(err))
			}
			return
		}
	}
}

const maxBatchSize = uint64(5000)
const additiveFactorMultiplier = 0.10
const multiplicativeDecreaseDivisor = 2

func tooMuchDataRequestedError(err error) bool {
	// this error is only infura specific (other providers might have different error messages)
	return err.Error() == "query returned more than 10000 results"
}

func (gm *DynamicGroupManager) getEvents(ctx context.Context, from uint64, to *uint64) ([]*contracts.RLNMemberRegistered, error) {
	var results []*contracts.RLNMemberRegistered

	// Adapted from prysm logic for fetching historical logs

	toBlock := to
	if to == nil {
		block, err := gm.web3Config.ETHClient.BlockByNumber(ctx, nil)
		if err != nil {
			return nil, err
		}

		blockNumber := block.Number().Uint64()
		toBlock = &blockNumber
	}

	if from == *toBlock { // Only loading a single block
		return gm.fetchEvents(ctx, from, toBlock)
	}

	// Fetching blocks in batches
	batchSize := maxBatchSize
	additiveFactor := uint64(float64(batchSize) * additiveFactorMultiplier)

	currentBlockNum := from
	for currentBlockNum < *toBlock {
		start := currentBlockNum
		end := currentBlockNum + batchSize
		if end > *toBlock {
			end = *toBlock
		}

		gm.log.Info("loading events...", zap.Uint64("fromBlock", start), zap.Uint64("toBlock", end))

		evts, err := gm.fetchEvents(ctx, start, &end)
		if err != nil {
			if tooMuchDataRequestedError(err) {
				if batchSize == 0 {
					return nil, errors.New("batch size is zero")
				}

				// multiplicative decrease
				batchSize = batchSize / multiplicativeDecreaseDivisor

				gm.log.Warn("too many logs requested!, retrying with a smaller chunk size", zap.Uint64("batchSize", batchSize))

				continue
			}
			return nil, err
		}

		results = append(results, evts...)

		currentBlockNum = end

		if batchSize < maxBatchSize {
			// update the batchSize with additive increase
			batchSize = batchSize + additiveFactor
			if batchSize > maxBatchSize {
				batchSize = maxBatchSize
			}
		}
	}

	return results, nil
}

func (gm *DynamicGroupManager) fetchEvents(ctx context.Context, from uint64, to *uint64) ([]*contracts.RLNMemberRegistered, error) {
	logIterator, err := gm.web3Config.RLNContract.FilterMemberRegistered(&bind.FilterOpts{Start: from, End: to, Context: ctx})
	if err != nil {
		return nil, err
	}

	var results []*contracts.RLNMemberRegistered

	for {
		if !logIterator.Next() {
			break
		}

		if logIterator.Error() != nil {
			return nil, logIterator.Error()
		}

		results = append(results, logIterator.Event)
	}

	return results, nil
}
