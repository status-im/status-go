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
	err := (*api.messenger).SaveOrUpdateAccount(account)
	if err != nil {
		return err
	}
	api.feed.Send([]*accounts.Account{account})
	return nil
}

// Setting `Keypair` without `Accounts` will update keypair only.
func (api *API) SaveKeypair(ctx context.Context, keypair *accounts.Keypair) error {
	log.Info("[AccountsAPI::SaveKeypair]")
	err := (*api.messenger).SaveOrUpdateKeypair(keypair)
	if err != nil {
		return err
	}
	for _, acc := range keypair.Accounts {
		api.feed.Send([]*accounts.Account{acc})
	}
	return nil
}

func (api *API) GetAccounts(ctx context.Context) ([]*accounts.Account, error) {
	return api.db.GetAccounts()
}

func (api *API) GetWatchOnlyAccounts(ctx context.Context) ([]*accounts.Account, error) {
	return api.db.GetWatchOnlyAccounts()
}

func (api *API) GetKeypairs(ctx context.Context) ([]*accounts.Keypair, error) {
	return api.db.GetKeypairs()
}

func (api *API) GetAccountByAddress(ctx context.Context, address types.Address) (*accounts.Account, error) {
	return api.db.GetAccountByAddress(address)
}

func (api *API) GetKeypairByKeyUID(ctx context.Context, keyUID string) (*accounts.Keypair, error) {
	return api.db.GetKeypairByKeyUID(keyUID)
}

func (api *API) DeleteAccount(ctx context.Context, address types.Address) error {
	acc, err := api.db.GetAccountByAddress(address)
	if err != nil {
		return err
	}

	if acc.Type != accounts.AccountTypeWatch {
		kp, err := api.db.GetKeypairByKeyUID(acc.KeyUID)
		if err != nil {
			return err
		}

		lastAcccountOfKeypairWithTheSameKey := len(kp.Accounts) == 1

		knownKeycardsForKeyUID, err := api.db.GetKeycardByKeyUID(acc.KeyUID)
		if err != nil {
			return err
		}

		if len(knownKeycardsForKeyUID) == 0 {
			err = api.manager.DeleteAccount(address)
			var e *account.ErrCannotLocateKeyFile
			if err != nil && !errors.As(err, &e) {
				return err
			}

			if acc.Type != accounts.AccountTypeKey {
				if lastAcccountOfKeypairWithTheSameKey {
					err = api.manager.DeleteAccount(types.Address(common.HexToAddress(kp.DerivedFrom)))
					var e *account.ErrCannotLocateKeyFile
					if err != nil && !errors.As(err, &e) {
						return err
					}
				}
			}
		} else {
			if lastAcccountOfKeypairWithTheSameKey {
				knownKeycards, err := api.db.GetAllKnownKeycards()
				if err != nil {
					return err
				}

				for _, kc := range knownKeycards {
					if kc.KeyUID == acc.KeyUID {
						clock := uint64(time.Now().Unix())
						err = (*api.messenger).RemoveMigratedAccountsForKeycard(ctx, kc.KeycardUID, kc.AccountsAddresses, clock)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return (*api.messenger).DeleteAccount(address)
}

func (api *API) AddKeypair(ctx context.Context, password string, keypair *accounts.Keypair) error {
	if len(keypair.KeyUID) == 0 {
		return errors.New("`KeyUID` field of a keypair must be set")
	}

	if len(keypair.Name) == 0 {
		return errors.New("`Name` field of a keypair must be set")
	}

	if len(keypair.Type) == 0 {
		return errors.New("`Type` field of a keypair must be set")
	}

	if keypair.Type != accounts.KeypairTypeKey {
		if len(keypair.DerivedFrom) == 0 {
			return errors.New("`DerivedFrom` field of a keypair must be set")
		}
	}

	for _, acc := range keypair.Accounts {
		if acc.KeyUID != keypair.KeyUID {
			return errors.New("all accounts of a keypair must have the same `KeyUID` as keypair key uid")
		}

		err := api.checkAccountValidity(acc)
		if err != nil {
			return err
		}
	}

	err := api.SaveKeypair(ctx, keypair)
	if err != nil {
		return err
	}

	if len(password) > 0 {
		for _, acc := range keypair.Accounts {
			if acc.Type == accounts.AccountTypeGenerated || acc.Type == accounts.AccountTypeSeed {
				err = api.createKeystoreFileForAccount(keypair.DerivedFrom, password, acc)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (api *API) checkAccountValidity(account *accounts.Account) error {
	if len(account.Address) == 0 {
		return errors.New("`Address` field of an account must be set")
	}

	if len(account.Type) == 0 {
		return errors.New("`Type` field of an account must be set")
	}

	if account.Wallet || account.Chat {
		return errors.New("default wallet and chat account cannot be added this way")
	}

	if len(account.Name) == 0 {
		return errors.New("`Name` field of an account must be set")
	}

	if len(account.Emoji) == 0 {
		return errors.New("`Emoji` field of an account must be set")
	}

	if len(account.Color) == 0 {
		return errors.New("`Color` field of an account must be set")
	}

	if account.Type != accounts.AccountTypeWatch {

		if len(account.KeyUID) == 0 {
			return errors.New("`KeyUID` field of an account must be set")
		}

		if len(account.PublicKey) == 0 {
			return errors.New("`PublicKey` field of an account must be set")
		}

		if account.Type != accounts.AccountTypeKey {
			if len(account.Path) == 0 {
				return errors.New("`Path` field of an account must be set")
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

	return nil
}

func (api *API) createKeystoreFileForAccount(masterAddress string, password string, account *accounts.Account) error {
	if account.Type != accounts.AccountTypeGenerated && account.Type != accounts.AccountTypeSeed {
		return errors.New("cannot create keystore file if account is not of `generated` or `seed` type")
	}
	if masterAddress == "" {
		return errors.New("cannot create keystore file if master address is empty")
	}
	if password == "" {
		return errors.New("cannot create keystore file if password is empty")
	}

	info, err := api.manager.AccountsGenerator().LoadAccount(masterAddress, password)
	if err != nil {
		return err
	}

	_, err = api.manager.AccountsGenerator().StoreDerivedAccounts(info.ID, password, []string{account.Path})
	return err
}

func (api *API) AddAccount(ctx context.Context, password string, account *accounts.Account) error {
	err := api.checkAccountValidity(account)
	if err != nil {
		return err
	}

	if account.Type != accounts.AccountTypeWatch {
		kp, err := api.db.GetKeypairByKeyUID(account.KeyUID)
		if err != nil {
			if err == accounts.ErrDbKeypairNotFound {
				return errors.New("cannot add an account for an unknown keypair")
			}
			return err
		}

		// we need to create local keystore file only if password is provided and the account is being added is of
		// "generated" or "seed" type.
		if (account.Type == accounts.AccountTypeGenerated || account.Type == accounts.AccountTypeSeed) && len(password) > 0 {
			err = api.createKeystoreFileForAccount(kp.DerivedFrom, password, account)
			if err != nil {
				return err
			}
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

	kp, err := api.db.GetKeypairByKeyUID(info.KeyUID)
	if err != nil && err != accounts.ErrDbKeypairNotFound {
		return err
	}

	if kp != nil {
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

	kp, err := api.db.GetKeypairByKeyUID(generatedAccountInfo.KeyUID)
	if err != nil && err != accounts.ErrDbKeypairNotFound {
		return err
	}

	if kp != nil {
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

// If keypair is migrated from keycard to app, then `accountsComingFromKeycard` should be set to true, otherwise false.
func (api *API) AddKeycardOrAddAccountsIfKeycardIsAdded(ctx context.Context, kcUID string, kpName string, keyUID string,
	accountAddresses []string, accountsComingFromKeycard bool) error {
	if len(accountAddresses) == 0 {
		return errors.New("cannot migrate a keypair without accounts")
	}

	kpDb, err := api.db.GetKeypairByKeyUID(keyUID)
	if err != nil {
		if err == accounts.ErrDbKeypairNotFound {
			return errors.New("cannot migrate an unknown keypair")
		}
		return err
	}

	kp := accounts.Keycard{
		KeycardUID:      kcUID,
		KeycardName:     kpName,
		KeycardLocked:   false,
		KeyUID:          keyUID,
		LastUpdateClock: uint64(time.Now().Unix()),
	}
	for _, addr := range accountAddresses {
		kp.AccountsAddresses = append(kp.AccountsAddresses, types.Address(common.HexToAddress(addr)))
	}

	knownKeycardsForKeyUID, err := api.db.GetKeycardByKeyUID(keyUID)
	if err != nil {
		return err
	}

	added, err := (*api.messenger).AddKeycardOrAddAccountsIfKeycardIsAdded(ctx, &kp)
	if err != nil {
		return err
	}

	if !accountsComingFromKeycard {
		// Once we migrate a keypair, corresponding keystore files need to be deleted
		// if the keypair being migrated is not already migrated (in case user is creating a copy of an existing Keycard)
		if added && len(knownKeycardsForKeyUID) == 0 {
			for _, addr := range kp.AccountsAddresses {
				err = api.manager.DeleteAccount(addr)
				if err != nil {
					return err
				}
			}

			err = api.manager.DeleteAccount(types.Address(common.HexToAddress(kpDb.DerivedFrom)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (api *API) RemoveMigratedAccountsForKeycard(ctx context.Context, kcUID string, accountAddresses []string) error {
	clock := uint64(time.Now().Unix())
	var addresses []types.Address
	for _, addr := range accountAddresses {
		addresses = append(addresses, types.HexToAddress(addr))
	}
	return (*api.messenger).RemoveMigratedAccountsForKeycard(ctx, kcUID, addresses, clock)
}

func (api *API) GetAllKnownKeycards(ctx context.Context) ([]*accounts.Keycard, error) {
	return api.db.GetAllKnownKeycards()
}

func (api *API) GetAllKnownKeycardsGroupedByKeyUID(ctx context.Context) ([]*accounts.Keycard, error) {
	return api.db.GetAllKnownKeycardsGroupedByKeyUID()
}

func (api *API) GetKeycardByKeyUID(ctx context.Context, keyUID string) ([]*accounts.Keycard, error) {
	return api.db.GetKeycardByKeyUID(keyUID)
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

func (api *API) DeleteAllKeycardsWithKeyUID(ctx context.Context, keyUID string) error {
	return api.db.DeleteAllKeycardsWithKeyUID(keyUID)
}

func (api *API) UpdateKeycardUID(ctx context.Context, oldKcUID string, newKcUID string) error {
	clock := uint64(time.Now().Unix())
	return (*api.messenger).UpdateKeycardUID(ctx, oldKcUID, newKcUID, clock)
}
