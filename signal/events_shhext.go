package signal

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	// EventEnvelopeSent is triggered when envelope was sent at least to a one peer.
	EventEnvelopeSent = "envelope.sent"

	// EventEnvelopeExpired is triggered when envelop was dropped by a whisper without being sent
	// to any peer
	EventEnvelopeExpired = "envelope.expired"

	// EventMailServerRequestCompleted is triggered when whisper receives a message ack from the mailserver
	EventMailServerRequestCompleted = "mailserver.request.completed"

	// EventMailServerRequestExpired is triggered when request TTL ends
	EventMailServerRequestExpired = "mailserver.request.expired"

	// EventEnodeDiscovered is tiggered when enode has been discovered.
	EventEnodeDiscovered = "enode.discovered"

	// EventDecryptMessageFailed is triggered when we receive a message from a bundle we don't have
	EventDecryptMessageFailed = "messages.decrypt.failed"

	// EventBundleAdded is triggered when we receive a bundle
	EventBundleAdded = "bundles.added"
)

// EnvelopeSignal includes hash of the envelope.
type EnvelopeSignal struct {
	Hash common.Hash `json:"hash"`
}

// MailServerResponseSignal holds the data received in the response from the mailserver.
type MailServerResponseSignal struct {
	RequestID        common.Hash `json:"requestID"`
	LastEnvelopeHash common.Hash `json:"lastEnvelopeHash"`
	Cursor           string      `json:"cursor"`
}

// DecryptMessageFailedSignal holds the sender of the message that could not be decrypted
type DecryptMessageFailedSignal struct {
	Sender string `json:"sender"`
}

// BundleAddedSignal holds the identity and installation id of the user
type BundleAddedSignal struct {
	Identity       string `json:"identity"`
	InstallationID string `json:"installationID"`
}

// SendEnvelopeSent triggered when envelope delivered at least to 1 peer.
func SendEnvelopeSent(hash common.Hash) {
	send(EventEnvelopeSent, EnvelopeSignal{hash})
}

// SendEnvelopeExpired triggered when envelope delivered at least to 1 peer.
func SendEnvelopeExpired(hash common.Hash) {
	send(EventEnvelopeExpired, EnvelopeSignal{hash})
}

// SendMailServerRequestCompleted triggered when mail server response has been received
func SendMailServerRequestCompleted(requestID common.Hash, lastEnvelopeHash common.Hash, cursor []byte) {
	sig := MailServerResponseSignal{
		RequestID:        requestID,
		LastEnvelopeHash: lastEnvelopeHash,
		Cursor:           string(cursor),
	}
	send(EventMailServerRequestCompleted, sig)
}

// SendMailServerRequestExpired triggered when mail server request expires
func SendMailServerRequestExpired(hash common.Hash) {
	send(EventMailServerRequestExpired, EnvelopeSignal{hash})
}

// EnodeDiscoveredSignal includes enode address and topic
type EnodeDiscoveredSignal struct {
	Enode string `json:"enode"`
	Topic string `json:"topic"`
}

// SendEnodeDiscovered tiggered when an enode is discovered.
// finds a new enode.
func SendEnodeDiscovered(enode, topic string) {
	send(EventEnodeDiscovered, EnodeDiscoveredSignal{
		Enode: enode,
		Topic: topic,
	})
}

func SendDecryptMessageFailed(sender string) {
	send(EventDecryptMessageFailed, DecryptMessageFailedSignal{sender})
}

func SendBundleAdded(identity string, installationID string) {
	send(EventBundleAdded, BundleAddedSignal{Identity: identity, InstallationID: installationID})
}
