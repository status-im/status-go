package statusaccounts

import (
	"context"
)

type API struct {
	s *Service
}

// Generate generates n accounts identified by mnemonic phrases with mnemonicPhraseLength words
// and using bip39Passphrase as extra entropy if needed.
// A slice of GeneratedAccountInfo is returned with each account identified by an ID, a public key, and an address.
// Keys are kept in memory until explicitly stored.
func (api *API) Generate(ctx context.Context, mnemonicPhraseLength int, n int, bip39Passphrase string) ([]GeneratedAccountInfo, error) {
	return api.s.g.generate(mnemonicPhraseLength, n, bip39Passphrase)
}

// GenerateAndDeriveAddresses combines Generate and DeriveAddresses in one call.
func (api *API) GenerateAndDeriveAddresses(ctx context.Context, mnemonicPhraseLength int, n int, bip39Passphrase string, paths []string) ([]GeneratedAndDerivedAccountInfo, error) {
	return api.s.g.generateAndDeriveAddresses(mnemonicPhraseLength, n, bip39Passphrase, paths)
}

// ImportMnemonic generates a master key from mnemonicPhrase and bip39Passphrase and keeps it in memory
// until explicitly stored.
func (api *API) ImportMnemonic(ctx context.Context, mnemonicPhrase string, bip39Passphrase string) (GeneratedAccountInfo, error) {
	return api.s.g.importMnemonic(mnemonicPhrase, bip39Passphrase)
}

// ImportPrivateKey imports a raw hex private key that is kept in memory until explicitly stored.
func (api *API) ImportPrivateKey(ctx context.Context, privateKeyHex string) (IdentifiedAccountInfo, error) {
	return api.s.g.importPrivateKey(privateKeyHex)
}

// ImportJSONKey decrypts a JSON keystore with password and imports the private key keeping it
// in memory until explicitly stored.
func (api *API) ImportJSONKey(ctx context.Context, json string, password string) (IdentifiedAccountInfo, error) {
	return api.s.g.importJSONKey(json, password)
}

// DeriveAddresses returns a list of addresses corresponding to keys derived with pathStrings from
// a private key identified by accountID. The selected account must have an extended key to be
// able to derive child keys.
func (api *API) DeriveAddresses(ctx context.Context, accountID string, paths []string) (map[string]AccountInfo, error) {
	return api.s.g.deriveAddresses(accountID, paths)
}

// StoreAccount selects account with ID accountID and stores it encrypted with password.
// After storing it, all accounts are removed from memory.
func (api *API) StoreAccount(ctx context.Context, accountID string, password string) (AccountInfo, error) {
	return api.s.g.storeAccount(accountID, password)
}

// StoreDerivedAccounts selects an account with ID accountID, derives child keys using pathStrings,
// and stores the derived child keys encrypting them with password.
// After storing them, all accounts are removed from memory.
func (api *API) StoreDerivedAccounts(ctx context.Context, accountID string, password string, paths []string) (map[string]AccountInfo, error) {
	return api.s.g.storeDerivedAccounts(accountID, password, paths)
}

// LoadAccount loads in memory an account previously stored.
// The account is identified by its address and decrypted using password.
func (api *API) LoadAccount(ctx context.Context, address string, password string) (IdentifiedAccountInfo, error) {
	return api.s.g.loadAccount(address, password)
}

// Reset resets the accounts mapping removing all the keys from memory.
func (api *API) Reset(ctx context.Context) {
	api.s.g.Reset()
}
