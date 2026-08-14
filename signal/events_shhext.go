package signal

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/crypto/types"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
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

	// EventNewMessages is triggered when we receive new messages
	EventNewMessages = "messages.new"

	// EventLocalMessageBackupDone is triggered when a local message backup is completed
	EventLocalMessageBackupDone = "local.message.backup.done"

	// EventHistoryRequestStarted is triggered before processing a store request
	EventHistoryRequestStarted = "history.request.started"

	// EventHistoryRequestCompleted is triggered after processing all storenode requests
	EventHistoryRequestCompleted = "history.request.completed"

	// EventBackupPerformed is triggered when a backup has been performed
	EventBackupPerformed = "backup.performed"

	// EventUpdateAvailable is triggered after a update verification is performed
	EventUpdateAvailable = "update.available"
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

type HistoryMessagesSignal struct {
	RequestID  string `json:"requestId"`
	PeerID     string `json:"peerId"`
	BatchIndex int    `json:"batchIndex"`
	NumBatches int    `json:"numBatches,omitempty"`
	ErrorMsg   string `json:"errorMessage,omitempty"`
}

type UpdateAvailableSignal struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	URL       string `json:"url"`
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
	Topic types2.ContentTopic `json:"topic"`
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

func SendHistoricMessagesRequestStarted(numBatches int) {
	send(EventHistoryRequestStarted, HistoryMessagesSignal{NumBatches: numBatches})
}

func SendHistoricMessagesRequestCompleted() {
	send(EventHistoryRequestCompleted, HistoryMessagesSignal{})
}

func SendUpdateAvailable(available bool, latestVersion string, url string) {
	send(EventUpdateAvailable, UpdateAvailableSignal{Available: available, Version: latestVersion, URL: url})
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

func SendNewMessages(obj json.Marshaler) {
	send(EventNewMessages, obj)
}

func LocalMessageBackupDone() {
	send(EventLocalMessageBackupDone, interface{}(nil))
}
