package localnotifications

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
)

type PushCategory string

type transactionState string

type NotificationType string

const (
	walletDeeplinkPrefix = "status-im://wallet/"

	failed   transactionState = "failed"
	inbound  transactionState = "inbound"
	outbound transactionState = "outbound"

	CategoryTransaction PushCategory = "transaction"
	CategoryMessage     PushCategory = "newMessage"

	TypeTransaction NotificationType = "transaction"
	TypeMessage     NotificationType = "message"
)

var (
	marshalTypeMismatchErr = "notification type mismatch, expected '%s', Body could not be marshalled into this type"
)

type notificationBody struct {
	State       transactionState  `json:"state"`
	From        common.Address    `json:"from"`
	To          common.Address    `json:"to"`
	FromAccount *accounts.Account `json:"fromAccount,omitempty"`
	ToAccount   *accounts.Account `json:"toAccount,omitempty"`
	Value       *hexutil.Big      `json:"value"`
	ERC20       bool              `json:"erc20"`
	Contract    common.Address    `json:"contract"`
	Network     uint64            `json:"network"`
}

type Notification struct {
	ID            common.Hash `json:"id"`
	Platform      float32     `json:"platform,omitempty"`
	Body          interface{}
	BodyType      NotificationType `json:"bodyType"`
	Category      PushCategory     `json:"category,omitempty"`
	Deeplink      string           `json:"deepLink,omitempty"`
	Image         string           `json:"imageUrl,omitempty"`
	IsScheduled   bool             `json:"isScheduled,omitempty"`
	ScheduledTime string           `json:"scheduleTime,omitempty"`
}

// notificationAlias is an interim struct used for json un/marshalling
type notificationAlias struct {
	ID            common.Hash      `json:"id"`
	Platform      float32          `json:"platform,omitempty"`
	Body          json.RawMessage  `json:"body"`
	BodyType      NotificationType `json:"bodyType"`
	Category      PushCategory     `json:"category,omitempty"`
	Deeplink      string           `json:"deepLink,omitempty"`
	Image         string           `json:"imageUrl,omitempty"`
	IsScheduled   bool             `json:"isScheduled,omitempty"`
	ScheduledTime string           `json:"scheduleTime,omitempty"`
}

// TransactionEvent - structure used to pass messages from wallet to bus
type TransactionEvent struct {
	Type                      string                      `json:"type"`
	BlockNumber               *big.Int                    `json:"block-number"`
	Accounts                  []common.Address            `json:"accounts"`
	NewTransactionsPerAccount map[common.Address]int      `json:"new-transactions"`
	ERC20                     bool                        `json:"erc20"`
	MaxKnownBlocks            map[common.Address]*big.Int `json:"max-known-blocks"`
}

// MessageEvent - structure used to pass messages from chat to bus
type MessageEvent struct{}

// CustomEvent - structure used to pass custom user set messages to bus
type CustomEvent struct{}

type transmitter struct {
	publisher *event.Feed

	wg   sync.WaitGroup
	quit chan struct{}
}

// Service keeps the state of message bus
type Service struct {
	started           bool
	transmitter       *transmitter
	walletTransmitter *transmitter
	db                *Database
	walletDB          *wallet.Database
	accountsDB        *accounts.Database
}

func NewService(appDB *sql.DB, network uint64) *Service {
	db := NewDB(appDB, network)
	walletDB := wallet.NewDB(appDB, network)
	accountsDB := accounts.NewDB(appDB)
	trans := &transmitter{}
	walletTrans := &transmitter{}

	return &Service{
		db:                db,
		walletDB:          walletDB,
		accountsDB:        accountsDB,
		transmitter:       trans,
		walletTransmitter: walletTrans,
	}
}

func (n *Notification) MarshalJSON() ([]byte, error) {
	var body json.RawMessage
	var err error

	switch n.BodyType {
	case TypeTransaction:
		if nb, ok := n.Body.(*notificationBody); ok {
			body, err = json.Marshal(nb)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf(marshalTypeMismatchErr, n.BodyType)
		}

	case TypeMessage:
		if nmb, ok := n.Body.(*protocol.MessageNotificationBody); ok {
			body, err = json.Marshal(nmb)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf(marshalTypeMismatchErr, n.BodyType)
		}

	default:
		return nil, fmt.Errorf("unknown NotificationType '%s'", n.BodyType)
	}

	alias := notificationAlias{
		n.ID,
		n.Platform,
		body,
		n.BodyType,
		n.Category,
		n.Deeplink,
		n.Image,
		n.IsScheduled,
		n.ScheduledTime,
	}

	return json.Marshal(alias)
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	var alias notificationAlias
	err := json.Unmarshal(data, &alias)
	if err != nil {
		return err
	}

	n.BodyType = alias.BodyType
	n.Category = alias.Category
	n.Platform = alias.Platform
	n.ID = alias.ID
	n.Image = alias.Image
	n.Deeplink = alias.Deeplink
	n.IsScheduled = alias.IsScheduled
	n.ScheduledTime = alias.ScheduledTime

	switch n.BodyType {
	case TypeTransaction:
		return n.unmarshalAndAttachBody(alias.Body, &notificationBody{})

	case TypeMessage:
		return n.unmarshalAndAttachBody(alias.Body, &protocol.MessageNotificationBody{})

	default:
		return fmt.Errorf("unknown NotificationType '%s'", n.BodyType)
	}
}

func (n *Notification) unmarshalAndAttachBody(body json.RawMessage, bodyStruct interface{}) error {
	err := json.Unmarshal(body, &bodyStruct)
	if err != nil {
		return err
	}

	n.Body = bodyStruct
	return nil
}

func pushMessages(ns []*Notification) {
	for _, n := range ns {
		pushMessage(n)
	}
}

func pushMessage(notification *Notification) {
	log.Info("Pushing a new push notification", "info", notification)
	signal.SendLocalNotifications(notification)
}

func (s *Service) buildTransactionNotification(rawTransfer wallet.Transfer) *Notification {
	log.Info("Handled a new transfer in buildTransactionNotification", "info", rawTransfer)

	var deeplink string
	var state transactionState
	transfer := wallet.CastToTransferView(rawTransfer)

	switch {
	case transfer.TxStatus == hexutil.Uint64(0):
		state = failed
	case transfer.Address == transfer.To:
		state = inbound
	default:
		state = outbound
	}

	from, err := s.accountsDB.GetAccountByAddress(types.Address(transfer.From))

	if err != nil {
		log.Debug("Could not select From account by address", "error", err)
	}

	to, err := s.accountsDB.GetAccountByAddress(types.Address(transfer.To))

	if err != nil {
		log.Debug("Could not select To account by address", "error", err)
	}

	if from != nil {
		deeplink = walletDeeplinkPrefix + from.Address.String()
	} else if to != nil {
		deeplink = walletDeeplinkPrefix + to.Address.String()
	}

	body := notificationBody{
		State:       state,
		From:        transfer.From,
		To:          transfer.Address,
		FromAccount: from,
		ToAccount:   to,
		Value:       transfer.Value,
		ERC20:       string(transfer.Type) == "erc20",
		Contract:    transfer.Contract,
		Network:     transfer.NetworkID,
	}

	return &Notification{
		BodyType: TypeTransaction,
		ID:       transfer.ID,
		Body:     &body,
		Deeplink: deeplink,
		Category: CategoryTransaction,
	}
}

func (s *Service) transactionsHandler(payload TransactionEvent) {
	log.Info("Handled a new transaction", "info", payload)

	limit := 20
	if payload.BlockNumber != nil {
		for _, address := range payload.Accounts {
			if payload.BlockNumber.Cmp(payload.MaxKnownBlocks[address]) >= 0 {
				log.Info("Handled transfer for address", "info", address)
				transfers, err := s.walletDB.GetTransfersByAddressAndBlock(address, payload.BlockNumber, int64(limit))
				if err != nil {
					log.Error("Could not fetch transfers", "error", err)
				}

				for _, transaction := range transfers {
					n := s.buildTransactionNotification(transaction)
					pushMessage(n)
				}
			}
		}
	}
}

// SubscribeWallet - Subscribes to wallet signals
func (s *Service) SubscribeWallet(publisher *event.Feed) error {
	s.walletTransmitter.publisher = publisher

	preference, err := s.db.GetWalletPreference()

	if err != nil {
		log.Error("Failed to get wallet preference", "error", err)
		return nil
	}

	if preference.Enabled {
		s.StartWalletWatcher()
	}

	return nil
}

// StartWalletWatcher - Forward wallet events to notifications
func (s *Service) StartWalletWatcher() {
	if s.walletTransmitter.quit != nil {
		// already running, nothing to do
		return
	}

	if s.walletTransmitter.publisher == nil {
		log.Error("wallet publisher was not initialized")
		return
	}

	s.walletTransmitter.quit = make(chan struct{})
	events := make(chan wallet.Event, 10)
	sub := s.walletTransmitter.publisher.Subscribe(events)

	s.walletTransmitter.wg.Add(1)

	maxKnownBlocks := map[common.Address]*big.Int{}
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
				if event.Type == wallet.EventNewBlock && len(maxKnownBlocks) > 0 {
					newBlocks := false
					for _, address := range event.Accounts {
						if _, ok := maxKnownBlocks[address]; !ok {
							newBlocks = true
							maxKnownBlocks[address] = event.BlockNumber
						} else if event.BlockNumber.Cmp(maxKnownBlocks[address]) == 1 {
							maxKnownBlocks[address] = event.BlockNumber
							newBlocks = true
						}
					}
					if newBlocks {
						s.transmitter.publisher.Send(TransactionEvent{
							Type:                      string(event.Type),
							BlockNumber:               event.BlockNumber,
							Accounts:                  event.Accounts,
							NewTransactionsPerAccount: event.NewTransactionsPerAccount,
							ERC20:                     event.ERC20,
							MaxKnownBlocks:            maxKnownBlocks,
						})
					}
				} else if event.Type == wallet.EventMaxKnownBlock {
					for _, address := range event.Accounts {
						if _, ok := maxKnownBlocks[address]; !ok {
							maxKnownBlocks[address] = event.BlockNumber
						}
					}
				}
			}
		}
	}()
}

// StopWalletWatcher - stops watching for new wallet events
func (s *Service) StopWalletWatcher() {
	if s.walletTransmitter.quit != nil {
		close(s.walletTransmitter.quit)
		s.walletTransmitter.wg.Wait()
		s.walletTransmitter.quit = nil
	}
}

// IsWatchingWallet - check if local-notifications are subscribed to wallet updates
func (s *Service) IsWatchingWallet() bool {
	return s.walletTransmitter.quit != nil
}

// Start Worker which processes all incoming messages
func (s *Service) Start(_ *p2p.Server) error {
	s.started = true

	s.transmitter.quit = make(chan struct{})
	s.transmitter.publisher = &event.Feed{}

	events := make(chan TransactionEvent, 10)
	sub := s.transmitter.publisher.Subscribe(events)

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
			Namespace: "localnotifications",
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

func SendMessageNotifications(mnb []protocol.MessageNotificationBody) {
	var ns []*Notification
	for _, n := range mnb {
		ns = append(ns, &Notification{
			Body:     n,
			BodyType: TypeMessage,
			Category: CategoryMessage,
			Deeplink: "", // TODO find what if any Deeplink should be used here
			Image:    "", // TODO do we want to attach any image data contained on the MessageBody{}?
		})
	}

	// sends notifications messages to the OS level application
	pushMessages(ns)
}
