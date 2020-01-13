package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/whisper/v6"
)

// NewWhisperMailServerResponseWrapper returns a types.MailServerResponse object that mimics Geth's MailServerResponse
func NewWhisperMailServerResponseWrapper(mailServerResponse *whisper.MailServerResponse) *types.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &types.MailServerResponse{
		LastEnvelopeHash: types.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}

// NewWakuMailServerResponseWrapper returns a types.MailServerResponse object that mimics Geth's MailServerResponse
func NewWakuMailServerResponseWrapper(mailServerResponse *waku.MailServerResponse) *types.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &types.MailServerResponse{
		LastEnvelopeHash: types.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}
