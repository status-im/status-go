package dynamic

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

func ToBigInt(i []byte) *big.Int {
	result := new(big.Int)
	result.SetBytes(i[:])
	return result
}

func register(ctx context.Context, backend *ethclient.Client, membershipFee *big.Int, idComm r.IDCommitment, ethAccountPrivateKey *ecdsa.PrivateKey, rlnContract *contracts.RLN, chainID *big.Int, registrationHandler RegistrationHandler, log *zap.Logger) (*r.MembershipIndex, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(ethAccountPrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	auth.Value = membershipFee
	auth.Context = ctx

	log.Debug("registering an id commitment", zap.Binary("idComm", idComm[:]))

	// registers the idComm  into the membership contract whose address is in rlnPeer.membershipContractAddress
	tx, err := rlnContract.Register(auth, ToBigInt(idComm[:]))
	if err != nil {
		return nil, err
	}

	log.Info("transaction broadcasted", zap.String("transactionHash", tx.Hash().Hex()))

	if registrationHandler != nil {
		registrationHandler(tx)
	}

	txReceipt, err := bind.WaitMined(ctx, backend, tx)
	if err != nil {
		return nil, err
	}

	if txReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("transaction reverted")
	}

	// the receipt topic holds the hash of signature of the raised events
	evt, err := rlnContract.ParseMemberRegistered(*txReceipt.Logs[0])
	if err != nil {
		return nil, err
	}

	var eventIdComm r.IDCommitment = r.Bytes32(evt.Pubkey.Bytes())

	log.Debug("the identity commitment key extracted from tx log", zap.Binary("eventIdComm", eventIdComm[:]))

	if eventIdComm != idComm {
		return nil, errors.New("invalid id commitment key")
	}

	result := new(r.MembershipIndex)
	*result = r.MembershipIndex(uint(evt.Index.Int64()))

	// debug "the index of registered identity commitment key", eventIndex=eventIndex

	log.Debug("the index of registered identity commitment key", zap.Uint("eventIndex", uint(*result)))

	return result, nil
}

// Register registers the public key of the rlnPeer which is rlnPeer.membershipKeyPair.publicKey
// into the membership contract whose address is in rlnPeer.membershipContractAddress
func (gm *DynamicGroupManager) Register(ctx context.Context) (*r.MembershipIndex, error) {
	return register(ctx,
		gm.ethClient,
		gm.membershipFee,
		gm.identityCredential.IDCommitment,
		gm.ethAccountPrivateKey,
		gm.rlnContract,
		gm.chainId,
		gm.registrationHandler,
		gm.log)
}

// the types of inputs to this handler matches the MemberRegistered event/proc defined in the MembershipContract interface
type RegistrationEventHandler = func(*DynamicGroupManager, []*contracts.RLNMemberRegistered) error

// HandleGroupUpdates mounts the supplied handler for the registration events emitting from the membership contract
// It connects to the eth client, subscribes to the `MemberRegistered` event emitted from the `MembershipContract`
// and collects all the events, for every received event, it calls the `handler`
func (gm *DynamicGroupManager) HandleGroupUpdates(ctx context.Context, handler RegistrationEventHandler) error {
	err := gm.loadOldEvents(ctx, gm.rlnContract, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	gm.wg.Add(1)
	go gm.watchNewEvents(ctx, gm.rlnContract, handler, gm.log, errCh)
	return <-errCh
}

func (gm *DynamicGroupManager) loadOldEvents(ctx context.Context, rlnContract *contracts.RLN, handler RegistrationEventHandler) error {
	events, err := gm.getEvents(ctx, 0, nil)
	if err != nil {
		return err
	}
	return handler(gm, events)
}

func (gm *DynamicGroupManager) watchNewEvents(ctx context.Context, rlnContract *contracts.RLN, handler RegistrationEventHandler, log *zap.Logger, errCh chan<- error) {
	defer gm.wg.Done()

	// Watch for new events
	firstErr := true
	headerCh := make(chan *types.Header)
	subs := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		s, err := gm.ethClient.SubscribeNewHead(ctx, headerCh)
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

const maxBatchSize = uint64(5000000) // TODO: tune this
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
		block, err := gm.ethClient.BlockByNumber(ctx, nil)
		if err != nil {
			return nil, err
		}

		blockNumber := block.Number().Uint64()
		toBlock = &blockNumber
	}

	batchSize := maxBatchSize
	additiveFactor := uint64(float64(batchSize) * additiveFactorMultiplier)

	currentBlockNum := from
	for currentBlockNum < *toBlock {
		start := currentBlockNum
		end := currentBlockNum + batchSize
		if end > *toBlock {
			end = *toBlock
		}

		evts, err := gm.fetchEvents(ctx, start, &end)
		if err != nil {
			if tooMuchDataRequestedError(err) {
				if batchSize == 0 {
					return nil, errors.New("batch size is zero")
				}

				// multiplicative decrease
				batchSize = batchSize / multiplicativeDecreaseDivisor
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
	logIterator, err := gm.rlnContract.FilterMemberRegistered(&bind.FilterOpts{Start: from, End: to, Context: ctx})
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
