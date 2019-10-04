package gethbridge

import (
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethMessageStoreWrapper struct {
	messageStore whisper.MessageStore
}

// NewGethMessageStoreWrapper returns an object that wraps Geth's MessageStore in a whispertypes interface
func NewGethMessageStoreWrapper(messageStore whisper.MessageStore) whispertypes.MessageStore {
	if messageStore == nil {
		panic("messageStore cannot be nil")
	}

	return &gethMessageStoreWrapper{
		messageStore: messageStore,
	}
}

// GetGethMessageStoreFrom retrieves the underlying whisper MessageStore interface from a wrapped MessageStore interface
func GetGethMessageStoreFrom(m whispertypes.MessageStore) whisper.MessageStore {
	return m.(*gethMessageStoreWrapper).messageStore
}

func (w *gethMessageStoreWrapper) Add(m whispertypes.ReceivedMessage) error {
	return w.messageStore.Add(GetGethReceivedMessageFrom(m))
}

func (w *gethMessageStoreWrapper) Pop() ([]whispertypes.ReceivedMessage, error) {
	msgs, err := w.messageStore.Pop()
	if err != nil {
		return nil, err
	}

	wrappedMsgs := make([]whispertypes.ReceivedMessage, len(msgs))
	for index, m := range msgs {
		wrappedMsgs[index] = NewGethReceivedMessageWrapper(m)
	}
	return wrappedMsgs, err
}
