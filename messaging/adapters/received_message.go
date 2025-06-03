package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func ToWakuMessage(m *types.ReceivedMessage) *wakutypes.Message {
	if m == nil {
		return nil
	}
	return &wakutypes.Message{
		Sig:          m.Sig,
		Timestamp:    m.Timestamp,
		Topic:        ToWakuTopic(m.Topic),
		Payload:      m.Payload,
		Padding:      m.Padding,
		Hash:         m.Hash,
		Dst:          m.Dst,
		ThirdPartyID: m.ThirdPartyID,
	}
}

func FromWakuMessage(m *wakutypes.Message) *types.ReceivedMessage {
	if m == nil {
		return nil
	}
	return &types.ReceivedMessage{
		Sig:          m.Sig,
		Timestamp:    m.Timestamp,
		Topic:        FromWakuTopic(m.Topic),
		Payload:      m.Payload,
		Padding:      m.Padding,
		Hash:         m.Hash,
		Dst:          m.Dst,
		ThirdPartyID: m.ThirdPartyID,
	}
}

func FromWakuMessages(messages []*wakutypes.Message) []*types.ReceivedMessage {
	if messages == nil {
		return nil
	}
	receivedMessages := make([]*types.ReceivedMessage, len(messages))
	for i, m := range messages {
		receivedMessages[i] = FromWakuMessage(m)
	}
	return receivedMessages
}
