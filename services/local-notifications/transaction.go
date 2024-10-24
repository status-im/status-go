package localnotifications

import (
	"encoding/json"
	"math/big"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type transactionState string

const (
	walletDeeplinkPrefix = "status-app://wallet/"

	failed   transactionState = "failed"
	inbound  transactionState = "inbound"
	outbound transactionState = "outbound"
)

// TransactionEvent - structure used to pass messages from wallet to bus
type TransactionEvent struct {
	Type           string                      `json:"type"`
	BlockNumber    *big.Int                    `json:"block-number"`
	Accounts       []common.Address            `json:"accounts"`
	MaxKnownBlocks map[common.Address]*big.Int `json:"max-known-blocks"`
}

type transactionBody struct {
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

func (t transactionBody) MarshalJSON() ([]byte, error) {
	type Alias transactionBody
	item := struct{ *Alias }{Alias: (*Alias)(&t)}
	return json.Marshal(item)
}

func (s *Service) buildTransactionNotification(rawTransfer transfer.Transfer) *Notification {
	logutils.ZapLogger().Debug("Handled a new transfer in buildTransactionNotification", zap.Any("info", rawTransfer))

	var deeplink string
	var state transactionState
	transfer := transfer.CastToTransferView(rawTransfer)

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
		logutils.ZapLogger().Debug("Could not select From account by address", zap.Error(err))
	}

	to, err := s.accountsDB.GetAccountByAddress(types.Address(transfer.To))

	if err != nil {
		logutils.ZapLogger().Debug("Could not select To account by address", zap.Error(err))
	}

	if from != nil {
		deeplink = walletDeeplinkPrefix + from.Address.String()
	} else if to != nil {
		deeplink = walletDeeplinkPrefix + to.Address.String()
	}

	body := transactionBody{
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
		Body:     body,
		Deeplink: deeplink,
		Category: CategoryTransaction,
	}
}

func (s *Service) transactionsHandler(payload TransactionEvent) {
	logutils.ZapLogger().Info("Handled a new transaction", zap.Any("info", payload))

	limit := 20
	if payload.BlockNumber != nil {
		for _, address := range payload.Accounts {
			if payload.BlockNumber.Cmp(payload.MaxKnownBlocks[address]) >= 0 {
				logutils.ZapLogger().Info("Handled transfer for address", zap.Stringer("info", address))
				transfers, err := s.walletDB.GetTransfersByAddressAndBlock(s.chainID, address, payload.BlockNumber, int64(limit))
				if err != nil {
					logutils.ZapLogger().Error("Could not fetch transfers", zap.Error(err))
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
		logutils.ZapLogger().Error("Failed to get wallet preference", zap.Error(err))
		s.WatchingEnabled = false
	} else {
		s.WatchingEnabled = preference.Enabled
	}

	s.StartWalletWatcher()

	return err
}

// StartWalletWatcher - Forward wallet events to notifications
func (s *Service) StartWalletWatcher() {
	if s.walletTransmitter.quit != nil {
		// already running, nothing to do
		return
	}

	if s.walletTransmitter.publisher == nil {
		logutils.ZapLogger().Error("wallet publisher was not initialized")
		return
	}

	s.walletTransmitter.quit = make(chan struct{})
	events := make(chan walletevent.Event, 10)
	sub := s.walletTransmitter.publisher.Subscribe(events)

	s.walletTransmitter.wg.Add(1)

	maxKnownBlocks := map[common.Address]*big.Int{}
	go func() {
		defer gocommon.LogOnPanic()
		defer s.walletTransmitter.wg.Done()
		historyReady := false
		for {
			select {
			case <-s.walletTransmitter.quit:
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				// technically event.Feed cannot send an error to subscription.Err channel.
				// the only time we will get an event is when that channel is closed.
				if err != nil {
					logutils.ZapLogger().Error("wallet signals transmitter failed with", zap.Error(err))
				}
				return
			case event := <-events:
				if event.Type == transfer.EventNewTransfers && historyReady && event.BlockNumber != nil {
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
					if newBlocks && s.WatchingEnabled {
						s.transmitter.publisher.Send(TransactionEvent{
							Type:           string(event.Type),
							BlockNumber:    event.BlockNumber,
							Accounts:       event.Accounts,
							MaxKnownBlocks: maxKnownBlocks,
						})
					}
				} else if event.Type == transfer.EventRecentHistoryReady {
					historyReady = true
					if event.BlockNumber != nil {
						for _, address := range event.Accounts {
							if _, ok := maxKnownBlocks[address]; !ok {
								maxKnownBlocks[address] = event.BlockNumber
							}
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
