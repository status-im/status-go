package shhext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/signal"
)

// EnvelopeSignalHandler sends signals when envelope is sent or expired.
type EnvelopeSignalHandler struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(hash common.Hash) {
	signal.SendEnvelopeSent(hash)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(hash common.Hash) {
	signal.SendEnvelopeExpired(hash)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID common.Hash, lastEnvelopeHash common.Hash, cursor []byte) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash common.Hash) {
	signal.SendMailServerRequestExpired(hash)
}

func (h EnvelopeSignalHandler) DecryptMessageFailed(pubKey string) {
	signal.SendDecryptMessageFailed(pubKey)
}

func (h EnvelopeSignalHandler) BundleAdded(identity string, installationID string) {
	signal.SendBundleAdded(identity, installationID)
}
