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
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")

	err := api.verifyPassword(password)
	if err != nil {
		return err
	}

	generatedAccountInfo, err := api.manager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return err
	}

	accountInfos, err := api.manager.AccountsGenerator().StoreDerivedAccounts(generatedAccountInfo.ID, password, []string{pathDefaultWallet})
	if err != nil {
		return err
	}

	addressExists, err := api.db.AddressExists(types.Address(common.HexToAddress(accountInfos[pathWalletRoot].Address)))
	if err != nil {
		return err
	}
	if addressExists {
		return errors.New("account already exists")
	}

	account := accounts.Account{
		Address:   types.Address(common.HexToAddress(accountInfos[pathDefaultWallet].Address)),
		PublicKey: types.HexBytes(accountInfos[pathDefaultWallet].PublicKey),
		Type:      "seed",
		Name:      name,
		Emoji:     emoji,
		Color:     color,
		Path:      pathDefaultWallet,
	}
	return api.SaveAccounts(ctx, []accounts.Account{account})
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
	err := api.verifyPassword(password)
	if err != nil {
		return err
	}

	address, err := api.db.GetWalletRoodAddress()
	if err != nil {
		return err
	}

	info, err := api.manager.AccountsGenerator().LoadAccount(address.Hex(), password)
	if err != nil {
		return err
	}

	latestDerivedPath, err := api.db.GetLatestDerivedPath()
	if err != nil {
		return err
	}
	newDerivedPath := latestDerivedPath + 1
	path := fmt.Sprint("m/", newDerivedPath)
	infos, err := api.manager.AccountsGenerator().DeriveAddresses(info.ID, []string{path})
	if err != nil {
		return err
	}

	_, err = api.manager.AccountsGenerator().StoreDerivedAccounts(info.ID, password, []string{path})
	if err != nil {
		return err
	}

	acc := accounts.Account{
		Address:   types.Address(common.HexToAddress(infos[path].Address)),
		PublicKey: types.HexBytes(infos[path].PublicKey),
		Type:      "generated",
		Name:      name,
		Emoji:     emoji,
		Color:     color,
		Path:      fmt.Sprint(pathWalletRoot, "/", newDerivedPath),
	}

	err = api.db.SaveSettingField(settings.LatestDerivedPath, newDerivedPath)
	if err != nil {
		return err
	}

	return api.SaveAccounts(ctx, []accounts.Account{acc})
}

func (api *API) verifyPassword(password string) error {
	address, err := api.db.GetChatAddress()
	if err != nil {
		return err
	}
	_, err = api.manager.VerifyAccountPassword(api.config.KeyStoreDir, address.Hex(), password)
	return err
}
