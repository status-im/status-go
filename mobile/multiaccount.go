package statusgo

import (
	"encoding/json"
	"strings"

	"github.com/status-im/status-go/internal/accounts-management/common"
	generator2 "github.com/status-im/status-go/internal/accounts-management/generator"
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

	generatedAccount, err := generator2.CreateAccountFromPrivateKey(p.PrivateKey)
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

	generatedAccount, err := generator2.CreateAccountFromMnemonic(mnemonicPhraseNoExtraSpaces, p.Bip39Passphrase)
	if err != nil {
		return makeJSONResponse(err)
	}

	derivedAccounts, err := generator2.DeriveChildrenFromAccount(generatedAccount, p.Paths)
	if err != nil {
		return makeJSONResponse(err)
	}

	generatedAccountInfo := generatedAccount.ToIdentifiedAccountInfo()

	derivedAccountsInfo := make(map[string]generator2.AccountInfo)
	for path, derivedAccount := range derivedAccounts {
		derivedAccountsInfo[path] = derivedAccount.ToAccountInfo()
	}

	generatedAndDerivedAccountsInfo := generator2.GeneratedAndDerivedAccountInfo{
		GeneratedAccountInfo: generatedAccountInfo.ToGeneratedAccountInfo(mnemonicPhraseNoExtraSpaces),
		Derived:              derivedAccountsInfo,
	}

	out, err := json.Marshal(generatedAndDerivedAccountsInfo)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(out)
}

// ConvertURCryptoHDKeyToXPub converts a UR:CRYPTO-HDKEY encoded string to a standard BIP32 xpub string.
// The resulting xpub can be passed to DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths.
func ConvertURCryptoHDKeyToXPub(ur string) string {
	xpub, err := common.URCryptoHDKeyToXPub(ur)
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return prepareJSONResponse(xpub, nil)
}

// DeriveExtendedPublicKeyAtPath derives an extended public key (xpub) at the given path.
func DeriveExtendedPublicKeyAtPath(paramsJSON string) string {
	type Params struct {
		Mnemonic   string `json:"mnemonic"`
		Passphrase string `json:"passphrase"`
		Path       string `json:"path"`
	}

	var p Params
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return prepareJSONResponse(nil, err)
	}

	xpub, err := generator2.DeriveExtendedPublicKeyAtPath(p.Mnemonic, p.Passphrase, p.Path)
	if err != nil {
		return prepareJSONResponse(nil, err)
	}
	return prepareJSONResponse(xpub, nil)
}

// DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths derives accounts public info from an extended public key (xpub)
// for the given derivation paths.Paths are relative to the provided xpub.
func DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths(paramsJSON string) string {
	type Params struct {
		ExtendedPublicKey string   `json:"extendedPublicKey"`
		Paths             []string `json:"paths"`
	}

	var p Params
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return prepareJSONResponse(nil, err)
	}

	result, err := generator2.DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths(p.ExtendedPublicKey, p.Paths)
	if err != nil {
		return prepareJSONResponse(nil, err)
	}

	return prepareJSONResponse(result, nil)
}
