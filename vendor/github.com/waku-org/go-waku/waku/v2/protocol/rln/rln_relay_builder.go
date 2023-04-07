package rln

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

func RlnRelayStatic(
	ctx context.Context,
	relay *relay.WakuRelay,
	group []r.IDCommitment,
	memKeyPair r.IdentityCredential,
	memIndex r.MembershipIndex,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler,
	timesource timesource.Timesource,
	log *zap.Logger,
) (*WakuRLNRelay, error) {
	log = log.Named("rln-static")

	log.Info("mounting rln-relay in off-chain/static mode")

	// check the peer's index and the inclusion of user's identity commitment in the group
	if memKeyPair.IDCommitment != group[int(memIndex)] {
		return nil, errors.New("peer's IDCommitment does not match commitment in group")
	}

	rlnInstance, err := r.NewRLN()
	if err != nil {
		return nil, err
	}

	// create the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		ctx:               ctx,
		membershipKeyPair: &memKeyPair,
		membershipIndex:   memIndex,
		RLN:               rlnInstance,
		pubsubTopic:       pubsubTopic,
		contentTopic:      contentTopic,
		log:               log,
		timesource:        timesource,
		nullifierLog:      make(map[r.Nullifier][]r.ProofMetadata),
	}

	root, err := rlnPeer.RLN.GetMerkleRoot()
	if err != nil {
		return nil, err
	}

	rlnPeer.validMerkleRoots = append(rlnPeer.validMerkleRoots, root)

	// add members to the Merkle tree
	for _, member := range group {
		if err := rlnPeer.insertMember(member); err != nil {
			return nil, err
		}
	}

	// adds a topic validator for the supplied pubsub topic at the relay protocol
	// messages published on this pubsub topic will be relayed upon a successful validation, otherwise they will be dropped
	// the topic validator checks for the correct non-spamming proof of the message
	err = rlnPeer.addValidator(relay, pubsubTopic, contentTopic, spamHandler)
	if err != nil {
		return nil, err
	}

	log.Info("rln relay topic validator mounted", zap.String("pubsubTopic", pubsubTopic), zap.String("contentTopic", contentTopic))

	return rlnPeer, nil
}

func RlnRelayDynamic(
	ctx context.Context,
	relay *relay.WakuRelay,
	ethClientAddr string,
	ethAccountPrivateKey *ecdsa.PrivateKey,
	memContractAddr common.Address,
	memKeyPair *r.IdentityCredential,
	memIndex r.MembershipIndex,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler,
	registrationHandler RegistrationHandler,
	timesource timesource.Timesource,
	log *zap.Logger,
) (*WakuRLNRelay, error) {
	log = log.Named("rln-dynamic")

	log.Info("mounting rln-relay in onchain/dynamic mode")

	rlnInstance, err := r.NewRLN()
	if err != nil {
		return nil, err
	}

	// create the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		ctx:                       ctx,
		membershipIndex:           memIndex,
		membershipContractAddress: memContractAddr,
		ethClientAddress:          ethClientAddr,
		ethAccountPrivateKey:      ethAccountPrivateKey,
		RLN:                       rlnInstance,
		pubsubTopic:               pubsubTopic,
		contentTopic:              contentTopic,
		log:                       log,
		timesource:                timesource,
		nullifierLog:              make(map[r.Nullifier][]r.ProofMetadata),
		registrationHandler:       registrationHandler,
		lastIndexLoaded:           -1,
	}

	root, err := rlnPeer.RLN.GetMerkleRoot()
	if err != nil {
		return nil, err
	}

	rlnPeer.validMerkleRoots = append(rlnPeer.validMerkleRoots, root)

	// prepare rln membership key pair
	if memKeyPair == nil && ethAccountPrivateKey != nil {
		log.Debug("no rln-relay key is provided, generating one")
		memKeyPair, err = rlnInstance.MembershipKeyGen()
		if err != nil {
			return nil, err
		}

		rlnPeer.membershipKeyPair = memKeyPair

		// register the rln-relay peer to the membership contract
		membershipIndex, err := rlnPeer.Register(ctx)
		if err != nil {
			return nil, err
		}

		rlnPeer.membershipIndex = *membershipIndex

		log.Info("registered peer into the membership contract")
	} else if memKeyPair != nil {
		rlnPeer.membershipKeyPair = memKeyPair
	}

	handler := func(pubkey r.IDCommitment, index r.MembershipIndex) error {
		return rlnPeer.insertMember(pubkey)
	}

	errChan := make(chan error)
	go rlnPeer.HandleGroupUpdates(handler, errChan)
	err = <-errChan
	if err != nil {
		return nil, err
	}

	// adds a topic validator for the supplied pubsub topic at the relay protocol
	// messages published on this pubsub topic will be relayed upon a successful validation, otherwise they will be dropped
	// the topic validator checks for the correct non-spamming proof of the message
	err = rlnPeer.addValidator(relay, pubsubTopic, contentTopic, spamHandler)
	if err != nil {
		return nil, err
	}

	log.Info("rln relay topic validator mounted", zap.String("pubsubTopic", pubsubTopic), zap.String("contentTopic", contentTopic))

	return rlnPeer, nil

}
