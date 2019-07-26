// +build e2e_test

package main

import (
	"C"
	"encoding/json"

	"github.com/ethereum/go-ethereum/crypto"
	mobile "github.com/status-im/status-go/mobile"
)
import (
	"fmt"
	"strings"
	"testing"

	"github.com/status-im/status-go/account/generator"
)

func checkMultiAccountErrorResponse(t *testing.T, respJSON *C.char, expectedError string) {
	var e struct {
		Error *string `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(C.GoString(respJSON)), &e); err != nil {
		t.Fatalf("error unmarshaling error response")
	}

	if e.Error == nil {
		t.Fatalf("unexpected empty error. expected %s, got nil", expectedError)
	}

	if *e.Error != expectedError {
		t.Fatalf("unexpected error. expected %s, got %+v", expectedError, *e.Error)
	}
}

func checkMultiAccountResponse(t *testing.T, respJSON *C.char, resp interface{}) {
	var e struct {
		Error *string `json:"error,omitempty"`
	}

	json.Unmarshal([]byte(C.GoString(respJSON)), &e)
	if e.Error != nil {
		t.Errorf("unexpected response error: %s", *e.Error)
	}

	if err := json.Unmarshal([]byte(C.GoString(respJSON)), resp); err != nil {
		t.Fatalf("error unmarshaling response to expected struct: %s", err)
	}
}

func testMultiAccountGenerateDeriveStoreLoadReset(t *testing.T) bool { //nolint: gocyclo
	// to make sure that we start with empty account (which might have gotten populated during previous tests)
	if err := statusBackend.Logout(); err != nil {
		t.Fatal(err)
	}

	params := C.CString(`{
		"n": 2,
		"mnemonicPhraseLength": 24,
		"bip39Passphrase": ""
	}`)

	// generate 2 random accounts
	rawResp := MultiAccountGenerate(params)
	var generateResp []generator.GeneratedAccountInfo
	// check there's no error in the response
	checkMultiAccountResponse(t, rawResp, &generateResp)
	if len(generateResp) != 2 {
		t.Errorf("expected 2 accounts created, got %d", len(generateResp))
		return false
	}

	bip44DerivationPath := "m/44'/60'/0'/0/0"
	eip1581DerivationPath := "m/43'/60'/1581'/0'/0"
	paths := []string{bip44DerivationPath, eip1581DerivationPath}

	// derive 2 child accounts for each account without storing them
	for i := 0; i < len(generateResp); i++ {
		info := generateResp[i]
		mnemonicLength := len(strings.Split(info.Mnemonic, " "))

		if mnemonicLength != 24 {
			t.Errorf("expected mnemonic to have 24 words, got %d", mnemonicLength)
			return false
		}

		if _, ok := testMultiAccountDeriveAddresses(t, info.ID, paths, false); !ok {
			return false
		}
	}

	password := "multi-account-test-password"

	// store 2 derived child accounts from the first account.
	// after that all the generated account should be remove from memory.
	addresses, ok := testMultiAccountStoreDerived(t, generateResp[0].ID, password, paths)
	if !ok {
		return false
	}

	loadedIDs := make([]string, 0)

	// unlock and load all stored accounts.
	for _, address := range addresses {
		loadedID, ok := testMultiAccountLoadAccount(t, address, password)
		if !ok {
			return false
		}

		loadedIDs = append(loadedIDs, loadedID)

		if _, ok := testMultiAccountDeriveAddresses(t, loadedID, paths, false); !ok {
			return false
		}
	}

	rawResp = MultiAccountReset()

	// try again deriving addresses.
	// it should fail because reset should remove all the accounts from memory.
	for _, loadedID := range loadedIDs {
		if _, ok := testMultiAccountDeriveAddresses(t, loadedID, paths, true); !ok {
			t.Errorf("account is still in memory, expected Reset to remove all accounts")
			return false
		}
	}

	return true
}

func testMultiAccountImportMnemonicAndDerive(t *testing.T) bool { //nolint: gocyclo
	// to make sure that we start with empty account (which might have gotten populated during previous tests)
	if err := statusBackend.Logout(); err != nil {
		t.Fatal(err)
	}

	mnemonicPhrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	bip39Passphrase := "TREZOR"
	params := mobile.MultiAccountImportMnemonicParams{
		MnemonicPhrase:  mnemonicPhrase,
		Bip39Passphrase: bip39Passphrase,
	}

	paramsJSON, err := json.Marshal(&params)
	if err != nil {
		t.Errorf("error encoding MultiAccountImportMnemonicParams")
		return false
	}

	// import mnemonic
	rawResp := MultiAccountImportMnemonic(C.CString(string(paramsJSON)))
	var importResp generator.IdentifiedAccountInfo
	// check the response doesn't have errors
	checkMultiAccountResponse(t, rawResp, &importResp)

	bip44DerivationPath := "m/44'/60'/0'/0/0"
	expectedBip44Address := "0x9c32F71D4DB8Fb9e1A58B0a80dF79935e7256FA6"
	addresses, ok := testMultiAccountDeriveAddresses(t, importResp.ID, []string{bip44DerivationPath}, false)
	if !ok {
		return false
	}

	if addresses[bip44DerivationPath] != expectedBip44Address {
		t.Errorf("unexpected address; expected %s, got %s", expectedBip44Address, addresses[bip44DerivationPath])
		return false
	}

	return true
}

func testMultiAccountDeriveAddresses(t *testing.T, accountID string, paths []string, expectAccountNotFoundError bool) (map[string]string, bool) { //nolint: gocyclo
	params := mobile.MultiAccountDeriveAddressesParams{
		AccountID: accountID,
		Paths:     paths,
	}

	paramsJSON, err := json.Marshal(&params)
	if err != nil {
		t.Errorf("error encoding MultiAccountDeriveAddressesParams")
		return nil, false
	}

	// derive addresses from account accountID
	rawResp := MultiAccountDeriveAddresses(C.CString(string(paramsJSON)))

	if expectAccountNotFoundError {
		checkMultiAccountErrorResponse(t, rawResp, "account not found")
		return nil, true
	}

	var deriveResp map[string]generator.AccountInfo
	// check the response doesn't have errors
	checkMultiAccountResponse(t, rawResp, &deriveResp)
	if len(deriveResp) != len(paths) {
		t.Errorf("expected %d derived accounts info, got %d", len(paths), len(deriveResp))
		return nil, false
	}

	addresses := make(map[string]string)

	// check that we have an address for each derivation path we used.
	for _, path := range paths {
		info, ok := deriveResp[path]
		if !ok {
			t.Errorf("results doesn't contain account info for path %s", path)
			return nil, false
		}

		addresses[path] = info.Address
	}

	return addresses, true
}

func testMultiAccountStoreDerived(t *testing.T, accountID string, password string, paths []string) ([]string, bool) { //nolint: gocyclo

	params := mobile.MultiAccountStoreDerivedAccountsParams{
		MultiAccountDeriveAddressesParams: mobile.MultiAccountDeriveAddressesParams{
			AccountID: accountID,
			Paths:     paths,
		},
		Password: password,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Errorf("error encoding MultiAccountStoreDerivedParams")
		return nil, false
	}

	// store one child account for each derivation path.
	rawResp := MultiAccountStoreDerivedAccounts(C.CString(string(paramsJSON)))
	var storeResp map[string]generator.AccountInfo

	// check that we don't have errors in the response
	checkMultiAccountResponse(t, rawResp, &storeResp)
	addresses := make([]string, 0)
	for _, info := range storeResp {
		addresses = append(addresses, info.Address)
	}

	if len(addresses) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(addresses))
		return nil, false
	}

	// for each stored account, check that we can decrypt it with the password we used.
	dir := statusBackend.StatusNode().Config().DataDir
	for _, address := range addresses {
		_, err = statusBackend.AccountManager().VerifyAccountPassword(dir, address, password)
		if err != nil {
			t.Errorf("failed to verify password on stored derived account")
			return nil, false
		}
	}

	return addresses, true
}

func testMultiAccountGenerateAndDerive(t *testing.T) bool { //nolint: gocyclo
	// to make sure that we start with empty account (which might have gotten populated during previous tests)
	if err := statusBackend.Logout(); err != nil {
		t.Fatal(err)
	}

	paths := []string{"m/0", "m/1"}
	params := mobile.MultiAccountGenerateAndDeriveAddressesParams{
		MultiAccountGenerateParams: mobile.MultiAccountGenerateParams{
			N:                    2,
			MnemonicPhraseLength: 12,
		},
		Paths: paths,
	}

	paramsJSON, err := json.Marshal(&params)
	if err != nil {
		t.Errorf("error encoding MultiAccountGenerateAndDeriveParams")
		return false
	}

	// generate 2 random accounts and derive 2 accounts from each one.
	rawResp := MultiAccountGenerateAndDeriveAddresses(C.CString(string(paramsJSON)))
	var generateResp []generator.GeneratedAndDerivedAccountInfo
	// check there's no error in the response
	checkMultiAccountResponse(t, rawResp, &generateResp)
	if len(generateResp) != 2 {
		t.Errorf("expected 2 accounts created, got %d", len(generateResp))
		return false
	}

	// check that for each account we have the 2 derived addresses
	for _, info := range generateResp {
		for _, path := range paths {
			if _, ok := info.Derived[path]; !ok {
				t.Errorf("results doesn't contain account info for path %s", path)
				return false
			}
		}
	}

	return true
}

func testMultiAccountImportStore(t *testing.T) bool { //nolint: gocyclo
	// to make sure that we start with empty account (which might have gotten populated during previous tests)
	if err := statusBackend.Logout(); err != nil {
		t.Fatal(err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("failed generating key")
	}

	hex := fmt.Sprintf("%#x", crypto.FromECDSA(key))
	importParams := mobile.MultiAccountImportPrivateKeyParams{
		PrivateKey: hex,
	}

	paramsJSON, err := json.Marshal(&importParams)
	if err != nil {
		t.Errorf("error encoding MultiAccountImportPrivateKeyParams")
		return false
	}

	// import raw private key
	rawResp := MultiAccountImportPrivateKey(C.CString(string(paramsJSON)))
	var importResp generator.IdentifiedAccountInfo
	// check the response doesn't have errors
	checkMultiAccountResponse(t, rawResp, &importResp)

	// prepare StoreAccount params
	password := "test-multiaccount-imported-key-password"
	storeParams := mobile.MultiAccountStoreAccountParams{
		AccountID: importResp.ID,
		Password:  password,
	}

	paramsJSON, err = json.Marshal(storeParams)
	if err != nil {
		t.Errorf("error encoding MultiAccountStoreParams")
		return false
	}

	// store the imported private key
	rawResp = MultiAccountStoreAccount(C.CString(string(paramsJSON)))
	var storeResp generator.AccountInfo
	// check the response doesn't have errors
	checkMultiAccountResponse(t, rawResp, &storeResp)

	dir := statusBackend.StatusNode().Config().DataDir
	_, err = statusBackend.AccountManager().VerifyAccountPassword(dir, storeResp.Address, password)
	if err != nil {
		t.Errorf("failed to verify password on stored derived account")
	}

	return true
}

func testMultiAccountLoadAccount(t *testing.T, address string, password string) (string, bool) { //nolint: gocyclo
	t.Log("loading account")
	params := mobile.MultiAccountLoadAccountParams{
		Address:  address,
		Password: password,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Errorf("error encoding MultiAccountLoadAccountParams")
		return "", false
	}

	// load the account in memory
	rawResp := MultiAccountLoadAccount(C.CString(string(paramsJSON)))
	var loadResp generator.IdentifiedAccountInfo

	// check that we don't have errors in the response
	checkMultiAccountResponse(t, rawResp, &loadResp)

	return loadResp.ID, true
}
