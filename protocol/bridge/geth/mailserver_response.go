package gethbridge

import (
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethMailServerResponseWrapper returns a whispertypes.MailServerResponse object that mimics Geth's MailServerResponse
func NewGethMailServerResponseWrapper(mailServerResponse *whisper.MailServerResponse) *whispertypes.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &whispertypes.MailServerResponse{
		LastEnvelopeHash: protocol.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}
