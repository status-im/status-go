package main

// #include <stdlib.h>
import "C"
import (
	"encoding/json"

	mobile "github.com/status-im/status-go/mobile"
)

// MultiAccountGenerate generates account in memory without storing them.
//export MultiAccountGenerate
func MultiAccountGenerate(paramsJSON *C.char) *C.char {
	var p mobile.MultiAccountGenerateParams

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
	var p mobile.MultiAccountGenerateAndDeriveAddressesParams

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
	var p mobile.MultiAccountDeriveAddressesParams

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

// MultiAccountStoreDerivedAccounts derive accounts from the specified key and store them encrypted with the specified password.
//export MultiAccountStoreDerivedAccounts
func MultiAccountStoreDerivedAccounts(paramsJSON *C.char) *C.char {
	var p mobile.MultiAccountStoreDerivedAccountsParams

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
	var p mobile.MultiAccountImportPrivateKeyParams

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

// MultiAccountImportMnemonic imports an account derived from the mnemonic phrase and the Bip39Passphrase storing it.
//export MultiAccountImportMnemonic
func MultiAccountImportMnemonic(paramsJSON *C.char) *C.char {
	var p mobile.MultiAccountImportMnemonicParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
		return makeJSONResponse(err)
	}

	resp, err := statusBackend.AccountManager().AccountsGenerator().ImportMnemonic(p.MnemonicPhrase, p.Bip39Passphrase)
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
	var p mobile.MultiAccountStoreAccountParams

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

// MultiAccountLoadAccount loads in memory the account specified by address unlocking it with password.
//export MultiAccountLoadAccount
func MultiAccountLoadAccount(paramsJSON *C.char) *C.char {
	var p mobile.MultiAccountLoadAccountParams

	if err := json.Unmarshal([]byte(C.GoString(paramsJSON)), &p); err != nil {
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

	return C.CString(string(out))
}

// MultiAccountReset remove all the multi-account keys from memory.
//export MultiAccountReset
func MultiAccountReset() *C.char {
	statusBackend.AccountManager().AccountsGenerator().Reset()
	return makeJSONResponse(nil)
}
