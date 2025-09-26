package syncing

import (
	"errors"

	gethcommon "github.com/ethereum/go-ethereum/common"

	accsmanagementtypes "github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiaccountscommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/accounts/accountsevent"
)

var (
	ErrNotWatchOnlyAccount           = errors.New("an account is not a watch only account")
	ErrTryingToStoreOldWalletAccount = errors.New("trying to store an old wallet account")
)

func MapSyncAccountToAccount(message *protobuf.SyncAccount, accountOperability accsmanagementtypes.AccountOperable,
	accType accsmanagementtypes.AccountType) *accsmanagementtypes.Account {
	return &accsmanagementtypes.Account{
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

func HandleSyncWatchOnlyAccount(accountsDB *accounts.Database, message *protobuf.SyncAccount, accountsPublisher *pubsub.Publisher) (*accsmanagementtypes.Account, error) {
	if message.KeyUid != "" {
		return nil, ErrNotWatchOnlyAccount
	}

	accountOperability := accsmanagementtypes.AccountFullyOperable

	accAddress := types.BytesToAddress(message.Address)
	dbAccount, err := accountsDB.GetAccountByAddress(accAddress)
	if err != nil && err != accounts.ErrDbAccountNotFound {
		return nil, err
	}

	if dbAccount != nil {
		if message.Clock <= dbAccount.Clock {
			return nil, ErrTryingToStoreOldWalletAccount
		}

		if message.Removed {
			err = accountsDB.RemoveAccount(accAddress, message.Clock)
			if err != nil {
				return nil, err
			}

			err = accountsDB.ResolveAccountsPositions(message.Clock)
			if err != nil {
				return nil, err
			}
			dbAccount.Removed = true
			return dbAccount, nil
		}
	}

	acc := MapSyncAccountToAccount(message, accountOperability, accsmanagementtypes.AccountTypeWatch)

	err = accountsDB.SaveOrUpdateAccounts([]*accsmanagementtypes.Account{acc}, false)
	if err != nil {
		return nil, err
	}

	if accountsPublisher != nil {
		payload := []gethcommon.Address{gethcommon.Address(acc.Address)}
		if acc.Removed {
			pubsub.Publish(accountsPublisher, accountsevent.AccountsRemovedEvent{
				Accounts: payload,
			})
		} else {
			pubsub.Publish(accountsPublisher, accountsevent.AccountsAddedEvent{
				Accounts: payload,
			})
		}
	}
	return acc, nil
}
