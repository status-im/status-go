package protocol

import (
	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/messaging/transport"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type MessagesIterator interface {
	HasNext() bool
	Next() (transport.Filter, []*wakutypes.Message)
}

type DefaultMessagesIterator struct {
	chatWithMessages map[transport.Filter][]*wakutypes.Message
	keys             []transport.Filter
	currentIndex     int
}

func NewDefaultMessagesIterator(chatWithMessages map[transport.Filter][]*wakutypes.Message) MessagesIterator {
	return &DefaultMessagesIterator{
		chatWithMessages: chatWithMessages,
		keys:             maps.Keys(chatWithMessages),
		currentIndex:     0,
	}
}

func (it *DefaultMessagesIterator) HasNext() bool {
	return it.currentIndex < len(it.keys)
}

func (it *DefaultMessagesIterator) Next() (transport.Filter, []*wakutypes.Message) {
	if it.HasNext() {
		key := it.keys[it.currentIndex]
		it.currentIndex++
		return key, it.chatWithMessages[key]
	}
	return transport.Filter{}, nil
}
