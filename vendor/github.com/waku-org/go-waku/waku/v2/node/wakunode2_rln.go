//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/static"
	r "github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

// RLNRelay is used to access any operation related to Waku RLN protocol
func (w *WakuNode) RLNRelay() RLNRelay {
	return w.rlnRelay
}

func (w *WakuNode) mountRlnRelay(ctx context.Context) error {
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
		groupKeys, idCredential, err := static.Setup(w.opts.rlnRelayMemIndex)
		if err != nil {
			return err
		}

		// rlnrelay in off-chain mode with a static group of user

		groupManager, err := static.NewStaticGroupManager(groupKeys, idCredential, w.opts.rlnRelayMemIndex, w.log)
		if err != nil {
			return err
		}

		rlnRelay, err := rln.New(w.Relay(), groupManager, w.opts.rlnRelayPubsubTopic, w.opts.rlnRelayContentTopic, w.opts.rlnSpamHandler, w.timesource, w.log)
		if err != nil {
			return err
		}

		err = rlnRelay.Start(ctx)
		if err != nil {
			return err
		}

		w.rlnRelay = rlnRelay

		// check the correct construction of the tree by comparing the calculated root against the expected root
		// no error should happen as it is already captured in the unit tests
		root, err := rlnRelay.RLN.GetMerkleRoot()
		if err != nil {
			return err
		}

		expectedRoot, err := r.ToBytes32LE(r.STATIC_GROUP_MERKLE_ROOT)
		if err != nil {
			return err
		}

		if !bytes.Equal(expectedRoot[:], root[:]) {
			return errors.New("root mismatch: something went wrong not in Merkle tree construction")
		}

		w.log.Debug("the calculated root", zap.String("root", hex.EncodeToString(root[:])))
	} else {
		w.log.Info("setting up waku-rln-relay in on-chain mode")

		/*//  check if the peer has provided its rln credentials
		var memKeyPair *r.IdentityCredential
		if w.opts.rlnRelayIDCommitment != nil && w.opts.rlnRelayIDKey != nil {
			memKeyPair = &r.IdentityCredential{
				IDCommitment: *w.opts.rlnRelayIDCommitment,
				IDSecretHash: *w.opts.rlnRelayIDKey,
			}
		}

		// mount the rln relay protocol in the on-chain/dynamic mode
		var err error
		w.rlnRelay, err = rln.RlnRelayDynamic(ctx, w.Relay(), w.opts.rlnETHClientAddress, w.opts.rlnETHPrivateKey, w.opts.rlnMembershipContractAddress, memKeyPair, w.opts.rlnRelayMemIndex, w.opts.rlnRelayPubsubTopic, w.opts.rlnRelayContentTopic, w.opts.rlnSpamHandler, w.opts.rlnRegistrationHandler, w.timesource, w.log)
		if err != nil {
			return err
		}*/
	}

	w.log.Info("mounted waku RLN relay", zap.String("pubsubTopic", w.opts.rlnRelayPubsubTopic), zap.String("contentTopic", w.opts.rlnRelayContentTopic))

	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	if w.rlnRelay != nil {
		w.rlnRelay.Stop()
	}
	return nil
}
