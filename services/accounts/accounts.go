package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/keypairs"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
)

func NewAccountsAPI(manager *account.GethManager, config *params.NodeConfig, db *accounts.Database, feed *event.Feed, messenger **protocol.Messenger) *API {
	return &API{manager, config, db, feed, messenger}
}

// API is class with methods available over RPC.
type API struct {
	manager   *account.GethManager
	config    *params.NodeConfig
	db        *accounts.Database
	feed      *event.Feed
	messenger **protocol.Messenger
}

type DerivedAddress struct {
	Address        common.Address `json:"address"`
	Path           string         `json:"path"`
	HasActivity    bool           `json:"hasActivity"`
	AlreadyCreated bool           `json:"alreadyCreated"`
}

func (api *API) SaveAccount(ctx context.Context, account *accounts.Account) error {
	log.Info("[AccountsAPI::SaveAccount]")
	err := (*api.messenger).SaveAccount(account)
	if err != nil {
		return err
	}
	api.feed.Send([]*accounts.Account{account})
	return nil
}

func (api *API) GetAccounts(ctx context.Context) ([]*accounts.Account, error) {
	accounts, err := api.db.GetAccounts()
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		account := accounts[i]
		if account.Wallet && account.DerivedFrom == "" {
			address, err := api.db.GetWalletRootAddress()
			if err != nil {
				return nil, err
			}
			account.DerivedFrom = address.Hex()
		}
	}

	return accounts, nil
}

func (api *API) DeleteAccount(ctx context.Context, address types.Address, password string) error {
	acc, err := api.db.GetAccountByAddress(address)
	if err != nil {
		return err
	}

	migratedKeyPairs, err := api.db.GetMigratedKeyPairByKeyUID(acc.KeyUID)
	if err != nil {
		return err
	}

	if acc.Type != accounts.AccountTypeWatch && len(migratedKeyPairs) == 0 {
		if len(password) == 0 {
			return errors.New("`password` must be provided for non keycard accounts")
		}

		err = api.manager.DeleteAccount(address, password)
		var e *account.ErrCannotLocateKeyFile
		if err != nil && !errors.As(err, &e) {
			return err
		}

		if acc.Type != accounts.AccountTypeKey {
			allAccountsOfKeypairWithKeyUID, err := api.db.GetAccountsByKeyUID(acc.KeyUID)
			if err != nil {
				return err
			}

			lastAcccountOfKeypairWithTheSameKey := len(allAccountsOfKeypairWithKeyUID) == 1
			if lastAcccountOfKeypairWithTheSameKey {
				err = api.manager.DeleteAccount(types.Address(common.HexToAddress(acc.DerivedFrom)), password)
				var e *account.ErrCannotLocateKeyFile
				if err != nil && !errors.As(err, &e) {
					return err
				}
			}
		}
	}

	return (*api.messenger).DeleteAccount(address)
}

func (api *API) AddAccount(ctx context.Context, password string, account *accounts.Account) error {
	if len(account.Address) == 0 {
		return errors.New("`Address` field must be set")
	}

	if account.Wallet || account.Chat {
		return errors.New("default wallet and chat account cannot be added this way")
	}

	if len(account.Name) == 0 {
		return errors.New("`Name` field must be set")
	}

	if len(account.Emoji) == 0 {
		return errors.New("`Emoji` field must be set")
	}

	if len(account.Color) == 0 {
		return errors.New("`Color` field must be set")
	}

	if account.Type != accounts.AccountTypeWatch {

		if len(account.KeyUID) == 0 {
			return errors.New("`KeyUID` field must be set")
		}

		if len(account.PublicKey) == 0 {
			return errors.New("`PublicKey` field must be set")
		}

		if len(account.KeypairName) == 0 {
			return errors.New("`KeypairName` field must be set")
		}

		if account.Type != accounts.AccountTypeKey {
			if len(account.DerivedFrom) == 0 {
				return errors.New("`DerivedFrom` field must be set")
			}

			if len(account.Path) == 0 {
				return errors.New("`Path` field must be set")
			}
		}
	}

	addressExists, err := api.db.AddressExists(account.Address)
	if err != nil {
		return err
	}

	if addressExists {
		return errors.New("account already exists")
	}

	// we need to create local keystore file only if password is provided and the account is being added is of
	// "generated" or "seed" type.
	if (account.Type == accounts.AccountTypeGenerated || account.Type == accounts.AccountTypeSeed) && len(password) > 0 {
		info, err := api.manager.AccountsGenerator().LoadAccount(account.DerivedFrom, password)
		if err != nil {
			return err
		}

		_, err = api.manager.AccountsGenerator().StoreDerivedAccounts(info.ID, password, []string{account.Path})
		if err != nil {
			return err
		}
	}

	return api.SaveAccount(ctx, account)
}

// Imports a new private key and creates local keystore file.
func (api *API) ImportPrivateKey(ctx context.Context, privateKey string, password string) error {
	info, err := api.manager.AccountsGenerator().ImportPrivateKey(privateKey)
	if err != nil {
		return err
	}

	accs, err := api.db.GetAccountsByKeyUID(info.KeyUID)
	if err != nil {
		return err
	}

	if len(accs) > 0 {
		return errors.New("provided private key was already imported")
	}

	_, err = api.manager.AccountsGenerator().StoreAccount(info.ID, password)
	return err
}

// Imports a new mnemonic and creates local keystore file.
func (api *API) ImportMnemonic(ctx context.Context, mnemonic string, password string) error {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	generatedAccountInfo, err := api.manager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return err
	}

	accs, err := api.db.GetAccountsByKeyUID(generatedAccountInfo.KeyUID)
	if err != nil {
		return err
	}

	if len(accs) > 0 {
		return errors.New("provided mnemonic was already imported, to add new account use `AddAccount` endpoint")
	}

	_, err = api.manager.AccountsGenerator().StoreAccount(generatedAccountInfo.ID, password)
	return err
}

// Creates a random new mnemonic.
func (api *API) GetRandomMnemonic(ctx context.Context) (string, error) {
	return api.manager.GetRandomMnemonic()
}

func (api *API) VerifyKeystoreFileForAccount(address types.Address, password string) bool {
	_, err := api.manager.VerifyAccountPassword(api.config.KeyStoreDir, address.Hex(), password)
	return err == nil
}

func (api *API) VerifyPassword(password string) bool {
	address, err := api.db.GetChatAddress()
	if err != nil {
		return false
	}
	return api.VerifyKeystoreFileForAccount(address, password)
}

func (api *API) AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(ctx context.Context, kcUID string, kpName string, keyUID string, accountAddresses []string, password string) error {
	kp := keypairs.KeyPair{
		KeycardUID:      kcUID,
		KeycardName:     kpName,
		KeycardLocked:   false,
		KeyUID:          keyUID,
		LastUpdateClock: uint64(time.Now().Unix()),
	}
	for _, addr := range accountAddresses {
		kp.AccountsAddresses = append(kp.AccountsAddresses, types.Address(common.HexToAddress(addr)))
	}

	added, err := (*api.messenger).AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(ctx, &kp)
	if err != nil {
		return err
	}

	// Once we migrate a keypair, corresponding keystore files need to be deleted.
	if added && len(password) > 0 {
		for _, addr := range kp.AccountsAddresses {
			err = api.manager.DeleteAccount(addr, password)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) RemoveMigratedAccountsForKeycard(ctx context.Context, kcUID string, accountAddresses []string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).RemoveMigratedAccountsForKeycard(ctx, kcUID, accountAddresses, clock)
}

func (api *API) GetAllKnownKeycards(ctx context.Context) ([]*keypairs.KeyPair, error) {
	return api.db.GetAllKnownKeycards()
}

func (api *API) GetAllMigratedKeyPairs(ctx context.Context) ([]*keypairs.KeyPair, error) {
	return api.db.GetAllMigratedKeyPairs()
}

func (api *API) GetMigratedKeyPairByKeyUID(ctx context.Context, keyUID string) ([]*keypairs.KeyPair, error) {
	return api.db.GetMigratedKeyPairByKeyUID(keyUID)
}

func (api *API) SetKeycardName(ctx context.Context, kcUID string, kpName string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).SetKeycardName(ctx, kcUID, kpName, clock)
}

func (api *API) KeycardLocked(ctx context.Context, kcUID string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).KeycardLocked(ctx, kcUID, clock)
}

func (api *API) KeycardUnlocked(ctx context.Context, kcUID string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).KeycardUnlocked(ctx, kcUID, clock)
}

func (api *API) DeleteKeycard(ctx context.Context, kcUID string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).DeleteKeycard(ctx, kcUID, clock)
}

func (api *API) DeleteKeypair(ctx context.Context, keyUID string) error {
	return api.db.DeleteKeypair(keyUID)
}

func (api *API) UpdateKeycardUID(ctx context.Context, oldKcUID string, newKcUID string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).UpdateKeycardUID(ctx, oldKcUID, newKcUID, clock)
}
