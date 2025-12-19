package syncing

import (
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	multiaccountscommon "github.com/status-im/status-go/internal/db/multiaccounts/common"
	maErrors "github.com/status-im/status-go/internal/db/multiaccounts/errors"
	settings2 "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/accounts/accountsevent"
)

// TODO move this code out of the protocol package
// https://github.com/status-im/status-go/pull/6967#discussion_r2391383496

var (
	ErrNotWatchOnlyAccount           = errors.New("an account is not a watch only account")
	ErrTryingToStoreOldWalletAccount = errors.New("trying to store an old wallet account")
)

func MapSyncAccountToAccount(message *protobuf.SyncAccount, accountOperability accsmanagementtypes.AccountOperable,
	accType accsmanagementtypes.AccountType) *accsmanagementtypes.Account {
	return &accsmanagementtypes.Account{
		Address:               types2.BytesToAddress(message.Address),
		KeyUID:                message.KeyUid,
		PublicKey:             types2.HexBytes(message.PublicKey),
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

	accAddress := types2.BytesToAddress(message.Address)
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

// extractSyncSetting parses incoming *protobuf.SyncSetting and stores the setting data if needed
func ExtractAndSaveSyncSetting(accountsDB *accounts.Database, logger *zap.Logger, syncSetting *protobuf.SyncSetting) (*settings2.SyncSettingField, error) {
	sf, err := settings2.GetFieldFromProtobufType(syncSetting.Type)
	if err != nil {
		logger.Error(
			"extractSyncSetting - settings.GetFieldFromProtobufType",
			zap.Error(err),
			zap.Any("syncSetting", syncSetting),
		)
		return nil, err
	}

	spf := sf.SyncProtobufFactory()
	if spf == nil {
		logger.Warn("extractSyncSetting - received protobuf for setting with no SyncProtobufFactory")
		return nil, nil
	}
	if spf.Inactive() {
		logger.Warn("extractSyncSetting - received protobuf for inactive sync setting")
		return nil, nil
	}

	value := spf.ExtractValueFromProtobuf()(syncSetting)

	err = accountsDB.SaveSyncSetting(sf, value, syncSetting.Clock)
	if err == maErrors.ErrNewClockOlderThanCurrent {
		logger.Info("extractSyncSetting - SaveSyncSetting :", zap.Error(err))
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if v, ok := value.([]byte); ok {
		value = json.RawMessage(v)
	}

	return &settings2.SyncSettingField{SettingField: sf, Value: value}, nil
}
