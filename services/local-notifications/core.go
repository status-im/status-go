package localnotifications

import (
	"context"
	"math/big"
	"sync"

	//TODO: Inspect replacement with go-ethereum/events
	messagebus "github.com/vardius/message-bus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
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

// Service keeps the state of message bus
type Service struct {
	started            bool
	bus                messagebus.MessageBus
	ctx                context.Context
	db                 *Database
	walletSubscription chan struct{}
	walletWg           sync.WaitGroup
}

func NewService(db *Database, queueSize int) *Service {
	return &Service{
		db:  db,
		bus: messagebus.New(queueSize),
		ctx: context.Background(),
	}
}

func pushMessage(notification *Notification) {
	signal.SendLocalNotifications(notification)
}

func buildTransactionNotification(payload messagePayload) *Notification {
	// GET Transfers by address and block, to build from it the notification
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

// PublishMessage - Send new message to be processed into a notification
func (s *Service) PublishMessage(messageType MessageType, payload interface{}) {
	s.bus.Publish(string(messageType), s.ctx, messagePayload(payload))
}

// Start Worker which processes all incoming messages
func (s *Service) Start(*p2p.Server) error {
	s.started = true

	if err := s.bus.Subscribe(string(Transaction), transactionsHandler); err != nil {
		log.Error("Could not create subscription", "error", err)
		return err
	}
	return nil
}

// SubscribeWallet - Subscribes to wallet signals and redirects them into Notifications
func (s *Service) SubscribeWallet(publisher *event.Feed) error {
	if s.walletSubscription != nil {
		// already running, nothing to do
		return nil
	}
	s.walletSubscription = make(chan struct{})
	events := make(chan wallet.Event, 10)
	sub := publisher.Subscribe(events)

	s.walletWg.Add(1)
	go func() {
		defer s.walletWg.Done()
		for {
			select {
			case <-s.walletSubscription:
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				// technically event.Feed cannot send an error to subscription.Err channel.
				// the only time we will get an event is when that channel is closed.
				if err != nil {
					log.Error("wallet signals transmitter failed with", "error", err)
				}
				return
			case event := <-events:
				log.Info("Process event:", event)
				s.PublishMessage(Transaction, TransactionEvent{
					Type: string(event.Type),
				})
			}
		}
	}()
	return nil
}

// Stop worker
func (s *Service) Stop() error {
	s.started = false
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "notifications",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) IsStarted() bool {
	return s.started
}
