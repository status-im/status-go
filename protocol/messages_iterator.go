package protocol

import (
	"golang.org/x/exp/maps"

	messagingtypes "github.com/status-im/status-go/messaging/types"
)

type MessagesIterator interface {
	HasNext() bool
	Next() (messagingtypes.ChatFilter, []*messagingtypes.ReceivedMessage)
}

type DefaultMessagesIterator struct {
	chatWithMessages map[messagingtypes.ChatFilter][]*messagingtypes.ReceivedMessage
	keys             []messagingtypes.ChatFilter
	currentIndex     int
}

func NewDefaultMessagesIterator(chatWithMessages map[messagingtypes.ChatFilter][]*messagingtypes.ReceivedMessage) MessagesIterator {
	return &DefaultMessagesIterator{
		chatWithMessages: chatWithMessages,
		keys:             maps.Keys(chatWithMessages),
		currentIndex:     0,
	}
}

func (it *DefaultMessagesIterator) HasNext() bool {
	return it.currentIndex < len(it.keys)
}

func (it *DefaultMessagesIterator) Next() (messagingtypes.ChatFilter, []*messagingtypes.ReceivedMessage) {
	if it.HasNext() {
		key := it.keys[it.currentIndex]
		it.currentIndex++
		return key, it.chatWithMessages[key]
	}
	return messagingtypes.ChatFilter{}, nil
}
