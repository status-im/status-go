package gethbridge

import (
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

// NewGethMailServerResponseWrapper returns a whispertypes.MailServerResponse object that mimics Geth's MailServerResponse
func NewGethMailServerResponseWrapper(mailServerResponse *whisper.MailServerResponse) *whispertypes.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &whispertypes.MailServerResponse{
		LastEnvelopeHash: statusproto.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}
