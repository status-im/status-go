//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"context"
	"encoding/hex"
	"errors"

	r "github.com/status-im/go-rln/rln"
	"github.com/status-im/go-waku/waku/v2/protocol/rln"
	"go.uber.org/zap"
)

// RLNRelay is used to access any operation related to Waku RLN protocol
func (w *WakuNode) RLNRelay() RLNRelay {
	return w.rlnRelay
}

func (w *WakuNode) mountRlnRelay() error {
	// check whether inputs are provided
	// relay protocol is the prerequisite of rln-relay
	if w.Relay() == nil {
		return errors.New("relay protocol is required")
	}

	// check whether the pubsub topic is supported at the relay level
	topicFound := false
	for _, t := range w.Relay().Topics() {
		if t == w.opts.rlnRelayPubsubTopic {
			topicFound = true
			break
		}
	}

	if !topicFound {
		return errors.New("relay protocol does not support the configured pubsub topic")
	}

	if !w.opts.rlnRelayDynamic {
		w.log.Info("setting up waku-rln-relay in off-chain mode")
		// set up rln relay inputs
		groupKeys, memKeyPair, memIndex, err := rln.StaticSetup(w.opts.rlnRelayMemIndex)
		if err != nil {
			return err
		}

		// mount rlnrelay in off-chain mode with a static group of users
		rlnRelay, err := rln.RlnRelayStatic(w.ctx, w.relay, groupKeys, memKeyPair, memIndex, w.opts.rlnRelayPubsubTopic, w.opts.rlnRelayContentTopic, w.opts.rlnSpamHandler, w.log)
		if err != nil {
			return err
		}

		w.rlnRelay = rlnRelay

		w.log.Info("membership id key", zap.String("IDKey", hex.EncodeToString(memKeyPair.IDKey[:])))
		w.log.Info("membership id commitment key", zap.String("IDCommitment", hex.EncodeToString(memKeyPair.IDCommitment[:])))

		// check the correct construction of the tree by comparing the calculated root against the expected root
		// no error should happen as it is already captured in the unit tests
		root, err := rlnRelay.RLN.GetMerkleRoot()
		if err != nil {
			return err
		}

		expectedRoot := r.STATIC_GROUP_MERKLE_ROOT
		if hex.EncodeToString(root[:]) != expectedRoot {
			return errors.New("root mismatch: something went wrong not in Merkle tree construction")
		}

		w.log.Info("the calculated root", zap.String("root", hex.EncodeToString(root[:])))
	} else {
		w.log.Info("setting up waku-rln-relay in on-chain mode")

		//  check if the peer has provided its rln credentials
		var memKeyPair *r.MembershipKeyPair
		if w.opts.rlnRelayIDCommitment != nil && w.opts.rlnRelayIDKey != nil {
			memKeyPair = &r.MembershipKeyPair{
				IDCommitment: *w.opts.rlnRelayIDCommitment,
				IDKey:        *w.opts.rlnRelayIDKey,
			}
		}

		// mount the rln relay protocol in the on-chain/dynamic mode
		var err error
		w.rlnRelay, err = rln.RlnRelayDynamic(context.Background(), w.relay, w.opts.rlnETHClientAddress, w.opts.rlnETHPrivateKey, w.opts.rlnMembershipContractAddress, memKeyPair, w.opts.rlnRelayMemIndex, w.opts.rlnRelayPubsubTopic, w.opts.rlnRelayContentTopic, w.opts.rlnSpamHandler, w.log)
		if err != nil {
			return err
		}
	}

	w.log.Info("mounted waku RLN relay", zap.String("pubsubTopic", w.opts.rlnRelayPubsubTopic), zap.String("contentTopic", w.opts.rlnRelayContentTopic))

	return nil
}
