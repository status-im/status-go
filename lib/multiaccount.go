package main

// #include <stdlib.h>
import "C"
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
//export MultiAccountGenerate
func MultiAccountGenerate(paramsJSON *C.char) *C.char {
	var p MultiAccountGenerateParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountGenerateAndDeriveAddresses combines Generate and DeriveAddresses in one call.
//export MultiAccountGenerateAndDeriveAddresses
func MultiAccountGenerateAndDeriveAddresses(paramsJSON *C.char) *C.char {
	var p MultiAccountGenerateAndDeriveAddressesParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountDeriveAddresses derive addresses from an account selected by ID, without storing them.
//export MultiAccountDeriveAddresses
func MultiAccountDeriveAddresses(paramsJSON *C.char) *C.char {
	var p MultiAccountDeriveAddressesParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountStoreDerived derive accounts from the specified key and store them encrypted with the specified password.
//export MultiAccountStoreDerived
func MultiAccountStoreDerived(paramsJSON *C.char) *C.char {
	var p MultiAccountStoreDerivedParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountImportPrivateKey imports a raw private key without storing it.
//export MultiAccountImportPrivateKey
func MultiAccountImportPrivateKey(paramsJSON *C.char) *C.char {
	var p MultiAccountImportPrivateKeyParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountStoreAccount stores the select account.
//export MultiAccountStoreAccount
func MultiAccountStoreAccount(paramsJSON *C.char) *C.char {
	var p MultiAccountStoreAccountParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}
