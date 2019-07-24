package statusgo

import (
	"encoding/json"
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

// MultiAccountStoreDerivedParams are the params sent to MultiAccountStoreDerived.
type MultiAccountStoreDerivedParams struct {
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

// MultiAccountStoreDerived derive accounts from the specified key and store them encrypted with the specified password.
func MultiAccountStoreDerived(paramsJSON string) string {
	var p MultiAccountStoreDerivedParams

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
