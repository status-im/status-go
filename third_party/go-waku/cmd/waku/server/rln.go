//go:build !gowaku_no_rln
// +build !gowaku_no_rln

package server

import (
	"fmt"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
)

func AppendRLNProof(node *node.WakuNode, msg *pb.WakuMessage) error {
	_, rlnEnabled := node.RLNRelay().(*rln.WakuRLNRelay)
	if rlnEnabled {
		err := node.RLNRelay().AppendRLNProof(msg, node.Timesource().Now())
		if err != nil {
			return fmt.Errorf("could not append rln proof: %w", err)
		}
	}
	return nil
}
