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
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
)

const pathWalletRoot = "m/44'/60'/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"

func NewAccountsAPI(manager *account.GethManager, config *params.NodeConfig, db *accounts.Database, feed *event.Feed) *API {
	return &API{manager, config, db, feed}
}

// API is class with methods available over RPC.
type API struct {
	manager *account.GethManager
	config  *params.NodeConfig
	db      *accounts.Database
	feed    *event.Feed
}

type DerivedAddress struct {
	Address     common.Address `json:"address"`
	Path        string         `json:"path"`
	HasActivity bool           `json:"hasActivity"`
}

func (api *API) SaveAccounts(ctx context.Context, accounts []accounts.Account) error {
	log.Info("[AccountsAPI::SaveAccounts]")
	err := api.db.SaveAccounts(accounts)
	if err != nil {
		return err
	}
	api.feed.Send(accounts)
	return nil
}

func (api *API) GetAccounts(ctx context.Context) ([]accounts.Account, error) {
	return api.db.GetAccounts()
}

func (api *API) DeleteAccount(ctx context.Context, address types.Address) error {
	return api.db.DeleteAccount(address)
}

func (api *API) AddAccountWatch(ctx context.Context, address string, name string, color string, emoji string) error {
	account := accounts.Account{
		Address: types.Address(common.HexToAddress(address)),
		Type:    "watch",
		Name:    name,
		Emoji:   emoji,
		Color:   color,
	}
	return api.SaveAccounts(ctx, []accounts.Account{account})
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

	account := accounts.Account{
		Address:   types.Address(common.HexToAddress(info.Address)),
		PublicKey: types.HexBytes(info.PublicKey),
		Type:      "key",
		Name:      name,
		Emoji:     emoji,
		Color:     color,
		Path:      pathDefaultWallet,
	}

	return api.SaveAccounts(ctx, []accounts.Account{account})
}

func (api *API) GenerateAccount(
	ctx context.Context,
	password string,
	name string,
	color string,
	emoji string,
) error {

	address, err := api.db.GetWalletRoodAddress()
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

func (api *API) GetDerivedAddressesForPath(password string, derivedFrom string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	info, err := api.manager.AccountsGenerator().LoadAccount(derivedFrom, password)
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(info.ID, path, pageSize, pageNumber)
}

func (api *API) GetDerivedAddressesForMenominicWithPath(mnemonic string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	info, err := api.manager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return nil, err
	}

	return api.getDerivedAddresses(info.ID, path, pageSize, pageNumber)
}

func (api *API) verifyPassword(password string) error {
	address, err := api.db.GetChatAddress()
	if err != nil {
		return err
	}
	_, err = api.manager.VerifyAccountPassword(api.config.KeyStoreDir, address.Hex(), password)
	return err
}

func (api *API) getDerivedAddresses(id string, path string, pageSize int, pageNumber int) ([]*DerivedAddress, error) {
	derivedAddresses := make([]*DerivedAddress, 0)

	if pageNumber <= 0 || pageSize <= 0 {
		return nil, fmt.Errorf("pageSize and pageNumber should be greater than 0")
	}

	var startIndex = ((pageNumber - 1) * pageSize)
	var endIndex = (pageNumber * pageSize)

	for i := startIndex; i < endIndex; i++ {
		derivedPath := fmt.Sprint(path, "/", i)

		info, err := api.manager.AccountsGenerator().DeriveAddresses(id, []string{derivedPath})
		if err != nil {
			return nil, err
		}

		address := &DerivedAddress{
			Address:     common.HexToAddress(info[derivedPath].Address),
			Path:        derivedPath,
			HasActivity: false,
		}

		derivedAddresses = append(derivedAddresses, address)
	}
	return derivedAddresses, nil
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

	account := accounts.Account{
		Address:     types.Address(common.HexToAddress(accountinfos[path].Address)),
		PublicKey:   types.HexBytes(accountinfos[path].PublicKey),
		Type:        "seed",
		Name:        name,
		Emoji:       emoji,
		Color:       color,
		Path:        path,
		DerivedFrom: generatedAccountInfo.Address,
	}
	return api.SaveAccounts(ctx, []accounts.Account{account})
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

	acc := accounts.Account{
		Address:     types.Address(common.HexToAddress(infos[path].Address)),
		PublicKey:   types.HexBytes(infos[path].PublicKey),
		Type:        "generated",
		Name:        name,
		Emoji:       emoji,
		Color:       color,
		Path:        path,
		DerivedFrom: address,
	}

	return api.SaveAccounts(ctx, []accounts.Account{acc})
}
