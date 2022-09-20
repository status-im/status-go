package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/keypairs"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
)

const pathWalletRoot = "m/44'/60'/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"

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

func (api *API) SaveAccounts(ctx context.Context, accounts []*accounts.Account) error {
	log.Info("[AccountsAPI::SaveAccounts]")
	err := (*api.messenger).SaveAccounts(accounts)
	if err != nil {
		return err
	}
	api.feed.Send(accounts)
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

func (api *API) DeleteAccount(ctx context.Context, address types.Address) error {
	acc, err := api.db.GetAccountByAddress(address)
	if err != nil {
		return err
	}
	if acc.Type != accounts.AccountTypeWatch {
		err = api.manager.DeleteAccount(api.config.KeyStoreDir, address)
		var e *account.ErrCannotLocateKeyFile
		if err != nil && !errors.As(err, &e) {
			return err
		}
	}

	return (*api.messenger).DeleteAccount(address)
}

func (api *API) AddAccountWatch(ctx context.Context, address string, name string, color string, emoji string) error {
	account := &accounts.Account{
		Address: types.Address(common.HexToAddress(address)),
		Type:    accounts.AccountTypeWatch,
		Name:    name,
		Emoji:   emoji,
		Color:   color,
	}
	return api.SaveAccounts(ctx, []*accounts.Account{account})
}

func (api *API) AddAccountWithMnemonic(
	ctx context.Context,
	mnemonic string,
	password string,
	name string,
	color string,
	emoji string,
) error {
	return api.addAccountWithMnemonic(ctx, mnemonic, password, name, color, emoji, pathWalletRoot)
}

func (api *API) AddAccountWithMnemonicAndPath(
	ctx context.Context,
	mnemonic string,
	password string,
	name string,
	color string,
	emoji string,
	path string,
) error {
	return api.addAccountWithMnemonic(ctx, mnemonic, password, name, color, emoji, path)
}

func (api *API) AddAccountWithPrivateKey(
	ctx context.Context,
	privateKey string,
	password string,
	name string,
	color string,
	emoji string,
) error {
	err := api.verifyPassword(password)
	if err != nil {
		return err
	}

	info, err := api.manager.AccountsGenerator().ImportPrivateKey(privateKey)
	if err != nil {
		return err
	}

	addressExists, err := api.db.AddressExists(types.Address(common.HexToAddress(info.Address)))
	if err != nil {
		return err
	}
	if addressExists {
		return errors.New("account already exists")
	}

	_, err = api.manager.AccountsGenerator().StoreAccount(info.ID, password)
	if err != nil {
		return err
	}

	account := &accounts.Account{
		Address:   types.Address(common.HexToAddress(info.Address)),
		KeyUID:    info.KeyUID,
		PublicKey: types.HexBytes(info.PublicKey),
		Type:      accounts.AccountTypeKey,
		Name:      name,
		Emoji:     emoji,
		Color:     color,
		Path:      pathDefaultWallet,
	}

	return api.SaveAccounts(ctx, []*accounts.Account{account})
}

func (api *API) GenerateAccount(
	ctx context.Context,
	password string,
	name string,
	color string,
	emoji string,
) error {
	address, err := api.db.GetWalletRootAddress()
	if err != nil {
		return err
	}

	latestDerivedPath, err := api.db.GetLatestDerivedPath()
	if err != nil {
		return err
	}

	newDerivedPath := latestDerivedPath + 1
	path := fmt.Sprint(pathWalletRoot, "/", newDerivedPath)

	err = api.generateAccount(ctx, password, name, color, emoji, path, address.Hex())
	if err != nil {
		return err
	}

	err = api.db.SaveSettingField(settings.LatestDerivedPath, newDerivedPath)
	if err != nil {
		return err
	}

	return err
}

func (api *API) GenerateAccountWithDerivedPath(
	ctx context.Context,
	password string,
	name string,
	color string,
	emoji string,
	path string,
	derivedFrom string,
) error {
	return api.generateAccount(ctx, password, name, color, emoji, path, derivedFrom)
}

func (api *API) verifyPassword(password string) error {
	address, err := api.db.GetChatAddress()
	if err != nil {
		return err
	}
	_, err = api.manager.VerifyAccountPassword(api.config.KeyStoreDir, address.Hex(), password)
	return err
}

func (api *API) addAccountWithMnemonic(
	ctx context.Context,
	mnemonic string,
	password string,
	name string,
	color string,
	emoji string,
	path string,
) error {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	err := api.verifyPassword(password)
	if err != nil {
		return err
	}

	generatedAccountInfo, err := api.manager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return err
	}

	_, err = api.manager.AccountsGenerator().StoreAccount(generatedAccountInfo.ID, password)
	if err != nil {
		return err
	}

	accountinfos, err := api.manager.AccountsGenerator().StoreDerivedAccounts(generatedAccountInfo.ID, password, []string{path})
	if err != nil {
		return err
	}

	account := &accounts.Account{
		Address:     types.Address(common.HexToAddress(accountinfos[path].Address)),
		KeyUID:      generatedAccountInfo.KeyUID,
		PublicKey:   types.HexBytes(accountinfos[path].PublicKey),
		Type:        accounts.AccountTypeSeed,
		Name:        name,
		Emoji:       emoji,
		Color:       color,
		Path:        path,
		DerivedFrom: generatedAccountInfo.Address,
	}
	return api.SaveAccounts(ctx, []*accounts.Account{account})
}

func (api *API) generateAccount(
	ctx context.Context,
	password string,
	name string,
	color string,
	emoji string,
	path string,
	address string,
) error {
	err := api.verifyPassword(password)
	if err != nil {
		return err
	}

	info, err := api.manager.AccountsGenerator().LoadAccount(address, password)
	if err != nil {
		return err
	}

	infos, err := api.manager.AccountsGenerator().DeriveAddresses(info.ID, []string{path})
	if err != nil {
		return err
	}

	_, err = api.manager.AccountsGenerator().StoreDerivedAccounts(info.ID, password, []string{path})
	if err != nil {
		return err
	}

	acc := &accounts.Account{
		Address:     types.Address(common.HexToAddress(infos[path].Address)),
		KeyUID:      info.KeyUID,
		PublicKey:   types.HexBytes(infos[path].PublicKey),
		Type:        accounts.AccountTypeGenerated,
		Name:        name,
		Emoji:       emoji,
		Color:       color,
		Path:        path,
		DerivedFrom: address,
	}

	return api.SaveAccounts(ctx, []*accounts.Account{acc})
}

func (api *API) VerifyPassword(password string) bool {
	err := api.verifyPassword(password)
	if err != nil {
		return false
	}
	return true
}

func (api *API) AddMigratedKeyPair(ctx context.Context, kcUID string, kpName string, keyUID string, accountAddresses []string) error {
	var addresses []types.Address
	for _, addr := range accountAddresses {
		addresses = append(addresses, types.Address(common.HexToAddress(addr)))
	}

	return api.db.AddMigratedKeyPair(kcUID, kpName, keyUID, addresses)
}

func (api *API) GetAllMigratedKeyPairs(ctx context.Context) ([]*keypairs.KeyPair, error) {
	return api.db.GetAllMigratedKeyPairs()
}

func (api *API) GetMigratedKeyPairByKeyUID(ctx context.Context, keyUID string) ([]*keypairs.KeyPair, error) {
	return api.db.GetMigratedKeyPairByKeyUID(keyUID)
}

func (api *API) SetKeycardName(ctx context.Context, kcUID string, kpName string) error {
	return api.db.SetKeycardName(kcUID, kpName)
}

func (api *API) KeycardLocked(ctx context.Context, kcUID string) error {
	return api.db.KeycardLocked(kcUID)
}

func (api *API) KeycardUnlocked(ctx context.Context, kcUID string) error {
	return api.db.KeycardUnlocked(kcUID)
}

func (api *API) DeleteKeycard(ctx context.Context, kcUID string) error {
	return api.db.DeleteKeycard(kcUID)
}

func (api *API) UpdateKeycardUID(ctx context.Context, oldKcUID string, newKcUID string) error {
	return api.db.UpdateKeycardUID(oldKcUID, newKcUID)
}
