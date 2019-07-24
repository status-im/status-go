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

func checkMultiAccountResponse(t *testing.T, respJSON *C.char, resp interface{}) {
	var e struct {
		Error *string `json:"error"`
	}

	json.Unmarshal([]byte(C.GoString(respJSON)), &e)
	if e.Error != nil {
		t.Errorf("unexpected response error: %s", *e.Error)
	}

	if err := json.Unmarshal([]byte(C.GoString(respJSON)), resp); err != nil {
		t.Fatalf("error unmarshaling response to expected struct: %s", err)
	}
}

func testMultiAccountGenerateDeriveAndStore(t *testing.T) bool { //nolint: gocyclo
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

		if ok := testMultiAccountDeriveAddresses(t, info.ID, paths); !ok {
			return false
		}
	}

	// store 2 derived child accounts from the first account.
	// after that all the generated account should be remove from memory.
	if ok := testMultiAccountStoreDerived(t, generateResp[0].ID, paths); !ok {
		return false
	}

	return true
}

func testMultiAccountDeriveAddresses(t *testing.T, accountID string, paths []string) bool { //nolint: gocyclo
	params := mobile.MultiAccountDeriveAddressesParams{
		AccountID: accountID,
		Paths:     paths,
	}

	paramsJSON, err := json.Marshal(&params)
	if err != nil {
		t.Errorf("error encoding MultiAccountDeriveAddressesParams")
		return false
	}

	// derive addresses from account accountID
	rawResp := MultiAccountDeriveAddresses(C.CString(string(paramsJSON)))
	var deriveResp map[string]generator.AccountInfo
	// check the response doesn't have errors
	checkMultiAccountResponse(t, rawResp, &deriveResp)
	if len(deriveResp) != 2 {
		t.Errorf("expected 2 derived accounts info, got %d", len(deriveResp))
		return false
	}

	// check that we have an address for each derivation path we used.
	for _, path := range paths {
		if _, ok := deriveResp[path]; !ok {
			t.Errorf("results doesn't contain account info for path %s", path)
			return false
		}
	}

	return true
}

func testMultiAccountStoreDerived(t *testing.T, accountID string, paths []string) bool { //nolint: gocyclo
	password := "test-multiaccount-password"

	params := mobile.MultiAccountStoreDerivedParams{
		MultiAccountDeriveAddressesParams: mobile.MultiAccountDeriveAddressesParams{
			AccountID: accountID,
			Paths:     paths,
		},
		Password: password,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Errorf("error encoding MultiAccountStoreDerivedParams")
		return false
	}

	// store one child account for each derivation path.
	rawResp := MultiAccountStoreDerived(C.CString(string(paramsJSON)))
	var storeResp map[string]generator.AccountInfo

	// check that we don't have errors in the response
	checkMultiAccountResponse(t, rawResp, &storeResp)
	addresses := make([]string, 0)
	for _, info := range storeResp {
		addresses = append(addresses, info.Address)
	}

	if len(addresses) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(addresses))
		return false
	}

	// for each stored account, check that we can decrypt it with the password we used.
	dir := statusBackend.StatusNode().Config().DataDir
	for _, address := range addresses {
		_, err = statusBackend.AccountManager().VerifyAccountPassword(dir, address, password)
		if err != nil {
			t.Errorf("failed to verify password on stored derived account")
		}
	}

	return true
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
