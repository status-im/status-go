package shhext

import (
	"github.com/status-im/status-go/signal"
	statusproto "github.com/status-im/status-protocol-go/types"
)

// EnvelopeSignalHandler sends signals when envelope is sent or expired.
type EnvelopeSignalHandler struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(identifiers [][]byte) {
	signal.SendEnvelopeSent(identifiers)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(identifiers [][]byte, err error) {
	signal.SendEnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID statusproto.Hash, lastEnvelopeHash statusproto.Hash, cursor []byte, err error) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash statusproto.Hash) {
	signal.SendMailServerRequestExpired(hash)
}

// PublisherSignalHandler sends signals on protocol events
type PublisherSignalHandler struct{}

func (h PublisherSignalHandler) DecryptMessageFailed(pubKey string) {
	signal.SendDecryptMessageFailed(pubKey)
}

func (h PublisherSignalHandler) BundleAdded(identity string, installationID string) {
	signal.SendBundleAdded(identity, installationID)
}

func (h PublisherSignalHandler) WhisperFilterAdded(filters []*signal.Filter) {
	signal.SendWhisperFilterAdded(filters)
}

func (h PublisherSignalHandler) NewMessages(messages []*signal.Messages) {
	signal.SendNewMessages(messages)
}
