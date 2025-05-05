package protocol

import (
	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/messaging"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type MessagesIterator interface {
	HasNext() bool
	Next() (messaging.ChatFilter, []*wakutypes.Message)
}

type DefaultMessagesIterator struct {
	chatWithMessages map[messaging.ChatFilter][]*wakutypes.Message
	keys             []messaging.ChatFilter
	currentIndex     int
}

func NewDefaultMessagesIterator(chatWithMessages map[messaging.ChatFilter][]*wakutypes.Message) MessagesIterator {
	return &DefaultMessagesIterator{
		chatWithMessages: chatWithMessages,
		keys:             maps.Keys(chatWithMessages),
		currentIndex:     0,
	}
}

func (it *DefaultMessagesIterator) HasNext() bool {
	return it.currentIndex < len(it.keys)
}

func (it *DefaultMessagesIterator) Next() (messaging.ChatFilter, []*wakutypes.Message) {
	if it.HasNext() {
		key := it.keys[it.currentIndex]
		it.currentIndex++
		return key, it.chatWithMessages[key]
	}
	return messaging.ChatFilter{}, nil
}
