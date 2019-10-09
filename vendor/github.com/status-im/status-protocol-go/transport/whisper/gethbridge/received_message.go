package gethbridge

import (
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethReceivedMessageWrapper struct {
	receivedMessage *whisper.ReceivedMessage
}

// NewGethReceivedMessageWrapper returns an object that wraps Geth's ReceivedMessage in a whispertypes interface
func NewGethReceivedMessageWrapper(receivedMessage *whisper.ReceivedMessage) whispertypes.ReceivedMessage {
	if receivedMessage == nil {
		panic("receivedMessage cannot be nil")
	}

	return &gethReceivedMessageWrapper{
		receivedMessage: receivedMessage,
	}
}

// GetGethReceivedMessageFrom retrieves the underlying whisper ReceivedMessage struct from a wrapped ReceivedMessage interface
func GetGethReceivedMessageFrom(m whispertypes.ReceivedMessage) *whisper.ReceivedMessage {
	return m.(*gethReceivedMessageWrapper).receivedMessage
}
