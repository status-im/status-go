package bridge

import (
	"github.com/status-im/status-go/eth-node/types"
	wakutypes "github.com/status-im/status-go/waku/types"

	"github.com/status-im/status-go/wakuv1"
)

// NewWakuMailServerResponseWrapper returns a wakutypes.MailServerResponse object that mimics Geth's MailServerResponse
func NewWakuMailServerResponseWrapper(mailServerResponse *wakuv1.MailServerResponse) *wakutypes.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &wakutypes.MailServerResponse{
		LastEnvelopeHash: types.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}
