package statusgo

import (
	"encoding/json"
	"strings"

	"github.com/status-im/status-go/accounts-management/generator"
)

// MultiAccountImportPrivateKeyParams are the params sent to MultiAccountImportPrivateKey.
type MultiAccountImportPrivateKeyParams struct {
	PrivateKey string `json:"privateKey"`
}

// MultiAccountImportMnemonicParams are the params sent to MultiAccountImportMnemonic.
type MultiAccountImportMnemonicParams struct {
	MnemonicPhrase  string   `json:"mnemonicPhrase"`
	Bip39Passphrase string   `json:"Bip39Passphrase"`
	Paths           []string `json:"paths"`
}

// CreateAccountFromPrivateKey returns an account derived from the private key without storing it
func CreateAccountFromPrivateKey(paramsJSON string) string {
	var p MultiAccountImportPrivateKeyParams

	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return makeJSONResponse(err)
	}

	generatedAccount, err := generator.CreateAccountFromPrivateKey(p.PrivateKey)
	if err != nil {
		return makeJSONResponse(err)
	}

	generatedAccountInfo := generatedAccount.ToIdentifiedAccountInfo()

	out, err := json.Marshal(generatedAccountInfo)
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

	generatedAccount, err := generator.CreateAccountFromMnemonic(mnemonicPhraseNoExtraSpaces, p.Bip39Passphrase)
	if err != nil {
		return makeJSONResponse(err)
	}

	derivedAccounts, err := generator.DeriveChildrenFromAccount(generatedAccount, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	generatedAccountInfo := generatedAccount.ToIdentifiedAccountInfo()

	derivedAccountsInfo := make(map[string]generator.AccountInfo)
	for path, derivedAccount := range derivedAccounts {
		derivedAccountsInfo[path] = derivedAccount.ToAccountInfo()
	}

	generatedAndDerivedAccountsInfo := generator.GeneratedAndDerivedAccountInfo{
		GeneratedAccountInfo: generatedAccountInfo.ToGeneratedAccountInfo(mnemonicPhraseNoExtraSpaces),
		Derived:              derivedAccountsInfo,
	}

	out, err := json.Marshal(generatedAndDerivedAccountsInfo)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}
