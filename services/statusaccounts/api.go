package statusaccounts

import (
	"context"
)

type API struct {
	s *Service
}

func (api *API) Generate(ctx context.Context, mnemonicPhraseLength int, n int) ([]CreatedAccountInfo, error) {
	return api.s.g.generate(mnemonicPhraseLength, n)
}

func (api *API) ImportMnemonic(ctx context.Context, mnemonic string) (CreatedAccountInfo, error) {
	return api.s.g.importMnemonic(mnemonic)
}

func (api *API) ImportPrivateKey(ctx context.Context, privateKeyHex string) (IdentifiedAccountInfo, error) {
	return api.s.g.importPrivateKey(privateKeyHex)
}

func (api *API) ImportJSONKey(ctx context.Context, json string, password string) (IdentifiedAccountInfo, error) {
	return api.s.g.importJSONKey(json, password)
}

func (api *API) DeriveAddresses(ctx context.Context, accountID string, paths []string) (map[string]AccountInfo, error) {
	return api.s.g.deriveAddresses(accountID, paths)
}

func (api *API) StoreAccount(ctx context.Context, accountID string, password string) (AccountInfo, error) {
	return api.s.g.storeAccount(accountID, password)
}

func (api *API) StoreDerivedAccounts(ctx context.Context, accountID string, password string, paths []string) (map[string]AccountInfo, error) {
	return api.s.g.storeDerivedAccounts(accountID, password, paths)
}

func (api *API) LoadAccount(ctx context.Context, address string, password string) (IdentifiedAccountInfo, error) {
	return api.s.g.loadAccount(address, password)
}
