// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package waku

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

// (Waku.Post was removed: all sends now go through Send with payloads
// pre-encoded by the transport via rfc26.Encode. Reception was moved to the
// transport too: the adapter forwards raw envelopes on the envelope feed
// (EventEnvelopeAvailable) and no longer decodes/polls — status-im/status-go#7464.)

// Send publishes a pre-encoded payload to the messaging network. The payload
// is expected to be already RFC26-encoded (see transport/rfc26.Encode); this
// method just wraps it in a WakuMessage envelope and hands it to the publish
// path. The `version` field is a wire label independent of that encoding, and
// is advertised as 0. Returns the wire hash.
//
// ctx is accepted for symmetry with transport.MessagingAPI; the send queue
// uses the waku instance's own lifecycle context.
func (w *Waku) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	var version uint32 = 0 // wire label only; the payload is RFC26-encoded by the caller
	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      &version,
		ContentTopic: contentTopic,
		Timestamp:    proto.Int64(w.timestamp()),
		Meta:         []byte{}, // TODO: empty for now. Once we use Waku Archive v2, we should deprecate the timestamp and use an ULID here
		Ephemeral:    &ephemeral,
	}
	return w.sendEnvelope(pubsubTopic, msg, priority)
}
