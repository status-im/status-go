package filter

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filtertrace"
)

func traceEnvelope(stage string, remote peer.ID, env *protocol.Envelope) filtertrace.Event {
	e := filtertrace.Event{Stage: stage}
	if remote != "" {
		e.RemotePeerID = remote.String()
	}
	if env != nil {
		e.EnvelopeHash = env.Hash().String()
		e.PubsubTopic = env.PubsubTopic()
		e.ContentTopic = env.Message().ContentTopic
	}
	return e
}

func (wf *WakuFilterFullNode) observe(e filtertrace.Event) {
	if wf.observer == nil {
		return
	}
	if wf.h != nil {
		e.LocalPeerID = wf.h.ID().String()
	}
	if e.TimestampNS == 0 {
		e.TimestampNS = time.Now().UnixNano()
	}
	wf.observer(e)
}

func (wf *WakuFilterLightNode) observe(e filtertrace.Event) {
	if wf.observer == nil {
		return
	}
	if wf.h != nil {
		e.LocalPeerID = wf.h.ID().String()
	}
	if e.TimestampNS == 0 {
		e.TimestampNS = time.Now().UnixNano()
	}
	wf.observer(e)
}

// TraceObserver lets the Filter API preserve observations through its multiplex
// queue. Nil means tracing is disabled and requires no extra envelope work.
func (wf *WakuFilterLightNode) TraceObserver() filtertrace.Observer {
	if wf == nil || wf.observer == nil {
		return nil
	}
	return wf.observe
}
