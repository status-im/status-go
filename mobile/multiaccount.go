package statusgo

import (
	"encoding/json"
	"strings"
)

// MultiAccountGenerateParams are the params sent to MultiAccountGenerate.
type MultiAccountGenerateParams struct {
	N                    int    `json:"n"`
	MnemonicPhraseLength int    `json:"mnemonicPhraseLength"`
	Bip39Passphrase      string `json:"bip39Passphrase"`
}

// MultiAccountGenerateAndDeriveAddressesParams are the params sent to MultiAccountGenerateAndDeriveAddresses.
type MultiAccountGenerateAndDeriveAddressesParams struct {
	MultiAccountGenerateParams
	Paths []string `json:"paths"`
}

// MultiAccountDeriveAddressesParams are the params sent to MultiAccountDeriveAddresses.
type MultiAccountDeriveAddressesParams struct {
	AccountID string   `json:"accountID"`
	Paths     []string `json:"paths"`
}

// MultiAccountStoreDerivedAccountsParams are the params sent to MultiAccountStoreDerivedAccounts.
type MultiAccountStoreDerivedAccountsParams struct {
	MultiAccountDeriveAddressesParams
	Password string `json:"password"`
}

// MultiAccountStoreAccountParams are the params sent to MultiAccountStoreAccount.
type MultiAccountStoreAccountParams struct {
	AccountID string `json:"accountID"`
	Password  string `json:"password"`
}

// MultiAccountImportPrivateKeyParams are the params sent to MultiAccountImportPrivateKey.
type MultiAccountImportPrivateKeyParams struct {
	PrivateKey string `json:"privateKey"`
}

// MultiAccountLoadAccountParams are the params sent to MultiAccountLoadAccount.
type MultiAccountLoadAccountParams struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

// MultiAccountImportMnemonicParams are the params sent to MultiAccountImportMnemonic.
type MultiAccountImportMnemonicParams struct {
	MnemonicPhrase  string   `json:"mnemonicPhrase"`
	Bip39Passphrase string   `json:"Bip39Passphrase"`
	Paths           []string `json:"paths"`
}

// MultiAccountGenerate generates account in memory without storing them.
func MultiAccountGenerate(paramsJSON string) string {
	var p MultiAccountGenerateParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().Generate(p.MnemonicPhraseLength, p.N, p.Bip39Passphrase)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountGenerateAndDeriveAddresses combines Generate and DeriveAddresses in one call.
func MultiAccountGenerateAndDeriveAddresses(paramsJSON string) string {
	var p MultiAccountGenerateAndDeriveAddressesParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().GenerateAndDeriveAddresses(p.MnemonicPhraseLength, p.N, p.Bip39Passphrase, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountDeriveAddresses derive addresses from an account selected by ID, without storing them.
func MultiAccountDeriveAddresses(paramsJSON string) string {
	var p MultiAccountDeriveAddressesParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().DeriveAddresses(p.AccountID, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountStoreDerivedAccounts derive accounts from the specified key and store them encrypted with the specified password.
func MultiAccountStoreDerivedAccounts(paramsJSON string) string {
	var p MultiAccountStoreDerivedAccountsParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().StoreDerivedAccounts(p.AccountID, p.Password, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// CreateAccountFromPrivateKey returns an account derived from the private key without storing it
func CreateAccountFromPrivateKey(paramsJSON string) string {
	var p MultiAccountImportPrivateKeyParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().CreateAccountFromPrivateKey(p.PrivateKey)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountImportPrivateKey imports a raw private key without storing it.
func MultiAccountImportPrivateKey(paramsJSON string) string {
	var p MultiAccountImportPrivateKeyParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().ImportPrivateKey(p.PrivateKey)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// CreateAccountFromMnemonicAndDeriveAccountsForPaths returns an account derived from the mnemonic phrase and the Bip39Passphrase
// and generate derived accounts for the list of paths without storing it
func CreateAccountFromMnemonicAndDeriveAccountsForPaths(paramsJSON string) string {
	var p MultiAccountImportMnemonicParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	// remove any duplicate whitespaces
	mnemonicPhraseNoExtraSpaces := strings.Join(strings.Fields(p.MnemonicPhrase), " ")

	resp, err := statusBackend.AccountManager().AccountsGenerator().CreateAccountFromMnemonicAndDeriveAccountsForPaths(mnemonicPhraseNoExtraSpaces, p.Bip39Passphrase, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountImportMnemonic imports an account derived from the mnemonic phrase and the Bip39Passphrase storing it.
func MultiAccountImportMnemonic(paramsJSON string) string {
	var p MultiAccountImportMnemonicParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	// remove any duplicate whitespaces
	mnemonicPhraseNoExtraSpaces := strings.Join(strings.Fields(p.MnemonicPhrase), " ")

	resp, err := statusBackend.AccountManager().AccountsGenerator().ImportMnemonic(mnemonicPhraseNoExtraSpaces, p.Bip39Passphrase)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountStoreAccount stores the select account.
func MultiAccountStoreAccount(paramsJSON string) string {
	var p MultiAccountStoreAccountParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().StoreAccount(p.AccountID, p.Password)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountLoadAccount loads in memory the account specified by address unlocking it with password.
func MultiAccountLoadAccount(paramsJSON string) string {
	var p MultiAccountLoadAccountParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().LoadAccount(p.Address, p.Password)
	if err != nil {
		return makeJSONResponse(err)
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// MultiAccountReset remove all the multi-account keys from memory.
func MultiAccountReset() string {
	statusBackend.AccountManager().AccountsGenerator().Reset()
	return makeJSONResponse(nil)
}
