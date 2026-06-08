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

package wakuv2

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging/waku/common"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// (Waku.Post was removed: all sends now go through Send with payloads
// pre-encoded by the transport via rfc26.Encode.)

// Send publishes a pre-encoded payload to the messaging network. The payload
// is expected to be already encoded for WakuMessage version=1 (see
// transport/rfc26.Encode); this method just wraps it in a WakuMessage
// envelope and hands it to the publish path. Returns the wire hash.
//
// ctx is accepted for symmetry with transport.MessagingAPI; the send queue
// uses the waku instance's own lifecycle context.
func (w *Waku) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	var version uint32 = 1 // wire-format discriminator; v1 encryption is applied by the caller
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

// ToWakuMessage converts an internal message into an API version.
func ToWakuMessage(message *common.ReceivedMessage) *types.Message {
	msg := types.Message{
		Payload:   message.Data,
		Padding:   message.Padding,
		Timestamp: message.Sent,
		Hash:      message.Hash().Bytes(),
		Topic:     types.TopicType(message.ContentTopic),
	}

	if message.Dst != nil {
		b := crypto.FromECDSAPub(message.Dst)
		if b != nil {
			msg.Dst = b
		}
	}

	if message.Src != nil {
		b := crypto.FromECDSAPub(message.Src)
		if b != nil {
			msg.Sig = b
		}
	}

	return &msg
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (w *Waku) GetFilterMessages(id string) ([]*types.Message, error) {
	w.getFilterMessagesMu.Lock()
	f := w.getFilter(id)
	if f == nil {
		w.getFilterMessagesMu.Unlock()
		return nil, fmt.Errorf("filter not found")
	}
	w.getFilterMessagesMu.Unlock()

	receivedMessages := f.Retrieve()
	messages := make([]*types.Message, 0, len(receivedMessages))
	for _, msg := range receivedMessages {
		messages = append(messages, ToWakuMessage(msg))
	}

	return messages, nil
}
