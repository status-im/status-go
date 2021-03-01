package signal

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/eth-node/types"
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

	// EventWhisperFilterAdded is triggered when we setup a new filter or restore existing ones
	EventWhisperFilterAdded = "whisper.filter.added"

	// EventNewMessages is triggered when we receive new messages
	EventNewMessages = "messages.new"
)

// EnvelopeSignal includes hash of the envelope.
type EnvelopeSignal struct {
	IDs     []hexutil.Bytes `json:"ids"`
	Hash    types.Hash      `json:"hash"`
	Message string          `json:"message"`
}

// MailServerResponseSignal holds the data received in the response from the mailserver.
type MailServerResponseSignal struct {
	RequestID        types.Hash `json:"requestID"`
	LastEnvelopeHash types.Hash `json:"lastEnvelopeHash"`
	Cursor           string     `json:"cursor"`
	ErrorMsg         string     `json:"errorMessage"`
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

type Filter struct {
	// ChatID is the identifier of the chat
	ChatID string `json:"chatId"`
	// SymKeyID is the symmetric key id used for symmetric chats
	SymKeyID string `json:"symKeyId"`
	// OneToOne tells us if we need to use asymmetric encryption for this chat
	Listen bool `json:"listen"`
	// FilterID the whisper filter id generated
	FilterID string `json:"filterId"`
	// Identity is the public key of the other recipient for non-public chats
	Identity string `json:"identity"`
	// Topic is the whisper topic
	Topic types.TopicType `json:"topic"`
}

type WhisperFilterAddedSignal struct {
	Filters []*Filter `json:"filters"`
}

// SendEnvelopeSent triggered when envelope delivered at least to 1 peer.
func SendEnvelopeSent(identifiers [][]byte) {
	var hexIdentifiers []hexutil.Bytes
	for _, i := range identifiers {
		hexIdentifiers = append(hexIdentifiers, i)
	}

	send(EventEnvelopeSent, EnvelopeSignal{
		IDs: hexIdentifiers,
	})
}

// SendEnvelopeExpired triggered when envelope delivered at least to 1 peer.
func SendEnvelopeExpired(identifiers [][]byte, err error) {
	var message string
	if err != nil {
		message = err.Error()
	}
	var hexIdentifiers []hexutil.Bytes
	for _, i := range identifiers {
		hexIdentifiers = append(hexIdentifiers, i)
	}

	send(EventEnvelopeExpired, EnvelopeSignal{IDs: hexIdentifiers, Message: message})
}

// SendMailServerRequestCompleted triggered when mail server response has been received
func SendMailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	sig := MailServerResponseSignal{
		RequestID:        requestID,
		LastEnvelopeHash: lastEnvelopeHash,
		Cursor:           hex.EncodeToString(cursor),
		ErrorMsg:         errorMsg,
	}
	send(EventMailServerRequestCompleted, sig)
}

// SendMailServerRequestExpired triggered when mail server request expires
func SendMailServerRequestExpired(hash types.Hash) {
	send(EventMailServerRequestExpired, EnvelopeSignal{Hash: hash})
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

func SendWhisperFilterAdded(filters []*Filter) {
	send(EventWhisperFilterAdded, WhisperFilterAddedSignal{Filters: filters})
}

func SendNewMessages(obj json.Marshaler) {
	send(EventNewMessages, obj)
}
