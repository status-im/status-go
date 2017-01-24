package geth_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/geth"
)

func TestAccountsList(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	les, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("expected LES service: %v", err)
	}
	accounts := les.StatusBackend.AccountManager().Accounts()
	geth.Logout()

	// make sure that we start with empty accounts list (nobody has logged in yet)
	if len(accounts) != 0 {
		t.Error("accounts returned, while there should be none (we haven't logged in yet)")
		return
	}

	// create an account
	address, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	// ensure that there is still no accounts returned
	accounts = les.StatusBackend.AccountManager().Accounts()
	if len(accounts) != 0 {
		t.Error("accounts returned, while there should be none (we haven't logged in yet)")
		return
	}

	// select account (sub-accounts will be created for this key)
	err = geth.SelectAccount(address, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	// at this point main account should show up
	accounts = les.StatusBackend.AccountManager().Accounts()
	if len(accounts) != 1 {
		t.Error("exactly single account is expected (main account)")
		return
	}
	if string(accounts[0].Address.Hex()) != "0x"+address {
		t.Errorf("main account is not retured as the first key: got %s, expected %s",
			accounts[0].Address.Hex(), "0x"+address)
		return
	}

	// create sub-account 1
	subAccount1, subPubKey1, err := geth.CreateChildAccount("", newAccountPassword)
	if err != nil {
		t.Errorf("cannot create sub-account: %v", err)
		return
	}

	// now we expect to see both main account and sub-account 1
	accounts = les.StatusBackend.AccountManager().Accounts()
	if len(accounts) != 2 {
		t.Error("exactly 2 accounts are expected (main + sub-account 1)")
		return
	}
	if string(accounts[0].Address.Hex()) != "0x"+address {
		t.Errorf("main account is not retured as the first key: got %s, expected %s",
			accounts[0].Address.Hex(), "0x"+address)
		return
	}
	if string(accounts[1].Address.Hex()) != "0x"+subAccount1 {
		t.Errorf("subAcount1 not returned: got %s, expected %s", accounts[1].Address.Hex(), "0x"+subAccount1)
		return
	}

	// create sub-account 2, index automatically progresses
	subAccount2, subPubKey2, err := geth.CreateChildAccount("", newAccountPassword)
	if err != nil {
		t.Errorf("cannot create sub-account: %v", err)
	}
	if subAccount1 == subAccount2 || subPubKey1 == subPubKey2 {
		t.Error("sub-account index auto-increament failed")
		return
	}

	// finally, all 3 accounts should show up (main account, sub-accounts 1 and 2)
	accounts = les.StatusBackend.AccountManager().Accounts()
	if len(accounts) != 3 {
		t.Errorf("unexpected number of accounts: expected %d, got %d", 3, len(accounts))
		return
	}
	if string(accounts[0].Address.Hex()) != "0x"+address {
		t.Errorf("main account is not retured as the first key: got %s, expected %s",
			accounts[0].Address.Hex(), "0x"+address)
		return
	}
	subAccount1MatchesKey1 := string(accounts[1].Address.Hex()) != "0x"+subAccount1
	subAccount1MatchesKey2 := string(accounts[2].Address.Hex()) != "0x"+subAccount1
	if !subAccount1MatchesKey1 && !subAccount1MatchesKey2 {
		t.Errorf("subAcount1 not returned: got %s, expected %s", accounts[1].Address.Hex(), "0x"+subAccount1)
		return
	}
	subAccount2MatchesKey1 := string(accounts[1].Address.Hex()) != "0x"+subAccount2
	subAccount2MatchesKey2 := string(accounts[2].Address.Hex()) != "0x"+subAccount2
	if !subAccount2MatchesKey1 && !subAccount2MatchesKey2 {
		t.Errorf("subAcount2 not returned: got %s, expected %s", accounts[2].Address.Hex(), "0x"+subAccount1)
		return
	}
}

func TestCreateChildAccount(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	geth.Logout() // to make sure that we start with empty account (which might get populated during previous tests)

	accountManager, err := geth.NodeManagerInstance().AccountManager()
	if err != nil {
		t.Error(err)
		return
	}

	// create an account
	address, pubKey, mnemonic, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	account, err := geth.ParseAccountString(accountManager, address)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
		return
	}

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	account, key, err := accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return
	}

	if key.ExtendedKey == nil {
		t.Error("CKD#2 has not been generated for new account")
		return
	}

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	_, _, err = geth.CreateChildAccount("", newAccountPassword)
	if !reflect.DeepEqual(err, geth.ErrNoAccountSelected) {
		t.Errorf("expected error is not returned (tried to create sub-account w/o login): %v", err)
		return
	}

	err = geth.SelectAccount(address, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}

	// try to create sub-account with wrong password
	_, _, err = geth.CreateChildAccount("", "wrong password")
	if !reflect.DeepEqual(err, errors.New("cannot retreive a valid key for a given account: could not decrypt key with given passphrase")) {
		t.Errorf("expected error is not returned (tried to create sub-account with wrong password): %v", err)
		return
	}

	// create sub-account (from implicit parent)
	subAccount1, subPubKey1, err := geth.CreateChildAccount("", newAccountPassword)
	if err != nil {
		t.Errorf("cannot create sub-account: %v", err)
		return
	}

	// make sure that sub-account index automatically progresses
	subAccount2, subPubKey2, err := geth.CreateChildAccount("", newAccountPassword)
	if err != nil {
		t.Errorf("cannot create sub-account: %v", err)
	}
	if subAccount1 == subAccount2 || subPubKey1 == subPubKey2 {
		t.Error("sub-account index auto-increament failed")
		return
	}

	// create sub-account (from explicit parent)
	subAccount3, subPubKey3, err := geth.CreateChildAccount(subAccount2, newAccountPassword)
	if err != nil {
		t.Errorf("cannot create sub-account: %v", err)
	}
	if subAccount1 == subAccount3 || subPubKey1 == subPubKey3 || subAccount2 == subAccount3 || subPubKey2 == subPubKey3 {
		t.Error("sub-account index auto-increament failed")
		return
	}
}

func TestRecoverAccount(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	accountManager, _ := geth.NodeManagerInstance().AccountManager()

	// create an account
	address, pubKey, mnemonic, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try recovering using password + mnemonic
	addressCheck, pubKeyCheck, err := geth.RecoverAccount(newAccountPassword, mnemonic)
	if err != nil {
		t.Errorf("recover account failed: %v", err)
		return
	}
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details")
	}

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := geth.ParseAccountString(accountManager, address)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
	}

	account, key, err := accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return
	}
	extChild2String := key.ExtendedKey.String()

	if err := accountManager.DeleteAccount(account, newAccountPassword); err != nil {
		t.Errorf("cannot remove account: %v", err)
	}

	addressCheck, pubKeyCheck, err = geth.RecoverAccount(newAccountPassword, mnemonic)
	if err != nil {
		t.Errorf("recover account failed (for non-cached account): %v", err)
		return
	}
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
	}

	// make sure that extended key exists and is imported ok too
	account, key, err = accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return
	}
	if extChild2String != key.ExtendedKey.String() {
		t.Errorf("CKD#2 key mismatch, expected: %s, got: %s", extChild2String, key.ExtendedKey.String())
	}

	// make sure that calling import several times, just returns from cache (no error is expected)
	addressCheck, pubKeyCheck, err = geth.RecoverAccount(newAccountPassword, mnemonic)
	if err != nil {
		t.Errorf("recover account failed (for non-cached account): %v", err)
		return
	}
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
	}

	// time to login with recovered data
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// make sure that identity is not (yet injected)
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKeyCheck))) {
		t.Error("identity already present in whisper")
	}
	err = geth.SelectAccount(addressCheck, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKeyCheck))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
}

func TestAccountSelect(t *testing.T) {

	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// test to see if the account was injected in whisper
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	address1, pubKey1, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Error("Test failed: could not create account")
		return
	}
	t.Logf("Account created: {address: %s, key: %s}", address2, pubKey2)

	// make sure that identity is not (yet injected)
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity already present in whisper")
	}

	// try selecting with wrong password
	err = geth.SelectAccount(address1, "wrongPassword")
	if err == nil {
		t.Error("select account is expected to throw error: wrong password used")
		return
	}
	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Error("identity already present in whisper")
	}
	err = geth.SelectAccount(address2, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity should be removed, but it is still present in whisper")
	}
}

func TestAccountLogout(t *testing.T) {

	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	address, pubKey, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	// make sure that identity doesn't exist (yet) in Whisper
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity already present in whisper")
	}

	// select/login
	err = geth.SelectAccount(address, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity not injected into whisper")
	}

	err = geth.Logout()
	if err != nil {
		t.Errorf("cannot logout: %v", err)
	}

	// now, logout and check if identity is removed indeed
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity not cleared from whisper")
	}
}

func TestSelectedAccountOnNodeRestart(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// we need to make sure that selected account is injected as identity into Whisper
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create test accounts
	address1, pubKey1, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("account1 created: {address: %s, key: %s}", address1, pubKey1)
	address2, pubKey2, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("account2 created: {address: %s, key: %s}", address2, pubKey2)

	// make sure that identity is not (yet injected)
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity already present in whisper")
	}

	// make sure that no account is selected by default
	if geth.NodeManagerInstance().SelectedAccount != nil {
		t.Error("account selected, but should not be")
		return
	}

	// select account
	err = geth.SelectAccount(address1, "wrongPassword")
	if err == nil {
		t.Error("select account is expected to throw error: wrong password used")
		return
	}
	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Error("identity already present in whisper")
	}
	err = geth.SelectAccount(address2, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity should be removed, but it is still present in whisper")
	}

	// stop node (and all of its sub-protocols)
	if err := geth.NodeManagerInstance().StopNode(); err != nil {
		t.Error(err)
		return
	}

	// make sure that account is still selected
	if geth.NodeManagerInstance().SelectedAccount == nil {
		t.Error("no selected account")
		return
	}
	if geth.NodeManagerInstance().SelectedAccount.Address.Hex() != "0x"+address2 {
		t.Error("incorrect address selected")
		return
	}

	// resume node
	if err := geth.NodeManagerInstance().ResumeNode(); err != nil {
		t.Error(err)
		return
	}

	// re-check selected account (account2 MUST be selected)
	if geth.NodeManagerInstance().SelectedAccount == nil {
		t.Error("no selected account")
		return
	}
	if geth.NodeManagerInstance().SelectedAccount.Address.Hex() != "0x"+address2 {
		t.Error("incorrect address selected")
		return
	}

	// make sure that Whisper gets identity re-injected
	whisperService, err = geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity should not be present, but it is still present in whisper")
	}
}

func TestNodeRestartWithNoSelectedAccount(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	geth.Logout()

	// we need to make sure that selected account is injected as identity into Whisper
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create test accounts
	address1, pubKey1, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	t.Logf("account1 created: {address: %s, key: %s}", address1, pubKey1)

	// make sure that identity is not present
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity already present in whisper")
	}

	// make sure that no account is selected
	if geth.NodeManagerInstance().SelectedAccount != nil {
		t.Error("account selected, but should not be")
		return
	}

	// stop node (and all of its sub-protocols)
	if err := geth.NodeManagerInstance().StopNode(); err != nil {
		t.Error(err)
		return
	}

	// make sure that no account is selected
	if geth.NodeManagerInstance().SelectedAccount != nil {
		t.Error("account selected, but should not be")
		return
	}

	// resume node
	if err := geth.NodeManagerInstance().ResumeNode(); err != nil {
		t.Error(err)
		return
	}

	// make sure that no account is selected
	if geth.NodeManagerInstance().SelectedAccount != nil {
		t.Error("account selected, but should not be")
		return
	}

	// make sure that Whisper gets identity re-injected
	whisperService, err = geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity should not be present, but it is present in whisper")
	}
}
