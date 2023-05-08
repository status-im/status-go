package dynamic

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	om "github.com/wk8/go-ordered-map"
	"go.uber.org/zap"
)

var RLNAppInfo = keystore.AppInfo{
	Application:   "go-waku-rln-relay",
	AppIdentifier: "01234567890abcdef",
	Version:       "0.1",
}

type DynamicGroupManager struct {
	rln *rln.RLN
	log *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup

	identityCredential *rln.IdentityCredential
	membershipIndex    *rln.MembershipIndex

	membershipContractAddress common.Address
	ethClientAddress          string
	ethClient                 *ethclient.Client

	// ethAccountPrivateKey is required for signing transactions
	// TODO may need to erase this ethAccountPrivateKey when is not used
	// TODO may need to make ethAccountPrivateKey mandatory
	ethAccountPrivateKey *ecdsa.PrivateKey

	eventHandler RegistrationEventHandler

	registrationHandler RegistrationHandler
	chainId             *big.Int
	rlnContract         *contracts.RLN
	membershipFee       *big.Int

	saveKeystore     bool
	keystorePath     string
	keystorePassword string

	rootTracker *group_manager.MerkleRootTracker
}

func handler(gm *DynamicGroupManager, events []*contracts.RLNMemberRegistered) error {
	toRemoveTable := om.New()
	toInsertTable := om.New()
	for _, event := range events {
		if event.Raw.Removed {
			var indexes []uint64
			i_idx, ok := toRemoveTable.Get(event.Raw.BlockNumber)
			if ok {
				indexes = i_idx.([]uint64)
			}
			indexes = append(indexes, event.Index.Uint64())
			toRemoveTable.Set(event.Raw.BlockNumber, indexes)
		} else {
			var eventsPerBlock []*contracts.RLNMemberRegistered
			i_evt, ok := toInsertTable.Get(event.Raw.BlockNumber)
			if ok {
				eventsPerBlock = i_evt.([]*contracts.RLNMemberRegistered)
			}
			eventsPerBlock = append(eventsPerBlock, event)
			toInsertTable.Set(event.Raw.BlockNumber, eventsPerBlock)
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

	return nil
}

type RegistrationHandler = func(tx *types.Transaction)

func NewDynamicGroupManager(
	ethClientAddr string,
	ethAccountPrivateKey *ecdsa.PrivateKey,
	memContractAddr common.Address,
	keystorePath string,
	keystorePassword string,
	saveKeystore bool,
	registrationHandler RegistrationHandler,
	log *zap.Logger,
) (*DynamicGroupManager, error) {
	log = log.Named("rln-dynamic")

	path := keystorePath
	if path == "" {
		log.Warn("keystore: no credentials path set, using default path", zap.String("path", keystore.RLN_CREDENTIALS_FILENAME))
		path = keystore.RLN_CREDENTIALS_FILENAME
	}

	password := keystorePassword
	if password == "" {
		log.Warn("keystore: no credentials password set, using default password", zap.String("password", keystore.RLN_CREDENTIALS_PASSWORD))
		password = keystore.RLN_CREDENTIALS_PASSWORD
	}

	return &DynamicGroupManager{
		membershipContractAddress: memContractAddr,
		ethClientAddress:          ethClientAddr,
		ethAccountPrivateKey:      ethAccountPrivateKey,
		registrationHandler:       registrationHandler,
		eventHandler:              handler,
		saveKeystore:              saveKeystore,
		keystorePath:              path,
		keystorePassword:          password,
		log:                       log,
	}, nil
}

func (gm *DynamicGroupManager) getMembershipFee(ctx context.Context) (*big.Int, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(gm.ethAccountPrivateKey, gm.chainId)
	if err != nil {
		return nil, err
	}
	auth.Context = ctx

	return gm.rlnContract.MEMBERSHIPDEPOSIT(&bind.CallOpts{Context: ctx})
}

func (gm *DynamicGroupManager) Start(ctx context.Context, rlnInstance *rln.RLN, rootTracker *group_manager.MerkleRootTracker) error {
	if gm.cancel != nil {
		return errors.New("already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	gm.cancel = cancel

	gm.log.Info("mounting rln-relay in on-chain/dynamic mode")

	backend, err := ethclient.Dial(gm.ethClientAddress)
	if err != nil {
		return err
	}
	gm.ethClient = backend

	gm.rln = rlnInstance
	gm.rootTracker = rootTracker

	gm.chainId, err = backend.ChainID(ctx)
	if err != nil {
		return err
	}

	gm.rlnContract, err = contracts.NewRLN(gm.membershipContractAddress, backend)
	if err != nil {
		return err
	}

	// check if the contract exists by calling a static function
	gm.membershipFee, err = gm.getMembershipFee(ctx)
	if err != nil {
		return err
	}

	if gm.identityCredential == nil && gm.keystorePassword != "" && gm.keystorePath != "" {
		credentials, err := keystore.GetMembershipCredentials(gm.log,
			gm.keystorePath,
			gm.keystorePassword,
			RLNAppInfo,
			nil,
			[]keystore.MembershipContract{{
				ChainId: gm.chainId.String(),
				Address: gm.membershipContractAddress.Hex(),
			}})
		if err != nil {
			return err
		}

		// TODO: accept an index from the config
		if len(credentials) != 0 {
			gm.identityCredential = &credentials[0].IdentityCredential
			gm.membershipIndex = &credentials[0].MembershipGroups[0].TreeIndex
		}
	}

	if gm.identityCredential == nil && gm.ethAccountPrivateKey == nil {
		return errors.New("either a credentials path or a private key must be specified")
	}

	// prepare rln membership key pair
	if gm.identityCredential == nil && gm.ethAccountPrivateKey != nil {
		gm.log.Info("no rln-relay key is provided, generating one")
		identityCredential, err := rlnInstance.MembershipKeyGen()
		if err != nil {
			return err
		}

		gm.identityCredential = identityCredential

		// register the rln-relay peer to the membership contract
		gm.membershipIndex, err = gm.Register(ctx)
		if err != nil {
			return err
		}

		err = gm.persistCredentials()
		if err != nil {
			return err
		}

		gm.log.Info("registered peer into the membership contract")
	}

	if gm.identityCredential == nil || gm.membershipIndex == nil {
		return errors.New("no credentials available")
	}

	if err = gm.HandleGroupUpdates(ctx, gm.eventHandler); err != nil {
		return err
	}

	return nil
}

func (gm *DynamicGroupManager) persistCredentials() error {
	if !gm.saveKeystore {
		return nil
	}

	if gm.identityCredential == nil || gm.membershipIndex == nil {
		return errors.New("no credentials to persist")
	}

	keystoreCred := keystore.MembershipCredentials{
		IdentityCredential: *gm.identityCredential,
		MembershipGroups: []keystore.MembershipGroup{{
			TreeIndex: *gm.membershipIndex,
			MembershipContract: keystore.MembershipContract{
				ChainId: gm.chainId.String(),
				Address: gm.membershipContractAddress.String(),
			},
		}},
	}

	err := keystore.AddMembershipCredentials(gm.keystorePath, []keystore.MembershipCredentials{keystoreCred}, gm.keystorePassword, RLNAppInfo, keystore.DefaultSeparator)
	if err != nil {
		return fmt.Errorf("failed to persist credentials: %w", err)
	}

	return nil
}

func (gm *DynamicGroupManager) InsertMembers(toInsert *om.OrderedMap) error {
	for pair := toInsert.Oldest(); pair != nil; pair = pair.Next() {
		events := pair.Value.([]*contracts.RLNMemberRegistered) // TODO: should these be sortered by index? we assume all members arrive in order
		for _, evt := range events {
			pubkey := rln.Bytes32(evt.Pubkey.Bytes())
			// TODO: should we track indexes to identify missing?
			err := gm.rln.InsertMember(pubkey)
			if err != nil {
				gm.log.Error("inserting member into merkletree", zap.Error(err))
				return err
			}
		}

		_, err := gm.rootTracker.UpdateLatestRoot(pair.Key.(uint64))
		if err != nil {
			return err
		}
	}
	return nil
}

func (gm *DynamicGroupManager) RemoveMembers(toRemove *om.OrderedMap) error {
	for pair := toRemove.Newest(); pair != nil; pair = pair.Prev() {
		memberIndexes := pair.Value.([]uint64)
		for _, index := range memberIndexes {
			err := gm.rln.DeleteMember(uint(index))
			if err != nil {
				gm.log.Error("deleting member", zap.Error(err))
				return err
			}
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

func (gm *DynamicGroupManager) SetCredentials(identityCredential *rln.IdentityCredential, index *rln.MembershipIndex) {
	gm.identityCredential = identityCredential
	gm.membershipIndex = index
}

func (gm *DynamicGroupManager) MembershipIndex() (rln.MembershipIndex, error) {
	if gm.membershipIndex == nil {
		return 0, errors.New("membership index has not been setup")
	}

	return *gm.membershipIndex, nil
}

func (gm *DynamicGroupManager) Stop() {
	if gm.cancel == nil {
		return
	}

	gm.cancel()
	gm.wg.Wait()
}
