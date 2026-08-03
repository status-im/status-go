package signal

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// EventMessagesSent is triggered when messages are confirmed as sent into the network
	EventMessagesSent = "messages.sent"

	// EventMessagesExpired is triggered when messages failed to be sent
	EventMessagesExpired = "messages.expired"

	// EventNewMessages is triggered when we receive new messages
	EventNewMessages = "messages.new"

	// EventLocalMessageBackupDone is triggered when a local message backup is completed
	EventLocalMessageBackupDone = "local.message.backup.done"

	// EventHistoryRequestStarted is triggered before processing a store request
	EventHistoryRequestStarted = "history.request.started"

	// EventHistoryRequestCompleted is triggered after processing all storenode requests
	EventHistoryRequestCompleted = "history.request.completed"

	// EventUpdateAvailable is triggered after a update verification is performed
	EventUpdateAvailable = "update.available"
)

// MessagesSignal includes the message identifiers the event refers to.
type MessagesSignal struct {
	IDs     []hexutil.Bytes `json:"ids"`
	Message string          `json:"message,omitempty"`
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

// SendMessagesSent notifies that messages were confirmed as sent into the network.
func SendMessagesSent(identifiers [][]byte) {
	var hexIdentifiers []hexutil.Bytes
	for _, i := range identifiers {
		hexIdentifiers = append(hexIdentifiers, i)
	}

	send(EventMessagesSent, MessagesSignal{
		IDs: hexIdentifiers,
	})
}

// SendMessagesExpired notifies that messages failed to be sent.
func SendMessagesExpired(identifiers [][]byte, err error) {
	var message string
	if err != nil {
		message = err.Error()
	}
	var hexIdentifiers []hexutil.Bytes
	for _, i := range identifiers {
		hexIdentifiers = append(hexIdentifiers, i)
	}

	send(EventMessagesExpired, MessagesSignal{IDs: hexIdentifiers, Message: message})
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

func SendNewMessages(obj json.Marshaler) {
	send(EventNewMessages, obj)
}

func LocalMessageBackupDone() {
	send(EventLocalMessageBackupDone, interface{}(nil))
}
