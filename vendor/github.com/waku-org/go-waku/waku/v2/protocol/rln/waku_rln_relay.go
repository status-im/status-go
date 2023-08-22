package rln

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
	proto "google.golang.org/protobuf/proto"
)

type GroupManager interface {
	Start(ctx context.Context, rln *rln.RLN, rootTracker *group_manager.MerkleRootTracker) error
	IdentityCredentials() (rln.IdentityCredential, error)
	MembershipIndex() (rln.MembershipIndex, error)
	Stop() error
}

type WakuRLNRelay struct {
	relay      *relay.WakuRelay
	timesource timesource.Timesource

	groupManager GroupManager
	rootTracker  *group_manager.MerkleRootTracker

	// pubsubTopic is the topic for which rln relay is mounted
	pubsubTopic  string
	contentTopic string
	spamHandler  SpamHandler

	RLN *rln.RLN

	// the log of nullifiers and Shamir shares of the past messages grouped per epoch
	nullifierLogLock sync.RWMutex
	nullifierLog     map[rln.Nullifier][]rln.ProofMetadata

	log *zap.Logger
}

const rlnDefaultTreePath = "./rln_tree.db"

func New(
	relay *relay.WakuRelay,
	groupManager GroupManager,
	treePath string,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler,
	timesource timesource.Timesource,
	log *zap.Logger) (*WakuRLNRelay, error) {

	if treePath == "" {
		treePath = rlnDefaultTreePath
	}

	rlnInstance, err := rln.NewWithConfig(rln.DefaultTreeDepth, &rln.TreeConfig{
		CacheCapacity: 15000,
		Mode:          rln.HighThroughput,
		Compression:   false,
		FlushInterval: 500,
		Path:          treePath,
	})
	if err != nil {
		return nil, err
	}

	rootTracker, err := group_manager.NewMerkleRootTracker(acceptableRootWindowSize, rlnInstance)
	if err != nil {
		return nil, err
	}

	// create the WakuRLNRelay
	rlnPeer := &WakuRLNRelay{
		RLN:          rlnInstance,
		groupManager: groupManager,
		rootTracker:  rootTracker,
		pubsubTopic:  pubsubTopic,
		contentTopic: contentTopic,
		relay:        relay,
		spamHandler:  spamHandler,
		log:          log,
		timesource:   timesource,
		nullifierLog: make(map[rln.MerkleNode][]rln.ProofMetadata),
	}

	return rlnPeer, nil
}

func (rlnRelay *WakuRLNRelay) Start(ctx context.Context) error {
	err := rlnRelay.groupManager.Start(ctx, rlnRelay.RLN, rlnRelay.rootTracker)
	if err != nil {
		return err
	}

	// adds a topic validator for the supplied pubsub topic at the relay protocol
	// messages published on this pubsub topic will be relayed upon a successful validation, otherwise they will be dropped
	// the topic validator checks for the correct non-spamming proof of the message
	err = rlnRelay.addValidator(rlnRelay.relay, rlnRelay.pubsubTopic, rlnRelay.contentTopic, rlnRelay.spamHandler)
	if err != nil {
		return err
	}

	log.Info("rln relay topic validator mounted", zap.String("pubsubTopic", rlnRelay.pubsubTopic), zap.String("contentTopic", rlnRelay.contentTopic))

	return nil
}

// Stop will stop any operation or goroutine started while using WakuRLNRelay
func (rlnRelay *WakuRLNRelay) Stop() error {
	return rlnRelay.groupManager.Stop()
}

func (rlnRelay *WakuRLNRelay) HasDuplicate(proofMD rln.ProofMetadata) (bool, error) {
	// returns true if there is another message in the  `nullifierLog` of the `rlnPeer` with the same
	// epoch and nullifier as `msg`'s epoch and nullifier but different Shamir secret shares
	// otherwise, returns false

	rlnRelay.nullifierLogLock.RLock()
	proofs, ok := rlnRelay.nullifierLog[proofMD.ExternalNullifier]
	rlnRelay.nullifierLogLock.RUnlock()

	// check if the epoch exists
	if !ok {
		return false, nil
	}

	for _, p := range proofs {
		if p.Equals(proofMD) {
			// there is an identical record, ignore rhe mag
			return true, nil
		}
	}

	// check for a message with the same nullifier but different secret shares
	matched := false
	for _, it := range proofs {
		if bytes.Equal(it.Nullifier[:], proofMD.Nullifier[:]) && (!bytes.Equal(it.ShareX[:], proofMD.ShareX[:]) || !bytes.Equal(it.ShareY[:], proofMD.ShareY[:])) {
			matched = true
			break
		}
	}

	return matched, nil
}

func (rlnRelay *WakuRLNRelay) updateLog(proofMD rln.ProofMetadata) (bool, error) {
	rlnRelay.nullifierLogLock.Lock()
	defer rlnRelay.nullifierLogLock.Unlock()
	proofs, ok := rlnRelay.nullifierLog[proofMD.ExternalNullifier]

	// check if the epoch exists
	if !ok {
		rlnRelay.nullifierLog[proofMD.ExternalNullifier] = []rln.ProofMetadata{proofMD}
		return true, nil
	}

	// check if an identical record exists
	for _, p := range proofs {
		if p.Equals(proofMD) {
			// TODO: slashing logic
			return true, nil
		}
	}

	// add proofMD to the log
	proofs = append(proofs, proofMD)
	rlnRelay.nullifierLog[proofMD.ExternalNullifier] = proofs

	return true, nil
}

// ValidateMessage validates the supplied message based on the waku-rln-relay routing protocol i.e.,
// the message's epoch is within `maxEpochGap` of the current epoch
// the message's has valid rate limit proof
// the message's does not violate the rate limit
// if `optionalTime` is supplied, then the current epoch is calculated based on that, otherwise the current time will be used
func (rlnRelay *WakuRLNRelay) ValidateMessage(msg *pb.WakuMessage, optionalTime *time.Time) (messageValidationResult, error) {
	//
	if msg == nil {
		return validationError, errors.New("nil message")
	}

	//  checks if the `msg`'s epoch is far from the current epoch
	// it corresponds to the validation of rln external nullifier
	var epoch rln.Epoch
	if optionalTime != nil {
		epoch = rln.CalcEpoch(*optionalTime)
	} else {
		// get current rln epoch
		epoch = rln.CalcEpoch(rlnRelay.timesource.Now())
	}

	msgProof := toRateLimitProof(msg)
	if msgProof == nil {
		// message does not contain a proof
		rlnRelay.log.Debug("invalid message: message does not contain a proof")
		return invalidMessage, nil
	}

	proofMD, err := rlnRelay.RLN.ExtractMetadata(*msgProof)
	if err != nil {
		rlnRelay.log.Debug("could not extract metadata", zap.Error(err))
		return invalidMessage, nil
	}

	// calculate the gaps and validate the epoch
	gap := rln.Diff(epoch, msgProof.Epoch)
	if int64(math.Abs(float64(gap))) > maxEpochGap {
		// message's epoch is too old or too ahead
		// accept messages whose epoch is within +-MAX_EPOCH_GAP from the current epoch
		rlnRelay.log.Debug("invalid message: epoch gap exceeds a threshold", zap.Int64("gap", gap))
		return invalidMessage, nil
	}

	valid, err := rlnRelay.verifyProof(msg, msgProof)
	if err != nil {
		rlnRelay.log.Debug("could not verify proof", zap.Error(err))
		return invalidMessage, nil
	}

	if !valid {
		// invalid proof
		rlnRelay.log.Debug("Invalid proof")
		return invalidMessage, nil
	}

	// check if double messaging has happened
	hasDup, err := rlnRelay.HasDuplicate(proofMD)
	if err != nil {
		rlnRelay.log.Debug("validation error", zap.Error(err))
		return validationError, err
	}

	if hasDup {
		rlnRelay.log.Debug("spam received")
		return spamMessage, nil
	}

	// insert the message to the log
	// the result of `updateLog` is discarded because message insertion is guaranteed by the implementation i.e.,
	// it will never error out
	_, err = rlnRelay.updateLog(proofMD)
	if err != nil {
		return validationError, err
	}

	rlnRelay.log.Debug("message is valid")
	return validMessage, nil
}

func (rlnRelay *WakuRLNRelay) verifyProof(msg *pb.WakuMessage, proof *rln.RateLimitProof) (bool, error) {
	contentTopicBytes := []byte(msg.ContentTopic)
	input := append(msg.Payload, contentTopicBytes...)
	return rlnRelay.RLN.Verify(input, *proof, rlnRelay.rootTracker.Roots()...)
}

func (rlnRelay *WakuRLNRelay) AppendRLNProof(msg *pb.WakuMessage, senderEpochTime time.Time) error {
	// returns error if it could not create and append a `RateLimitProof` to the supplied `msg`
	// `senderEpochTime` indicates the number of seconds passed since Unix epoch. The fractional part holds sub-seconds.
	// The `epoch` field of `RateLimitProof` is derived from the provided `senderEpochTime` (using `calcEpoch()`)

	if msg == nil {
		return errors.New("nil message")
	}

	input := toRLNSignal(msg)

	proof, err := rlnRelay.generateProof(input, rln.CalcEpoch(senderEpochTime))
	if err != nil {
		return err
	}

	msg.RateLimitProof = proof

	return nil
}

// this function sets a validator for the waku messages published on the supplied pubsubTopic and contentTopic
// if contentTopic is empty, then validation takes place for All the messages published on the given pubsubTopic
// the message validation logic is according to https://rfc.vac.dev/spec/17/
func (rlnRelay *WakuRLNRelay) addValidator(
	relay *relay.WakuRelay,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler) error {
	validator := func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		rlnRelay.log.Debug("rln-relay topic validator called")

		wakuMessage := &pb.WakuMessage{}
		if err := proto.Unmarshal(message.Data, wakuMessage); err != nil {
			rlnRelay.log.Debug("could not unmarshal message")
			return true
		}

		// check the contentTopic
		if (wakuMessage.ContentTopic != "") && (contentTopic != "") && (wakuMessage.ContentTopic != contentTopic) {
			rlnRelay.log.Debug("content topic did not match", zap.String("contentTopic", contentTopic))
			return true
		}

		// validate the message
		validationRes, err := rlnRelay.ValidateMessage(wakuMessage, nil)
		if err != nil {
			rlnRelay.log.Debug("validating message", zap.Error(err))
			return false
		}

		switch validationRes {
		case validMessage:
			rlnRelay.log.Debug("message verified",
				zap.String("pubsubTopic", pubsubTopic),
				zap.String("id", hex.EncodeToString(wakuMessage.Hash(pubsubTopic))),
			)
			return true
		case invalidMessage:
			rlnRelay.log.Debug("message could not be verified",
				zap.String("pubsubTopic", pubsubTopic),
				zap.String("id", hex.EncodeToString(wakuMessage.Hash(pubsubTopic))),
			)
			return false
		case spamMessage:
			rlnRelay.log.Debug("spam message found",
				zap.String("pubsubTopic", pubsubTopic),
				zap.String("id", hex.EncodeToString(wakuMessage.Hash(pubsubTopic))),
			)

			if spamHandler != nil {
				if err := spamHandler(wakuMessage); err != nil {
					rlnRelay.log.Error("executing spam handler", zap.Error(err))
				}
			}

			return false
		default:
			rlnRelay.log.Debug("unhandled validation result", zap.Int("validationResult", int(validationRes)))
			return false
		}
	}

	// In case there's a topic validator registered
	_ = relay.PubSub().UnregisterTopicValidator(pubsubTopic)

	return relay.PubSub().RegisterTopicValidator(pubsubTopic, validator)
}

func (rlnRelay *WakuRLNRelay) generateProof(input []byte, epoch rln.Epoch) (*pb.RateLimitProof, error) {
	identityCredentials, err := rlnRelay.groupManager.IdentityCredentials()
	if err != nil {
		return nil, err
	}

	membershipIndex, err := rlnRelay.groupManager.MembershipIndex()
	if err != nil {
		return nil, err
	}

	proof, err := rlnRelay.RLN.GenerateProof(input, identityCredentials, membershipIndex, epoch)
	if err != nil {
		return nil, err
	}

	return &pb.RateLimitProof{
		Proof:         proof.Proof[:],
		MerkleRoot:    proof.MerkleRoot[:],
		Epoch:         proof.Epoch[:],
		ShareX:        proof.ShareX[:],
		ShareY:        proof.ShareY[:],
		Nullifier:     proof.Nullifier[:],
		RlnIdentifier: proof.RLNIdentifier[:],
	}, nil
}

func (rlnRelay *WakuRLNRelay) IdentityCredential() (rln.IdentityCredential, error) {
	return rlnRelay.groupManager.IdentityCredentials()
}

func (rlnRelay *WakuRLNRelay) MembershipIndex() (uint, error) {
	return rlnRelay.groupManager.MembershipIndex()
}
