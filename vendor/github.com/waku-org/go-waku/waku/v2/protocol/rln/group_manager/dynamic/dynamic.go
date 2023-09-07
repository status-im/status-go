package dynamic

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-zerokit-rln/rln"
	om "github.com/wk8/go-ordered-map"
	"go.uber.org/zap"
)

var RLNAppInfo = keystore.AppInfo{
	Application:   "waku-rln-relay",
	AppIdentifier: "01234567890abcdef",
	Version:       "0.2",
}

type DynamicGroupManager struct {
	rln     *rln.RLN
	log     *zap.Logger
	metrics Metrics

	cancel context.CancelFunc
	wg     sync.WaitGroup

	identityCredential *rln.IdentityCredential
	membershipIndex    rln.MembershipIndex

	web3Config         *web3.Config
	lastBlockProcessed uint64

	eventHandler RegistrationEventHandler

	appKeystore      *keystore.AppKeystore
	keystorePassword string

	rootTracker *group_manager.MerkleRootTracker
}

func handler(gm *DynamicGroupManager, events []*contracts.RLNMemberRegistered) error {
	toRemoveTable := om.New()
	toInsertTable := om.New()

	lastBlockProcessed := gm.lastBlockProcessed
	for _, event := range events {
		if event.Raw.Removed {
			var indexes []uint
			i_idx, ok := toRemoveTable.Get(event.Raw.BlockNumber)
			if ok {
				indexes = i_idx.([]uint)
			}
			indexes = append(indexes, uint(event.Index.Uint64()))
			toRemoveTable.Set(event.Raw.BlockNumber, indexes)
		} else {
			var eventsPerBlock []*contracts.RLNMemberRegistered
			i_evt, ok := toInsertTable.Get(event.Raw.BlockNumber)
			if ok {
				eventsPerBlock = i_evt.([]*contracts.RLNMemberRegistered)
			}
			eventsPerBlock = append(eventsPerBlock, event)
			toInsertTable.Set(event.Raw.BlockNumber, eventsPerBlock)

			if event.Raw.BlockNumber > lastBlockProcessed {
				lastBlockProcessed = event.Raw.BlockNumber
			}
		}
	}

	err := gm.RemoveMembers(toRemoveTable)
	if err != nil {
		return err
	}

	err = gm.InsertMembers(toInsertTable)
	if err != nil {
		return err
	}

	gm.metrics.RecordRegisteredMembership(toInsertTable.Len() - toRemoveTable.Len())

	gm.lastBlockProcessed = lastBlockProcessed
	err = gm.SetMetadata(RLNMetadata{
		LastProcessedBlock: gm.lastBlockProcessed,
		ChainID:            gm.web3Config.ChainID,
		ContractAddress:    gm.web3Config.RegistryContract.Address,
		ValidRootsPerBlock: gm.rootTracker.ValidRootsPerBlock(),
	})
	if err != nil {
		// this is not a fatal error, hence we don't raise an exception
		gm.log.Warn("failed to persist rln metadata", zap.Error(err))
	} else {
		gm.log.Debug("rln metadata persisted", zap.Uint64("lastProcessedBlock", gm.lastBlockProcessed), zap.Uint64("chainID", gm.web3Config.ChainID.Uint64()), logging.HexBytes("contractAddress", gm.web3Config.RegistryContract.Address.Bytes()))
	}

	return nil
}

type RegistrationHandler = func(tx *types.Transaction)

func NewDynamicGroupManager(
	ethClientAddr string,
	memContractAddr common.Address,
	membershipIndex uint,
	appKeystore *keystore.AppKeystore,
	keystorePassword string,
	reg prometheus.Registerer,
	log *zap.Logger,
) (*DynamicGroupManager, error) {
	log = log.Named("rln-dynamic")

	return &DynamicGroupManager{
		membershipIndex:  membershipIndex,
		web3Config:       web3.NewConfig(ethClientAddr, memContractAddr),
		eventHandler:     handler,
		appKeystore:      appKeystore,
		keystorePassword: keystorePassword,
		log:              log,
		metrics:          newMetrics(reg),
	}, nil
}

func (gm *DynamicGroupManager) getMembershipFee(ctx context.Context) (*big.Int, error) {
	return gm.web3Config.RLNContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: ctx})
}

func (gm *DynamicGroupManager) memberExists(ctx context.Context, idCommitment rln.IDCommitment) (bool, error) {
	return gm.web3Config.RLNContract.MemberExists(&bind.CallOpts{Context: ctx}, rln.Bytes32ToBigInt(idCommitment))
}

func (gm *DynamicGroupManager) Start(ctx context.Context, rlnInstance *rln.RLN, rootTracker *group_manager.MerkleRootTracker) error {
	if gm.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	gm.cancel = cancel

	gm.log.Info("mounting rln-relay in on-chain/dynamic mode")

	err := gm.web3Config.Build(ctx)
	if err != nil {
		return err
	}

	gm.rln = rlnInstance
	gm.rootTracker = rootTracker

	// check if the contract exists by calling a static function
	_, err = gm.getMembershipFee(ctx)
	if err != nil {
		return err
	}

	err = gm.loadCredential(ctx)
	if err != nil {
		return err
	}

	if err = gm.HandleGroupUpdates(ctx, gm.eventHandler); err != nil {
		return err
	}

	return nil
}

func (gm *DynamicGroupManager) loadCredential(ctx context.Context) error {
	start := time.Now()

	credentials, err := gm.appKeystore.GetMembershipCredentials(
		gm.keystorePassword,
		gm.membershipIndex,
		keystore.NewMembershipContractInfo(gm.web3Config.ChainID, gm.web3Config.RegistryContract.Address))
	if err != nil {
		return err
	}
	gm.metrics.RecordMembershipCredentialsImportDuration(time.Since(start))

	if credentials == nil {
		return errors.New("no credentials available")
	}

	exists, err := gm.memberExists(ctx, credentials.IdentityCredential.IDCommitment)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("the provided commitment does not have a membership")
	}

	gm.identityCredential = credentials.IdentityCredential

	return nil
}

func (gm *DynamicGroupManager) InsertMembers(toInsert *om.OrderedMap) error {
	for pair := toInsert.Oldest(); pair != nil; pair = pair.Next() {
		events := pair.Value.([]*contracts.RLNMemberRegistered) // TODO: should these be sortered by index? we assume all members arrive in order
		var idCommitments []rln.IDCommitment
		var oldestIndexInBlock *big.Int
		for _, evt := range events {
			if oldestIndexInBlock == nil {
				oldestIndexInBlock = evt.Index
			}
			idCommitments = append(idCommitments, rln.BigIntToBytes32(evt.IdCommitment))
		}

		if len(idCommitments) == 0 {
			continue
		}

		// TODO: should we track indexes to identify missing?
		startIndex := rln.MembershipIndex(uint(oldestIndexInBlock.Int64()))
		start := time.Now()
		err := gm.rln.InsertMembers(startIndex, idCommitments)
		if err != nil {
			gm.log.Error("inserting members into merkletree", zap.Error(err))
			return err
		}
		gm.metrics.RecordMembershipInsertionDuration(time.Since(start))

		_, err = gm.rootTracker.UpdateLatestRoot(pair.Key.(uint64))
		if err != nil {
			return err
		}
	}
	return nil
}

func (gm *DynamicGroupManager) RemoveMembers(toRemove *om.OrderedMap) error {
	for pair := toRemove.Newest(); pair != nil; pair = pair.Prev() {
		memberIndexes := pair.Value.([]uint)
		err := gm.rln.DeleteMembers(memberIndexes)
		if err != nil {
			gm.log.Error("deleting members", zap.Error(err))
			return err
		}
		gm.rootTracker.Backfill(pair.Key.(uint64))
	}

	return nil
}

func (gm *DynamicGroupManager) IdentityCredentials() (rln.IdentityCredential, error) {
	if gm.identityCredential == nil {
		return rln.IdentityCredential{}, errors.New("identity credential has not been setup")
	}

	return *gm.identityCredential, nil
}

func (gm *DynamicGroupManager) MembershipIndex() rln.MembershipIndex {
	return gm.membershipIndex
}

// Stop stops all go-routines, eth client and closes the rln database
func (gm *DynamicGroupManager) Stop() error {
	if gm.cancel == nil {
		return nil
	}

	gm.cancel()

	err := gm.rln.Flush()
	if err != nil {
		return err
	}

	gm.web3Config.ETHClient.Close()

	gm.wg.Wait()

	return nil
}
