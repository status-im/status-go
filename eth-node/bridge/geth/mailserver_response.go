package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/wakuv1"
)

// NewWakuMailServerResponseWrapper returns a types.MailServerResponse object that mimics Geth's MailServerResponse
func NewWakuMailServerResponseWrapper(mailServerResponse *wakuv1.MailServerResponse) *types.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &types.MailServerResponse{
		LastEnvelopeHash: types.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}
