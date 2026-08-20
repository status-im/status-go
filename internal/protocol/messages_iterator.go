package protocol

import (
	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/pkg/messaging/types"
)

type MessagesIterator interface {
	HasNext() bool
	Next() (types.ChatFilter, []*types.ReceivedMessage)
}

type DefaultMessagesIterator struct {
	chatWithMessages map[types.ChatFilter][]*types.ReceivedMessage
	keys             []types.ChatFilter
	currentIndex     int
}

func NewDefaultMessagesIterator(chatWithMessages map[types.ChatFilter][]*types.ReceivedMessage) MessagesIterator {
	return &DefaultMessagesIterator{
		chatWithMessages: chatWithMessages,
		keys:             maps.Keys(chatWithMessages),
		currentIndex:     0,
	}
}

func (it *DefaultMessagesIterator) HasNext() bool {
	return it.currentIndex < len(it.keys)
}

func (it *DefaultMessagesIterator) Next() (types.ChatFilter, []*types.ReceivedMessage) {
	if it.HasNext() {
		key := it.keys[it.currentIndex]
		it.currentIndex++
		return key, it.chatWithMessages[key]
	}
	return types.ChatFilter{}, nil
}
