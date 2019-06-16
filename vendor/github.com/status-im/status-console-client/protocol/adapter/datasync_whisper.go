package adapter

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"sort"

	"github.com/gogo/protobuf/proto"

	"github.com/status-im/mvds/node"
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	dstrns "github.com/status-im/mvds/transport"

	dspeer "github.com/status-im/status-console-client/protocol/datasync/peer"
	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"

	whisper "github.com/status-im/whisper/whisperv6"
)

type PacketHandler interface {
	AddPacket(dstrns.Packet)
}

type DataSyncWhisperAdapter struct {
	node      *node.Node
	transport transport.WhisperTransport
	packets   PacketHandler
}

// DataSyncWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*DataSyncWhisperAdapter)(nil)

func NewDataSyncWhisperAdapter(n *node.Node, t transport.WhisperTransport, h PacketHandler) *DataSyncWhisperAdapter {
	return &DataSyncWhisperAdapter{
		node:      n,
		transport: t,
		packets:   h,
	}
}

// Subscribe listens to new messages.
func (w *DataSyncWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*subscription.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter := newFilter(w.transport.KeysManager())
	if err := updateFilterFromSubscribeOptions(filter, options); err != nil {
		return nil, err
	}

	// Messages income in batches and hence a buffered channel is used.
	in := make(chan *whisper.ReceivedMessage, 1024)
	sub, err := w.transport.Subscribe(ctx, in, filter.Filter)
	if err != nil {
		return nil, err
	}

	go func() {
		for item := range in {
			payload, err := w.decodePayload(item)
			if err != nil {
				log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
				continue
			}

			packet := dstrns.Packet{
				Group:   toGroupId(item.Topic),
				Sender:  dspeer.PublicKeyToPeerID(*item.Src),
				Payload: payload,
			}
			w.packets.AddPacket(packet)

			for _, m := range w.decodeMessages(payload) {
				m.SigPubKey = item.Src
				messages <- m
			}
		}
	}()

	return sub, nil
}

func (w *DataSyncWhisperAdapter) decodePayload(message *whisper.ReceivedMessage) (payload protobuf.Payload, err error) {
	err = proto.Unmarshal(message.Payload, &payload)
	return
}

func (w *DataSyncWhisperAdapter) decodeMessages(payload protobuf.Payload) []*protocol.Message {
	messages := make([]*protocol.Message, 0)

	for _, message := range payload.Messages {
		decoded, err := protocol.DecodeMessage(message.Body)
		if err != nil {
			// @todo log or something?
			continue
		}

		id := state.ID(*message)
		decoded.ID = id[:]

		messages = append(messages, &decoded)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Clock < messages[j].Clock
	})

	return messages
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (w *DataSyncWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if options.ChatName == "" {
		return nil, errors.New("missing chat name")
	}

	topic, err := ToTopic(options.ChatName)
	if err != nil {
		return nil, err
	}

	gid := toGroupId(topic)

	w.peer(gid, options.Recipient)

	id, err := w.node.AppendMessage(gid, data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

// Request retrieves historic messages.
func (m *DataSyncWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func (c *DataSyncWhisperAdapter) peer(id state.GroupID, peer *ecdsa.PublicKey) {
	if peer == nil {
		return
	}

	p := dspeer.PublicKeyToPeerID(*peer)

	if c.node.IsPeerInGroup(id, p) {
		return
	}

	c.node.AddPeer(id, p)
}

func toGroupId(topicType whisper.TopicType) state.GroupID {
	g := state.GroupID{}
	copy(g[:], topicType[:])
	return g
}
