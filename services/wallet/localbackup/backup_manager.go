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

// BackupManager is a wallet local backup service.
type BackupManager struct {
	accountsDB *accounts.Database
	feed       *event.Feed
}

func NewBackupManager(accountsDB *accounts.Database, feed *event.Feed) *BackupManager {
	return &BackupManager{
		accountsDB: accountsDB,
		feed:       feed,
	}
}

func (b *BackupManager) prepareSyncAccountMessage(acc *accounts.Account) *protobuf.SyncAccount {
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

func (b *BackupManager) backupWatchOnlyAccounts() ([]*protobuf.Backup, error) {
	accounts, err := b.accountsDB.GetAllWatchOnlyAccounts()
	if err != nil {
		return nil, err
	}

	var backupMessages []*protobuf.Backup
	for _, acc := range accounts {

		backupMessage := &protobuf.Backup{}
		backupMessage.WatchOnlyAccount = b.prepareSyncAccountMessage(acc)

		backupMessages = append(backupMessages, backupMessage)
	}

	return backupMessages, nil
}

func (b *BackupManager) ExportBackup() ([]byte, error) {
	backup := &protobuf.WalletLocalBackup{}

	woAccountsToBackup, err := b.backupWatchOnlyAccounts()
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
func (b *BackupManager) handleSyncWatchOnlyAccount(message *protobuf.SyncAccount) (*accounts.Account, error) {
	if message.KeyUid != "" {
		return nil, ErrNotWatchOnlyAccount
	}

	accountOperability := accounts.AccountFullyOperable

	accAddress := types.BytesToAddress(message.Address)
	dbAccount, err := b.accountsDB.GetAccountByAddress(accAddress)
	if err != nil && err != accounts.ErrDbAccountNotFound {
		return nil, err
	}

	if dbAccount != nil {
		if message.Clock <= dbAccount.Clock {
			return nil, ErrTryingToStoreOldWalletAccount
		}

		if message.Removed {
			err = b.accountsDB.RemoveAccount(accAddress, message.Clock)
			if err != nil {
				return nil, err
			}
			dbAccount.Removed = true
			return dbAccount, nil
		}
	}

	acc := mapSyncAccountToAccount(message, accountOperability, accounts.AccountTypeWatch)

	err = b.accountsDB.SaveOrUpdateAccounts([]*accounts.Account{acc}, false)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (b *BackupManager) handleWatchOnlyAccount(message *protobuf.SyncAccount) error {
	if message == nil {
		return nil
	}

	acc, err := b.handleSyncWatchOnlyAccount(message)
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
	b.feed.Send(event)

	return nil
}

func (b *BackupManager) ImportBackup(data []byte) error {
	var backup protobuf.WalletLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}
	var errs []error

	for _, watchOnlyAccount := range backup.WatchOnlyAccounts {
		err = b.handleWatchOnlyAccount(watchOnlyAccount)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
