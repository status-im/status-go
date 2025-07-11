package localbackup

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/event"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiaccountscommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/wakusync"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// TODO this is duplicated
var (
	ErrNotWatchOnlyAccount           = errors.New("an account is not a watch only account")
	ErrTryingToStoreOldWalletAccount = errors.New("trying to store an old wallet account")
)

const (
	EventWatchOnlyAccountRetrieved walletevent.EventType = "wallet-watch-only-account-retrieved"
)

// Service is a wallet local backup service.
type Service struct {
	accountsDB *accounts.Database
	feed       *event.Feed
}

func NewService(accountsDB *accounts.Database, feed *event.Feed) *Service {
	return &Service{
		accountsDB: accountsDB,
		feed:       feed,
	}
}

func (s *Service) prepareSyncAccountMessage(acc *accounts.Account) *protobuf.SyncAccount {
	return &protobuf.SyncAccount{
		Clock:                 acc.Clock,
		Address:               acc.Address.Bytes(),
		KeyUid:                acc.KeyUID,
		PublicKey:             acc.PublicKey,
		Path:                  acc.Path,
		Name:                  acc.Name,
		ColorId:               string(acc.ColorID),
		Emoji:                 acc.Emoji,
		Wallet:                acc.Wallet,
		Chat:                  acc.Chat,
		Hidden:                acc.Hidden,
		Removed:               acc.Removed,
		Operable:              acc.Operable.String(),
		Position:              acc.Position,
		ProdPreferredChainIDs: acc.ProdPreferredChainIDs,
		TestPreferredChainIDs: acc.TestPreferredChainIDs,
	}
}

func (s *Service) backupWatchOnlyAccounts() ([]*protobuf.Backup, error) {
	accounts, err := s.accountsDB.GetAllWatchOnlyAccounts()
	if err != nil {
		return nil, err
	}

	var backupMessages []*protobuf.Backup
	for _, acc := range accounts {

		backupMessage := &protobuf.Backup{}
		backupMessage.WatchOnlyAccount = s.prepareSyncAccountMessage(acc)

		backupMessages = append(backupMessages, backupMessage)
	}

	return backupMessages, nil
}

func (s *Service) ExportBackup() ([]byte, error) {
	backup := &protobuf.WalletLocalBackup{}

	woAccountsToBackup, err := s.backupWatchOnlyAccounts()
	if err != nil {
		return nil, err
	}
	for _, d := range woAccountsToBackup {
		backup.WatchOnlyAccounts = append(backup.WatchOnlyAccounts, d.WatchOnlyAccount)
	}

	return proto.Marshal(backup)
}

func mapSyncAccountToAccount(message *protobuf.SyncAccount, accountOperability accounts.AccountOperable, accType accounts.AccountType) *accounts.Account {
	return &accounts.Account{
		Address:               types.BytesToAddress(message.Address),
		KeyUID:                message.KeyUid,
		PublicKey:             types.HexBytes(message.PublicKey),
		Type:                  accType,
		Path:                  message.Path,
		Name:                  message.Name,
		ColorID:               multiaccountscommon.CustomizationColor(message.ColorId),
		Emoji:                 message.Emoji,
		Wallet:                message.Wallet,
		Chat:                  message.Chat,
		Hidden:                message.Hidden,
		Clock:                 message.Clock,
		Operable:              accountOperability,
		Removed:               message.Removed,
		Position:              message.Position,
		ProdPreferredChainIDs: message.ProdPreferredChainIDs,
		TestPreferredChainIDs: message.TestPreferredChainIDs,
	}
}

// TODO this is a duplicate of the code in messenger_handler. Should it be moved to a common place?
func (s *Service) handleSyncWatchOnlyAccount(message *protobuf.SyncAccount) (*accounts.Account, error) {
	if message.KeyUid != "" {
		return nil, ErrNotWatchOnlyAccount
	}

	accountOperability := accounts.AccountFullyOperable

	accAddress := types.BytesToAddress(message.Address)
	dbAccount, err := s.accountsDB.GetAccountByAddress(accAddress)
	if err != nil && err != accounts.ErrDbAccountNotFound {
		return nil, err
	}

	if dbAccount != nil {
		if message.Clock <= dbAccount.Clock {
			return nil, ErrTryingToStoreOldWalletAccount
		}

		if message.Removed {
			err = s.accountsDB.RemoveAccount(accAddress, message.Clock)
			if err != nil {
				return nil, err
			}
			dbAccount.Removed = true
			return dbAccount, nil
		}
	}

	acc := mapSyncAccountToAccount(message, accountOperability, accounts.AccountTypeWatch)

	err = s.accountsDB.SaveOrUpdateAccounts([]*accounts.Account{acc}, false)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (s *Service) handleWatchOnlyAccount(message *protobuf.SyncAccount) error {
	if message == nil {
		return nil
	}

	acc, err := s.handleSyncWatchOnlyAccount(message)
	if err != nil {
		return err
	}
	response := wakusync.WakuBackedUpDataResponse{
		WatchOnlyAccount: acc,
	}
	encodedmessage, err := json.Marshal(response)
	if err != nil {
		return err
	}
	event := walletevent.Event{
		Type:    EventWatchOnlyAccountRetrieved,
		Message: string(encodedmessage),
	}
	s.feed.Send(event)

	return nil
}

func (s *Service) ImportBackup(data []byte) error {
	var backup protobuf.WalletLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}
	var errs []error

	for _, watchOnlyAccount := range backup.WatchOnlyAccounts {
		err = s.handleWatchOnlyAccount(watchOnlyAccount)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
