package main

import (
	"fmt"
	"path/filepath"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
)

// LoadTestAccounts loads public key files for test accounts
func LoadTestAccounts(dataDir string) error {
	files := []string{"test-account1.pk", "test-account2.pk"}
	dir := filepath.Join(dataDir, "keystore")
	for _, filename := range files {
		if err := common.ImportTestAccount(dir, filename); err != nil {
			return err
		}
	}
	return nil
}

// InjectTestAccounts injects test accounts into running node
func InjectTestAccounts(node common.NodeManager) error {
	testConfig, err := common.LoadTestConfig()
	if err != nil {
		return err
	}

	if err = injectAccountIntoWhisper(node, testConfig.Account1.Address,
		testConfig.Account1.Password); err != nil {
		return err
	}
	if err = injectAccountIntoWhisper(node, testConfig.Account2.Address,
		testConfig.Account2.Password); err != nil {
		return err
	}

	return nil
}

// injectAccountIntoWhisper adds key pair into Whisper. Similar to Select/Login,
// but allows multiple accounts to be injected.
func injectAccountIntoWhisper(node common.NodeManager, address, password string) error {
	keyStore, err := node.AccountKeyStore()
	if err != nil {
		return err
	}

	acct, err := common.ParseAccountString(address)
	if err != nil {
		return account.ErrAddressToAccountMappingFailure
	}

	_, accountKey, err := keyStore.AccountDecryptedKey(acct, password)
	if err != nil {
		return fmt.Errorf("%s: %v", account.ErrAccountToKeyMappingFailure.Error(), err)
	}

	whisperService, err := node.WhisperService()
	if err != nil {
		return err
	}
	if _, err = whisperService.AddKeyPair(accountKey.PrivateKey); err != nil {
		return err
	}

	return nil
}
