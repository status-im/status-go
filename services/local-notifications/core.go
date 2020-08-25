package localnotifications

import (
	"context"
	"encoding/json"
	"fmt"

	messagebus "github.com/vardius/message-bus"
)

type messagePayload []byte

// MessageType - Enum defining types of message handled by the bus
type MessageType string

type PushCategory string

type notificationBody struct {
	State string `json:"state"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
}

type Notification struct {
	ID            string           `json:"id"`
	Platform      float32          `json:"platform,omitempty"`
	Body          notificationBody `json:"body"`
	Category      PushCategory     `json:"category,omitempty"`
	Image         string           `json:"imageUrl,omitempty"`
	IsScheduled   bool             `json:"isScheduled,omitempty"`
	ScheduledTime string           `json:"scheduleTime,omitempty"`
}

const (
	// Transaction - Ethereum transaction
	Transaction MessageType = "transaction"

	// Message - Waku message
	Message MessageType = "message"

	// Custom - User defined notifications
	Custom MessageType = "custom"
)

const topic = "local-notifications"

// Broker keeps the state of message bus
type Broker struct {
	bus messagebus.MessageBus
	ctx context.Context
}

// InitializeBus - Initialize MessageBus which will handle generation of notifications
func InitializeBus(queueSize int) *Broker {
	return &Broker{
		bus: messagebus.New(queueSize),
		ctx: context.Background(),
	}
}

func pushMessage(notification *Notification) {
	// Send signal with this notification
	jsonBody, _ := json.Marshal(notification)
	fmt.Println(string(jsonBody))
}

func buildTransactionNotification(payload messagePayload) *Notification {

	transaction := struct {
		ID    string `json:"id"`
		State string `json:"state"`
		From  string `json:"from"`
		To    string `json:"to"`
		Value string `json:"value"`
	}{}

	if err := json.Unmarshal(payload, &transaction); err != nil {
		return nil
	}

	body := notificationBody{
		State: transaction.State,
		From:  transaction.From,
		To:    transaction.To,
		Value: transaction.Value,
	}

	return &Notification{
		ID:       transaction.ID,
		Body:     body,
		Category: "transaction",
	}
}

// TransactionsHandler - Handles new transaction messages
func TransactionsHandler(ctx context.Context, payload messagePayload) {
	n := buildTransactionNotification(payload)
	pushMessage(n)
}

// NotificationWorker - Worker which processes all incoming messages
func (s *Broker) NotificationWorker() error {

	if err := s.bus.Subscribe(string(Transaction), TransactionsHandler); err != nil {
		return err
	}

	return nil
}

// PublishMessage - Send new message to be processed into a notification
func (s *Broker) PublishMessage(messageType MessageType, payload string) {
	s.bus.Publish(string(messageType), s.ctx, messagePayload(payload))
}
