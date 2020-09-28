package localnotifications

import (
	"math/big"
	"sync"

	//TODO: Inspect replacement with go-ethereum/events

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
)

type messagePayload interface{}

type PushCategory string

type transactionState string

const (
	undetermined transactionState = "undetermined"
	failed       transactionState = "failed"
	inbound      transactionState = "inbound"
	outbound     transactionState = "outbound"
)

type notificationBody struct {
	State    transactionState `json:"state"`
	From     common.Address   `json:"from"`
	To       common.Address   `json:"to"`
	Value    *hexutil.Big     `json:"value"`
	ERC20    bool             `json:"erc20"`
	Contract common.Address   `json:"contract"`
}

type Notification struct {
	ID            common.Hash      `json:"id"`
	Platform      float32          `json:"platform,omitempty"`
	Body          notificationBody `json:"body"`
	Category      PushCategory     `json:"category,omitempty"`
	Image         string           `json:"imageUrl,omitempty"`
	IsScheduled   bool             `json:"isScheduled,omitempty"`
	ScheduledTime string           `json:"scheduleTime,omitempty"`
}

// TransactionEvent - structure used to pass messages from wallet to bus
type TransactionEvent struct {
	Type                      string                 `json:"type"`
	BlockNumber               *big.Int               `json:"block-number"`
	Accounts                  []common.Address       `json:"accounts"`
	NewTransactionsPerAccount map[common.Address]int `json:"new-transactions"`
	ERC20                     bool                   `json:"erc20"`
}

// MessageEvent - structure used to pass messages from chat to bus
type MessageEvent struct{}

// CustomEvent - structure used to pass custom user set messages to bus
type CustomEvent struct{}

const topic = "local-notifications"

type transmitter struct {
	wg   sync.WaitGroup
	quit chan struct{}
}

// Service keeps the state of message bus
type Service struct {
	started            bool
	bus                *event.Feed
	transmitter        transmitter
	walletTransmitter transmitter
	db                 *Database
	walletDB           *wallet.Database
}

func NewService(db *Database) *Service {
	return &Service{
		db:  db,
		bus: &event.Feed{},
	}
}

func pushMessage(notification *Notification) {
	log.Info("Pushing a new push notification", "info", notification)
	signal.SendLocalNotifications(notification)
}

func buildTransactionNotification(rawTransfer wallet.Transfer) *Notification {
	log.Info("Handled a new transfer in buildTransactionNotification", "info", rawTransfer)

	var state = undetermined
	transfer := wallet.CastToTransferView(rawTransfer)

	switch {
	case transfer.TxStatus == hexutil.Uint64(0):
		state = failed
	case transfer.Address == transfer.To:
		state = inbound
	default:
		state = outbound
	}

	// TODO: Check if From and To have name in wallet
	// TODO: Add chain
	body := notificationBody{
		State:    state,
		From:     transfer.From,
		To:       transfer.Address,
		Value:    transfer.Value,
		ERC20:    string(transfer.Type) == "erc20",
		Contract: transfer.Contract,
	}

	return &Notification{
		ID:       transfer.ID,
		Body:     body,
		Category: "transaction",
	}
}

func (s *Service) transactionsHandler(payload TransactionEvent) {
	log.Info("Handled a new transaction", "info", payload)

	limit := 20
	if payload.BlockNumber != nil {
		for _, address := range payload.Accounts {
			log.Info("Handled transfer for address", "info", address)
			transfers, err := s.walletDB.GetTransfersByAddressAndBlock(address, payload.BlockNumber, int64(limit))
			if err != nil {
				log.Error("Could not fetch transfers", "error", err)
			}
	
			for _, transaction := range transfers {
				n := buildTransactionNotification(transaction)
				pushMessage(n)
			}
		}
	}
}

// SubscribeWallet - Subscribes to wallet signals and redirects them into Notifications
func (s *Service) SubscribeWallet(publisher *event.Feed, walletDB *wallet.Database) error {
	s.walletDB = walletDB

	if s.walletTransmitter.quit != nil {
		// already running, nothing to do
		return nil
	}
	s.walletTransmitter.quit = make(chan struct{})
	events := make(chan wallet.Event, 10)
	sub := publisher.Subscribe(events)

	s.walletTransmitter.wg.Add(1)
	go func() {
		defer s.walletTransmitter.wg.Done()
		for {
			select {
			case <-s.walletTransmitter.quit:
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
				if event.Type == wallet.EventNewBlock {
					s.bus.Send(TransactionEvent{
						Type:                      string(event.Type),
						BlockNumber:               event.BlockNumber,
						Accounts:                  []common.Address(event.Accounts),
						NewTransactionsPerAccount: map[common.Address]int(event.NewTransactionsPerAccount),
						ERC20:                     bool(event.ERC20),
					})
				}
			}
		}
	}()
	return nil
}

// Start Worker which processes all incoming messages
func (s *Service) Start(_ *p2p.Server) error {
	s.started = true

	s.transmitter.quit = make(chan struct{})
	events := make(chan TransactionEvent, 10)
	sub := s.bus.Subscribe(events)

	s.transmitter.wg.Add(1)
	go func() {
		defer s.transmitter.wg.Done()
		for {
			select {
			case <-s.transmitter.quit:
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				if err != nil {
					log.Error("Local notifications transmitter failed with", "error", err)
				}
				return
			case event := <-events:
				s.transactionsHandler(event)
			}
		}
	}()

	log.Info("Successful start")

	return nil
}

// Stop worker
func (s *Service) Stop() error {
	s.started = false
	
	if s.transmitter.quit != nil {
		close(s.transmitter.quit)
		s.transmitter.wg.Wait()
		s.transmitter.quit = nil
	}

	if s.walletTransmitter.quit != nil {
		close(s.walletTransmitter.quit)
		s.walletTransmitter.wg.Wait()
		s.walletTransmitter.quit = nil
	}

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
