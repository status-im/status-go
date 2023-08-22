//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"bytes"
	"context"
	"errors"

	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager/dynamic"
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

	var err error
	var groupManager rln.GroupManager

	if !w.opts.rlnRelayDynamic {
		w.log.Info("setting up waku-rln-relay in off-chain mode")

		// set up rln relay inputs
		groupKeys, idCredential, err := static.Setup(w.opts.rlnRelayMemIndex)
		if err != nil {
			return err
		}

		groupManager, err = static.NewStaticGroupManager(groupKeys, idCredential, w.opts.rlnRelayMemIndex, w.log)
		if err != nil {
			return err
		}
	} else {
		w.log.Info("setting up waku-rln-relay in on-chain mode")

		groupManager, err = dynamic.NewDynamicGroupManager(
			w.opts.rlnETHClientAddress,
			w.opts.rlnETHPrivateKey,
			w.opts.rlnMembershipContractAddress,
			w.opts.rlnRelayMemIndex,
			w.opts.keystorePath,
			w.opts.keystorePassword,
			w.opts.keystoreIndex,
			true,
			w.opts.rlnRegistrationHandler,
			w.log,
		)
		if err != nil {
			return err
		}
	}

	rlnRelay, err := rln.New(w.Relay(), groupManager, w.opts.rlnTreePath, w.opts.rlnRelayPubsubTopic, w.opts.rlnRelayContentTopic, w.opts.rlnSpamHandler, w.timesource, w.log)
	if err != nil {
		return err
	}

	err = rlnRelay.Start(ctx)
	if err != nil {
		return err
	}

	w.rlnRelay = rlnRelay

	if !w.opts.rlnRelayDynamic {
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
	}

	w.log.Info("mounted waku RLN relay", zap.String("pubsubTopic", w.opts.rlnRelayPubsubTopic), zap.String("contentTopic", w.opts.rlnRelayContentTopic))

	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	if w.rlnRelay != nil {
		return w.rlnRelay.Stop()
	}
	return nil
}
