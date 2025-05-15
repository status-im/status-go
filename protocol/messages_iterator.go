package protocol

import (
	"golang.org/x/exp/maps"

	messagingtypes "github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type MessagesIterator interface {
	HasNext() bool
	Next() (messagingtypes.ChatFilter, []*wakutypes.Message)
}

type DefaultMessagesIterator struct {
	chatWithMessages map[messagingtypes.ChatFilter][]*wakutypes.Message
	keys             []messagingtypes.ChatFilter
	currentIndex     int
}

func NewDefaultMessagesIterator(chatWithMessages map[messagingtypes.ChatFilter][]*wakutypes.Message) MessagesIterator {
	return &DefaultMessagesIterator{
		chatWithMessages: chatWithMessages,
		keys:             maps.Keys(chatWithMessages),
		currentIndex:     0,
	}
}

func (it *DefaultMessagesIterator) HasNext() bool {
	return it.currentIndex < len(it.keys)
}

func (it *DefaultMessagesIterator) Next() (messagingtypes.ChatFilter, []*wakutypes.Message) {
	if it.HasNext() {
		key := it.keys[it.currentIndex]
		it.currentIndex++
		return key, it.chatWithMessages[key]
	}
	return messagingtypes.ChatFilter{}, nil
}
