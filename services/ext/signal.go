package ext

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/signal"
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
func (h EnvelopeSignalHandler) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandler) MailServerRequestExpired(hash types.Hash) {
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

func (h PublisherSignalHandler) FilterAdded(filters []*signal.Filter) {
	// TODO(waku): change the name of the filter to generic one.
	signal.SendWhisperFilterAdded(filters)
}

func (h PublisherSignalHandler) NewMessages(response *protocol.MessengerResponse) {
	signal.SendNewMessages(response)
}

// MessengerSignalHandler sends signals on messenger events
type MessengerSignalsHandler struct{}

// MessageDelivered passes information that message was delivered
func (m MessengerSignalsHandler) MessageDelivered(chatID string, messageID string) {
	signal.SendMessageDelivered(chatID, messageID)
}

// MessageDelivered passes info about community that was requested before
func (m MessengerSignalsHandler) CommunityInfoFound(community *communities.Community) {
	signal.SendCommunityInfoFound(community)
}
