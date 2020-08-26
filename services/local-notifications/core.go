package localnotifications

import (
	"context"
	"math/big"

	messagebus "github.com/vardius/message-bus"

	"github.com/status-im/status-go/signal"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
)

type messagePayload interface{}

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

// TransactionEvent - structure used to pass messages from wallet to bus
type TransactionEvent struct {
	Type                      string
	BlockNumber               *big.Int
	Accounts                  []common.Address
	NewTransactionsPerAccount map[common.Address]int
	ERC20                     bool
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
	signal.SendLocalNotifications(notification)
}

func buildTransactionNotification(payload messagePayload) *Notification {
	transaction := payload.(TransactionEvent)
	body := notificationBody{
		State: transaction.Type,
		From:  transaction.Type,
		To:    transaction.Type,
		Value: transaction.Type,
	}

	return &Notification{
		ID:       string(transaction.Type),
		Body:     body,
		Category: "transaction",
	}
}

func transactionsHandler(ctx context.Context, payload messagePayload) {
	log.Info("Handled a new transactopn")
	n := buildTransactionNotification(payload)
	pushMessage(n)
}

// NotificationWorker - Worker which processes all incoming messages
func (s *Broker) NotificationWorker() error {

	if err := s.bus.Subscribe(string(Transaction), transactionsHandler); err != nil {
		log.Error("Could not create subscription", "error", err)
		return err
	}

	return nil
}

// PublishMessage - Send new message to be processed into a notification
func (s *Broker) PublishMessage(messageType MessageType, payload interface{}) {
	s.bus.Publish(string(messageType), s.ctx, messagePayload(payload))
}
