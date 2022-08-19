package rln

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	proto "github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	r "github.com/status-im/go-rln/rln"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

// the maximum clock difference between peers in seconds
const MAX_CLOCK_GAP_SECONDS = 20

// maximum allowed gap between the epochs of messages' RateLimitProofs
const MAX_EPOCH_GAP = int64(MAX_CLOCK_GAP_SECONDS / r.EPOCH_UNIT_SECONDS)

type WakuRLNRelay struct {
	ctx context.Context

	membershipKeyPair r.MembershipKeyPair

	// membershipIndex denotes the index of a leaf in the Merkle tree
	// that contains the pk of the current peer
	// this index is used to retrieve the peer's authentication path
	membershipIndex           r.MembershipIndex
	membershipContractAddress common.Address
	ethClientAddress          string
	// ethAccountPrivateKey is required for signing transactions
	// TODO may need to erase this ethAccountPrivateKey when is not used
	// TODO may need to make ethAccountPrivateKey mandatory
	ethAccountPrivateKey *ecdsa.PrivateKey
	RLN                  *r.RLN
	// pubsubTopic is the topic for which rln relay is mounted
	pubsubTopic  string
	contentTopic string
	// the log of nullifiers and Shamir shares of the past messages grouped per epoch
	nullifierLog map[r.Epoch][]r.ProofMetadata

	log *zap.Logger
}

func StaticSetup(rlnRelayMemIndex r.MembershipIndex) ([]r.IDCommitment, r.MembershipKeyPair, r.MembershipIndex, error) {
	// static group
	groupKeys := r.STATIC_GROUP_KEYS
	groupSize := r.STATIC_GROUP_SIZE

	// validate the user-supplied membership index
	if rlnRelayMemIndex >= r.MembershipIndex(groupSize) {
		return nil, r.MembershipKeyPair{}, 0, errors.New("wrong membership index")
	}

	// prepare the outputs from the static group keys

	// create a sequence of MembershipKeyPairs from the group keys (group keys are in string format)
	groupKeyPairs, err := toMembershipKeyPairs(groupKeys)
	if err != nil {
		return nil, r.MembershipKeyPair{}, 0, errors.New("invalid data on group keypairs")
	}

	// extract id commitment keys
	var groupOpt []r.IDCommitment
	for _, c := range groupKeyPairs {
		groupOpt = append(groupOpt, c.IDCommitment)
	}

	//  user selected membership key pair
	memKeyPairOpt := groupKeyPairs[rlnRelayMemIndex]
	memIndexOpt := rlnRelayMemIndex

	return groupOpt, memKeyPairOpt, memIndexOpt, nil
}

func (rln *WakuRLNRelay) HasDuplicate(msg *pb.WakuMessage) (bool, error) {
	// returns true if there is another message in the  `nullifierLog` of the `rlnPeer` with the same
	// epoch and nullifier as `msg`'s epoch and nullifier but different Shamir secret shares
	// otherwise, returns false

	if msg == nil {
		return false, errors.New("nil message")
	}

	msgProof := ToRateLimitProof(msg)

	// extract the proof metadata of the supplied `msg`
	proofMD := r.ProofMetadata{
		Nullifier: msgProof.Nullifier,
		ShareX:    msgProof.ShareX,
		ShareY:    msgProof.ShareY,
	}

	proofs, ok := rln.nullifierLog[msgProof.Epoch]

	// check if the epoch exists
	if !ok {
		return false, nil
	}

	for _, p := range proofs {
		if p.Equals(proofMD) {
			// there is an identical record, ignore rhe mag
			return false, nil
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

func (rln *WakuRLNRelay) updateLog(msg *pb.WakuMessage) (bool, error) {
	// extracts  the `ProofMetadata` of the supplied messages `msg` and
	// saves it in the `nullifierLog` of the `rlnPeer`

	if msg == nil {
		return false, errors.New("nil message")
	}

	msgProof := ToRateLimitProof(msg)

	proofMD := r.ProofMetadata{
		Nullifier: msgProof.Nullifier,
		ShareX:    msgProof.ShareX,
		ShareY:    msgProof.ShareY,
	}

	proofs, ok := rln.nullifierLog[msgProof.Epoch]

	// check if the epoch exists
	if !ok {
		rln.nullifierLog[msgProof.Epoch] = []r.ProofMetadata{proofMD}
		return true, nil
	}

	// check if an identical record exists
	for _, p := range proofs {
		if p.Equals(proofMD) {
			return true, nil
		}
	}

	// add proofMD to the log
	proofs = append(proofs, proofMD)
	rln.nullifierLog[msgProof.Epoch] = proofs

	return true, nil
}

func (rln *WakuRLNRelay) ValidateMessage(msg *pb.WakuMessage, optionalTime *time.Time) (MessageValidationResult, error) {
	// validate the supplied `msg` based on the waku-rln-relay routing protocol i.e.,
	// the `msg`'s epoch is within MAX_EPOCH_GAP of the current epoch
	// the `msg` has valid rate limit proof
	// the `msg` does not violate the rate limit
	// `timeOption` indicates Unix epoch time (fractional part holds sub-seconds)
	// if `timeOption` is supplied, then the current epoch is calculated based on that

	if msg == nil {
		return MessageValidationResult_Unknown, errors.New("nil message")
	}

	//  checks if the `msg`'s epoch is far from the current epoch
	// it corresponds to the validation of rln external nullifier
	var epoch r.Epoch
	if optionalTime != nil {
		epoch = r.CalcEpoch(*optionalTime)
	} else {
		// get current rln epoch
		epoch = r.GetCurrentEpoch()
	}

	msgProof := ToRateLimitProof(msg)
	if msgProof == nil {
		// message does not contain a proof
		rln.log.Debug("invalid message: message does not contain a proof")
		return MessageValidationResult_Invalid, nil
	}

	// calculate the gaps and validate the epoch
	gap := r.Diff(epoch, msgProof.Epoch)
	if int64(math.Abs(float64(gap))) >= MAX_EPOCH_GAP {
		// message's epoch is too old or too ahead
		// accept messages whose epoch is within +-MAX_EPOCH_GAP from the current epoch
		rln.log.Debug("invalid message: epoch gap exceeds a threshold", zap.Int64("gap", gap))
		return MessageValidationResult_Invalid, nil
	}

	// verify the proof
	contentTopicBytes := []byte(msg.ContentTopic)
	input := append(msg.Payload, contentTopicBytes...)
	if !rln.RLN.Verify(input, *msgProof) {
		// invalid proof
		rln.log.Debug("invalid message: invalid proof")
		return MessageValidationResult_Invalid, nil
	}

	// check if double messaging has happened
	hasDup, err := rln.HasDuplicate(msg)
	if err != nil {
		rln.log.Debug("validation error", zap.Error(err))
		return MessageValidationResult_Unknown, err
	}

	if hasDup {
		rln.log.Debug("spam received")
		return MessageValidationResult_Spam, nil
	}

	// insert the message to the log
	// the result of `updateLog` is discarded because message insertion is guaranteed by the implementation i.e.,
	// it will never error out
	_, err = rln.updateLog(msg)
	if err != nil {
		return MessageValidationResult_Unknown, err
	}

	rln.log.Debug("message is valid")
	return MessageValidationResult_Valid, nil
}

func (rln *WakuRLNRelay) AppendRLNProof(msg *pb.WakuMessage, senderEpochTime time.Time) error {
	// returns error if it could not create and append a `RateLimitProof` to the supplied `msg`
	// `senderEpochTime` indicates the number of seconds passed since Unix epoch. The fractional part holds sub-seconds.
	// The `epoch` field of `RateLimitProof` is derived from the provided `senderEpochTime` (using `calcEpoch()`)

	if msg == nil {
		return errors.New("nil message")
	}

	input := toRLNSignal(msg)

	proof, err := rln.RLN.GenerateProof(input, rln.membershipKeyPair, rln.membershipIndex, r.CalcEpoch(senderEpochTime))
	if err != nil {
		return err
	}

	msg.RateLimitProof = &pb.RateLimitProof{
		Proof:      proof.Proof[:],
		MerkleRoot: proof.MerkleRoot[:],
		Epoch:      proof.Epoch[:],
		ShareX:     proof.ShareX[:],
		ShareY:     proof.ShareY[:],
		Nullifier:  proof.Nullifier[:],
	}

	return nil
}

func (r *WakuRLNRelay) MembershipKeyPair() r.MembershipKeyPair {
	return r.membershipKeyPair
}

func (r *WakuRLNRelay) MembershipIndex() r.MembershipIndex {
	return r.membershipIndex
}

type SpamHandler = func(message *pb.WakuMessage) error

// this function sets a validator for the waku messages published on the supplied pubsubTopic and contentTopic
// if contentTopic is empty, then validation takes place for All the messages published on the given pubsubTopic
// the message validation logic is according to https://rfc.vac.dev/spec/17/
func (r *WakuRLNRelay) addValidator(
	relay *relay.WakuRelay,
	pubsubTopic string,
	contentTopic string,
	spamHandler SpamHandler) error {
	validator := func(ctx context.Context, peerID peer.ID, message *pubsub.Message) bool {
		r.log.Debug("rln-relay topic validator called")

		wakuMessage := &pb.WakuMessage{}
		if err := proto.Unmarshal(message.Data, wakuMessage); err != nil {
			r.log.Debug("could not unmarshal message")
			return true
		}

		// check the contentTopic
		if (wakuMessage.ContentTopic != "") && (contentTopic != "") && (wakuMessage.ContentTopic != contentTopic) {
			r.log.Debug("content topic did not match", zap.String("contentTopic", contentTopic))
			return true
		}

		// validate the message
		validationRes, err := r.ValidateMessage(wakuMessage, nil)
		if err != nil {
			r.log.Debug("validating message", zap.Error(err))
			return false
		}

		switch validationRes {
		case MessageValidationResult_Valid:
			r.log.Debug("message verified",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)
			return true
		case MessageValidationResult_Invalid:
			r.log.Debug("message could not be verified",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)
			return true
		case MessageValidationResult_Spam:
			r.log.Debug("spam message found",
				zap.String("contentTopic", wakuMessage.ContentTopic),
				zap.Binary("epoch", wakuMessage.RateLimitProof.Epoch),
				zap.Int("timestamp", int(wakuMessage.Timestamp)),
				zap.Binary("payload", wakuMessage.Payload),
				zap.Any("proof", wakuMessage.RateLimitProof),
			)

			if spamHandler != nil {
				if err := spamHandler(wakuMessage); err != nil {
					r.log.Error("executing spam handler", zap.Error(err))
				}
			}

			return false
		default:
			r.log.Debug("unhandled validation result", zap.Int("validationResult", int(validationRes)))
			return false
		}
	}

	return relay.PubSub().RegisterTopicValidator(pubsubTopic, validator)
}

func toMembershipKeyPairs(groupKeys [][]string) ([]r.MembershipKeyPair, error) {
	// groupKeys is  sequence of membership key tuples in the form of (identity key, identity commitment) all in the hexadecimal format
	// the ToMembershipKeyPairs proc populates a sequence of MembershipKeyPairs using the supplied groupKeys

	groupKeyPairs := []r.MembershipKeyPair{}
	for _, pair := range groupKeys {
		idKey, err := hex.DecodeString(pair[0])
		if err != nil {
			return nil, err
		}
		idCommitment, err := hex.DecodeString(pair[1])
		if err != nil {
			return nil, err
		}

		groupKeyPairs = append(groupKeyPairs, r.MembershipKeyPair{IDKey: r.IDKey(r.Bytes32(idKey)), IDCommitment: r.IDCommitment(r.Bytes32(idCommitment))})
	}

	return groupKeyPairs, nil
}

func toRLNSignal(wakuMessage *pb.WakuMessage) []byte {
	if wakuMessage == nil {
		return []byte{}
	}

	contentTopicBytes := []byte(wakuMessage.ContentTopic)
	return append(wakuMessage.Payload, contentTopicBytes...)
}

func ToRateLimitProof(msg *pb.WakuMessage) *r.RateLimitProof {
	if msg == nil || msg.RateLimitProof == nil {
		return nil
	}

	result := &r.RateLimitProof{
		Proof:      r.ZKSNARK(Bytes256(msg.RateLimitProof.Proof)),
		MerkleRoot: r.MerkleNode(Bytes32(msg.RateLimitProof.MerkleRoot)),
		Epoch:      r.Epoch(Bytes32(msg.RateLimitProof.Epoch)),
		ShareX:     r.MerkleNode(Bytes32(msg.RateLimitProof.ShareX)),
		ShareY:     r.MerkleNode(Bytes32(msg.RateLimitProof.ShareY)),
		Nullifier:  r.Nullifier(Bytes32(msg.RateLimitProof.Nullifier)),
	}

	return result
}

func Bytes32(b []byte) [32]byte {
	var result [32]byte
	copy(result[:], b)
	return result
}

func Bytes256(b []byte) [256]byte {
	var result [256]byte
	copy(result[:], b)
	return result
}
