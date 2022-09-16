package rln

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	r "github.com/status-im/go-rln/rln"
	"github.com/status-im/go-waku/waku/v2/protocol/rln/contracts"
	"go.uber.org/zap"
)

var MEMBERSHIP_FEE = big.NewInt(1000000000000000) // wei - 0.001 eth

func toBigInt(i []byte) *big.Int {
	result := new(big.Int)
	result.SetBytes(i[:])
	return result
}

func register(ctx context.Context, idComm r.IDCommitment, ethAccountPrivateKey *ecdsa.PrivateKey, ethClientAddress string, membershipContractAddress common.Address, registrationHandler RegistrationHandler, log *zap.Logger) (*r.MembershipIndex, error) {
	backend, err := ethclient.Dial(ethClientAddress)
	if err != nil {
		return nil, err
	}
	defer backend.Close()

	chainID, err := backend.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(ethAccountPrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	auth.Value = MEMBERSHIP_FEE
	auth.Context = ctx

	rlnContract, err := contracts.NewRLN(membershipContractAddress, backend)
	if err != nil {
		return nil, err
	}

	log.Debug("registering an id commitment", zap.Binary("idComm", idComm[:]))

	// registers the idComm  into the membership contract whose address is in rlnPeer.membershipContractAddress
	tx, err := rlnContract.Register(auth, toBigInt(idComm[:]))
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

	var eventIdComm r.IDCommitment
	copy(eventIdComm[:], evt.Pubkey.Bytes())

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
func (rln *WakuRLNRelay) Register(ctx context.Context) (*r.MembershipIndex, error) {
	pk := rln.membershipKeyPair.IDCommitment
	return register(ctx, pk, rln.ethAccountPrivateKey, rln.ethClientAddress, rln.membershipContractAddress, rln.registrationHandler, rln.log)
}

// the types of inputs to this handler matches the MemberRegistered event/proc defined in the MembershipContract interface
type RegistrationEventHandler = func(pubkey r.IDCommitment, index r.MembershipIndex) error

func processLogs(evt *contracts.RLNMemberRegistered, handler RegistrationEventHandler) error {
	if evt == nil {
		return nil
	}

	var pubkey r.IDCommitment
	copy(pubkey[:], evt.Pubkey.Bytes())

	index := r.MembershipIndex(uint(evt.Index.Int64()))

	return handler(pubkey, index)
}

// HandleGroupUpdates mounts the supplied handler for the registration events emitting from the membership contract
// It connects to the eth client, subscribes to the `MemberRegistered` event emitted from the `MembershipContract`
// and collects all the events, for every received event, it calls the `handler`
func (rln *WakuRLNRelay) HandleGroupUpdates(handler RegistrationEventHandler, errChan chan<- error) {
	defer close(errChan)

	backend, err := ethclient.Dial(rln.ethClientAddress)
	if err != nil {
		errChan <- err
		return
	}
	rln.ethClient = backend

	rlnContract, err := contracts.NewRLN(rln.membershipContractAddress, backend)
	if err != nil {
		errChan <- err
		return
	}

	err = rln.loadOldEvents(rlnContract, handler)
	if err != nil {
		errChan <- err
		return
	}

	doneCh := make(chan struct{})
	errCh := make(chan error)
	go rln.watchNewEvents(rlnContract, handler, rln.log, doneCh, errCh)

	select {
	case <-doneCh:
		return
	case err := <-errCh:
		errChan <- err
		return
	}
}

func (rln *WakuRLNRelay) loadOldEvents(rlnContract *contracts.RLN, handler RegistrationEventHandler) error {
	logIterator, err := rlnContract.FilterMemberRegistered(&bind.FilterOpts{Start: 0, End: nil, Context: rln.ctx})
	if err != nil {
		return err
	}
	for {
		if !logIterator.Next() {
			break
		}

		if logIterator.Error() != nil {
			return logIterator.Error()
		}

		err = processLogs(logIterator.Event, handler)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rln *WakuRLNRelay) watchNewEvents(rlnContract *contracts.RLN, handler RegistrationEventHandler, log *zap.Logger, doneCh chan struct{}, errCh chan error) {
	// Watch for new events
	logSink := make(chan *contracts.RLNMemberRegistered)
	subs, err := rlnContract.WatchMemberRegistered(&bind.WatchOpts{Context: rln.ctx, Start: nil}, logSink)
	if err != nil {
		if err == rpc.ErrNotificationsUnsupported {
			err = errors.New("notifications not supported. The node must support websockets")
		}
		errCh <- err
		return
	}
	defer subs.Unsubscribe()

	close(doneCh)

	for {
		select {
		case evt := <-logSink:
			err = processLogs(evt, handler)
			if err != nil {
				rln.log.Error("processing rln log", zap.Error(err))
			}
		case <-rln.ctx.Done():
			subs.Unsubscribe()
			close(logSink)
			return
		case err := <-subs.Err():
			close(logSink)
			if err != nil {
				rln.log.Error("watching new events", zap.Error(err))
				errCh <- err
			}
			return
		}
	}
}
