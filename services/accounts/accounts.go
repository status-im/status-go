package accounts

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	accscommon "github.com/status-im/status-go/accounts-management/common"
	accsmanagementtypes "github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	walletsettings "github.com/status-im/status-go/multiaccounts/settings_wallet"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/accounts/accountsevent"
)

func NewAccountsAPI(manager *accsmanagement.AccountsManager, config *params.NodeConfig, db *accounts.Database, messenger **protocol.Messenger, publisher *pubsub.Publisher) *API {
	return &API{manager, config, db, messenger, publisher}
}

// API is class with methods available over RPC.
type API struct {
	manager   *accsmanagement.AccountsManager
	config    *params.NodeConfig
	db        *accounts.Database
	messenger **protocol.Messenger
	publisher *pubsub.Publisher
}

type DerivedAddress struct {
	Address        common.Address `json:"address"`
	Path           string         `json:"path"`
	HasActivity    bool           `json:"hasActivity"`
	AlreadyCreated bool           `json:"alreadyCreated"`
}

func (api *API) publishAccountsEvent(accounts []*accounts.Account, removeEvent bool) {
	if api.publisher != nil {
		commonAddresses := []common.Address{}
		for _, acc := range accounts {
			commonAddresses = append(commonAddresses, common.Address(acc.Address))
		}

		if removeEvent {
			pubsub.Publish(api.publisher, accountsevent.AccountsRemovedEvent{
				Accounts: commonAddresses,
			})
		} else {
			pubsub.Publish(api.publisher, accountsevent.AccountsAddedEvent{
				Accounts: commonAddresses,
			})
		}
	}
}

func (api *API) AddKeypairViaSeedPhrase(ctx context.Context, mnemonic string, password string, name string,
	walletAccount *accsmanagementtypes.AccountCreationDetails) (*accounts.Keypair, error) {
	addedKeypair, err := (*api.messenger).AddKeypairViaSeedPhrase(mnemonic, password, name, walletAccount)
	if err != nil {
		return nil, err
	}

	api.publishAccountsEvent(addedKeypair.Accounts, false)

	return addedKeypair, nil
}

func (api *API) AddKeypairStoredToKeycard(ctx context.Context, keyUID string, masterAddress string, name string,
	walletAccounts []*accounts.Account) (*accounts.Keypair, error) {
	addedKeypair, err := (*api.messenger).AddKeypairStoredToKeycard(keyUID, masterAddress, name, walletAccounts)
	if err != nil {
		return nil, err
	}

	api.publishAccountsEvent(addedKeypair.Accounts, false)

	return addedKeypair, nil
}

func (api *API) AddKeypairViaPrivateKey(ctx context.Context, privateKey string, password string, name string,
	walletAccount *accsmanagementtypes.AccountCreationDetails) (*accounts.Keypair, error) {
	addedKeypair, err := (*api.messenger).AddKeypairViaPrivateKey(privateKey, password, name, walletAccount)
	if err != nil {
		return nil, err
	}

	api.publishAccountsEvent(addedKeypair.Accounts, false)

	return addedKeypair, nil
}

func (api *API) AddAccount(ctx context.Context, password string, account *accounts.Account) error {
	if account.Type == accounts.AccountTypeGenerated {
		account.AddressWasNotShown = true
	}

	err := (*api.messenger).AddAccount(account, password)
	if err != nil {
		return err
	}

	api.publishAccountsEvent([]*accounts.Account{account}, false)

	return nil
}

func (api *API) UpdateAccount(ctx context.Context, account *accounts.Account) error {
	logutils.ZapLogger().Info("[AccountsAPI::SaveAccount]")
	err := (*api.messenger).UpdateAccount(account)
	if err != nil {
		return err
	}

	api.publishAccountsEvent([]*accounts.Account{account}, false)

	return nil
}

// Setting `Keypair` without `Accounts` will update keypair only, `Keycards` won't be saved/updated this way.
func (api *API) UpdateKeypair(ctx context.Context, keypair *accounts.Keypair) error {
	logutils.ZapLogger().Info("[AccountsAPI::SaveKeypair]")
	err := (*api.messenger).UpdateKeypair(keypair)
	if err != nil {
		return err
	}

	api.publishAccountsEvent(keypair.Accounts, false)

	return nil
}

func (api *API) HasPairedDevices(ctx context.Context) bool {
	return (*api.messenger).HasPairedDevices()
}

// Setting `Keypair` without `Accounts` will update keypair only.
func (api *API) UpdateKeypairName(ctx context.Context, keyUID string, name string) error {
	return (*api.messenger).UpdateKeypairName(keyUID, name)
}

func (api *API) MoveWalletAccount(ctx context.Context, fromPosition int64, toPosition int64) error {
	return (*api.messenger).MoveWalletAccount(fromPosition, toPosition)
}

func (api *API) UpdateTokenPreferences(ctx context.Context, preferences []walletsettings.TokenPreferences) error {
	return (*api.messenger).UpdateTokenPreferences(preferences)
}

func (api *API) GetTokenPreferences(ctx context.Context) ([]walletsettings.TokenPreferences, error) {
	return (*api.messenger).GetTokenPreferences()
}

func (api *API) UpdateCollectiblePreferences(ctx context.Context, preferences []walletsettings.CollectiblePreferences) error {
	return (*api.messenger).UpdateCollectiblePreferences(preferences)
}

func (api *API) GetCollectiblePreferences(ctx context.Context) ([]walletsettings.CollectiblePreferences, error) {
	return (*api.messenger).GetCollectiblePreferences()
}

func (api *API) GetAccounts(ctx context.Context) ([]*accounts.Account, error) {
	return api.db.GetActiveAccounts()
}

func (api *API) GetWatchOnlyAccounts(ctx context.Context) ([]*accounts.Account, error) {
	return api.db.GetActiveWatchOnlyAccounts()
}

func (api *API) GetKeypairs(ctx context.Context) ([]*accounts.Keypair, error) {
	return api.db.GetActiveKeypairs()
}

func (api *API) GetAccountByAddress(ctx context.Context, address types.Address) (*accounts.Account, error) {
	return api.db.GetAccountByAddress(address)
}

func (api *API) GetKeypairByKeyUID(ctx context.Context, keyUID string) (*accounts.Keypair, error) {
	return api.db.GetKeypairByKeyUID(keyUID)
}

func (api *API) DeleteAccount(ctx context.Context, address types.Address) error {
	err := (*api.messenger).DeleteAccount(address)
	if err != nil {
		return err
	}

	api.publishAccountsEvent([]*accounts.Account{{Address: address}}, true)

	return nil
}

func (api *API) DeleteKeypair(ctx context.Context, keyUID string) error {
	keypair, err := api.db.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return err
	}

	if keypair.Type == accounts.KeypairTypeProfile {
		return accounts.ErrCannotRemoveProfileKeypair
	}

	err = (*api.messenger).DeleteKeypair(keyUID)
	if err != nil {
		return err
	}

	api.publishAccountsEvent(keypair.Accounts, true)

	return nil
}

// RemainingAccountCapacity returns the number of accounts that can be added.
func (api *API) RemainingAccountCapacity(ctx context.Context) (int, error) {
	return (*api.messenger).RemainingAccountCapacity()
}

// RemainingKeypairCapacity returns the number of keypairs that can be added.
func (api *API) RemainingKeypairCapacity(ctx context.Context) (int, error) {
	return (*api.messenger).RemainingKeypairCapacity()
}

// RemainingWatchOnlyAccountCapacity returns the number of watch-only accounts that can be added.
func (api *API) RemainingWatchOnlyAccountCapacity(ctx context.Context) (int, error) {
	return (*api.messenger).RemainingWatchOnlyAccountCapacity()
}

// Creates all keystore files for a keypair and mark it in db as fully operable.
func (api *API) MakePrivateKeyKeypairFullyOperable(ctx context.Context, privateKey string, password string) error {
	return (*api.messenger).MakePrivateKeyKeypairFullyOperable(privateKey, password)
}

func (api *API) MakePartiallyOperableAccoutsFullyOperable(ctx context.Context, password string) (addresses []types.Address, err error) {
	return api.manager.MakePartiallyOperableAccoutsFullyOperable(password)
}

// Creates all keystore files for a keypair and mark it in db as fully operable.
func (api *API) MakeSeedPhraseKeypairFullyOperable(ctx context.Context, mnemonic string, password string) error {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	return (*api.messenger).MakeSeedPhraseKeypairFullyOperable(mnemonicNoExtraSpaces, password)
}

// Creates a random new mnemonic.
func (api *API) GetRandomMnemonic(ctx context.Context) (string, error) {
	return accscommon.CreateRandomMnemonicWithDefaultLength()
}

func (api *API) VerifyKeystoreFileForAccount(address types.Address, password string) bool {
	_, err := api.manager.LoadAccount(address, password)
	return err == nil
}

func (api *API) VerifyPassword(password string) bool {
	address, err := api.db.GetChatAddress()
	if err != nil {
		return false
	}
	return api.VerifyKeystoreFileForAccount(address, password)
}

func (api *API) MigrateNonProfileKeycardKeypairToApp(ctx context.Context, mnemonic string, password string) error {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	return (*api.messenger).MigrateNonProfileKeycardKeypairToApp(ctx, mnemonicNoExtraSpaces, password)
}

// If keypair is migrated from keycard to app, then `accountsComingFromKeycard` should be set to true, otherwise false.
// If keycard is new `Position` will be determined and set by the backend and `KeycardLocked` will be set to false.
// If keycard is already added, `Position` and `KeycardLocked` will be unchanged.
func (api *API) SaveOrUpdateKeycard(ctx context.Context, keycard *accounts.Keycard, accountsComingFromKeycard bool) error {
	return (*api.messenger).SaveOrUpdateKeycard(ctx, keycard, !accountsComingFromKeycard)
}

func (api *API) GetAllKnownKeycards(ctx context.Context) ([]*accounts.Keycard, error) {
	return api.db.GetAllKnownKeycards()
}

func (api *API) GetKeycardsWithSameKeyUID(ctx context.Context, keyUID string) ([]*accounts.Keycard, error) {
	return api.db.GetKeycardsWithSameKeyUID(keyUID)
}

func (api *API) GetKeycardByKeycardUID(ctx context.Context, keycardUID string) (*accounts.Keycard, error) {
	return api.db.GetKeycardByKeycardUID(keycardUID)
}

func (api *API) SetKeycardName(ctx context.Context, keycardUID string, kpName string) error {
	return (*api.messenger).SetKeycardName(ctx, keycardUID, kpName)
}

func (api *API) KeycardLocked(ctx context.Context, keycardUID string) error {
	return (*api.messenger).KeycardLocked(ctx, keycardUID)
}

func (api *API) KeycardUnlocked(ctx context.Context, keycardUID string) error {
	return (*api.messenger).KeycardUnlocked(ctx, keycardUID)
}

func (api *API) DeleteKeycardAccounts(ctx context.Context, keycardUID string, accountAddresses []types.Address) error {
	return (*api.messenger).DeleteKeycardAccounts(ctx, keycardUID, accountAddresses)
}

func (api *API) DeleteKeycard(ctx context.Context, keycardUID string) error {
	return (*api.messenger).DeleteKeycard(ctx, keycardUID)
}

func (api *API) DeleteAllKeycardsWithKeyUID(ctx context.Context, keyUID string) error {
	return (*api.messenger).DeleteAllKeycardsWithKeyUID(ctx, keyUID)
}

func (api *API) UpdateKeycardUID(ctx context.Context, oldKeycardUID string, newKeycardUID string) error {
	return (*api.messenger).UpdateKeycardUID(ctx, oldKeycardUID, newKeycardUID)
}

func (api *API) AddressWasShown(address types.Address) error {
	return api.db.AddressWasShown(address)
}

func (api *API) GetNumOfAddressesToGenerateForKeypair(keyUID string) (uint64, error) {
	return api.db.GetNumOfAddressesToGenerateForKeypair(keyUID)
}

func (api *API) ResolveSuggestedPathForKeypair(keyUID string) (string, error) {
	return api.db.ResolveSuggestedPathForKeypair(keyUID)
}
