package shhext

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// Post shamelessly copied from whisper codebase with slight modifications.
func (api *PublicAPI) Post(ctx context.Context, req whisper.NewMessage) (hash common.Hash, err error) {
	env, err := whisper.MakeEnvelope(api.service.w, req)
	if err != nil {
		return hash, err
	}
	// send to specific node (skip PoW check)
	if len(req.TargetPeer) > 0 {
		n, err := discover.ParseNode(req.TargetPeer)
		if err != nil {
			return hash, fmt.Errorf("failed to parse target peer: %s", err)
		}
		err = api.service.w.SendP2PMessage(n.ID[:], env)
		if err == nil {
			api.service.tracker.Add(env.Hash())
			return env.Hash(), nil
		}
		return hash, err
	}

	// ensure that the message PoW meets the node's minimum accepted PoW
	if req.PowTarget < api.service.w.MinPow() {
		return hash, whisper.ErrTooLowPoW
	}
	err = api.service.w.Send(env)
	if err == nil {
		api.service.tracker.Add(env.Hash())
		return env.Hash(), nil
	}
	return hash, err
}
