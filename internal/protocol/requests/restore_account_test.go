package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const validWhisperPrivateKey = "0x1111111111111111111111111111111111111111111111111111111111111111"

// keycardDataFromCard mirrors the shape the client sends after reading a card:
// every field the backend maps into a derived address is present. The values are
// the same card fixture used by TestRestoreKeycardAccountAndLogin.
func keycardDataFromCard() *KeycardData {
	return &KeycardData{
		KeyUID:              "0x579324c53f347e18961c775a00ec13ed7d59a225b1859d5125ff36b450b8778d",
		Address:             "0xbf9dE86774051537b2192Ce9c8d2496f129bA24b",
		WhisperPrivateKey:   "5a42b4f15ff1a5da95d116442ce11a31e9020f562224bf60b1d8d3a99d90653d",
		WhisperPublicKey:    "0x0441468c39b579259676350b9736b01cdadb740f67bfd022fa2b985123b1d66fc3191cfe73205e3d3d84148f0248f9a2978afeeda16d7c3db90bd2579f0de33459",
		WhisperAddress:      "0xBa122B9c0Ef560813b5D2C0961094aC36289f846",
		WalletPublicKey:     "0x04c16e7748f34e0ab2c9c13350d7872d928e942934dd8b8abd3fb12b8c742a5ee8cf0919731e800907068afec25f577bde3a9c534795e359ee48097e4e55f4aaca",
		WalletAddress:       "0xB9E1998e1A8854887CA327D1aF5894B6CB0AC07D",
		WalletRootAddress:   "0xFf59db9F2f97Db7104A906C390D33C342a1309C8",
		Eip1581Address:      "0xA8d50f0B3bc581298446be8FBfF5c71684Ea6c01",
		EncryptionPublicKey: "0x04c4b16f670b51702dc130673bf9c64ffd1f69383cef2127dfa05031b9b1359120f7342134af9a350465126a85e87cb003b7c4f93d2ba2ff98bb73277b119c7a87",
	}
}

func validRestoreAccount() RestoreAccount {
	return RestoreAccount{
		CreateAccount: CreateAccount{
			Password:           "test-password",
			CustomizationColor: "primary",
			RootDataDir:        "/tmp/status-test",
		},
	}
}

func TestLoginValidateRejectsEmptyKeyUID(t *testing.T) {
	login := Login{}

	err := login.Validate()
	require.ErrorIs(t, err, ErrLoginInvalidKeyUID, "a login without a key-uid cannot resolve an account and must be rejected")
}

func TestLoginValidateRejectsUnparseableKeycardWhisperPrivateKey(t *testing.T) {
	login := Login{
		KeyUID:                   "0x1",
		KeycardWhisperPrivateKey: "not-a-private-key",
	}

	err := login.Validate()
	require.ErrorIs(t, err, ErrLoginInvalidKeycardWhisperPrivateKey, "an unparseable keycard whisper key must be rejected at validation because ChatPrivateKey ignores the parse error later")
}

func TestLoginValidateAcceptsParseableKeycardWhisperPrivateKey(t *testing.T) {
	login := Login{
		KeyUID:                   "0x1",
		KeycardWhisperPrivateKey: validWhisperPrivateKey,
	}

	require.NoError(t, login.Validate(), "a well-formed keycard whisper key must pass validation")
	require.NotNil(t, login.ChatPrivateKey(), "a validated keycard login must yield a chat identity")
}

func TestLoginValidateAllowsMissingKeycardWhisperPrivateKey(t *testing.T) {
	login := Login{KeyUID: "0x1"}

	require.NoError(t, login.Validate(), "a regular password login carries no keycard whisper key and must validate")
	require.Nil(t, login.ChatPrivateKey(), "without a keycard whisper key there is no derived chat key")
}

func TestRestoreAccountValidateRejectsMnemonicForKeycardRestore(t *testing.T) {
	req := validRestoreAccount()
	req.Mnemonic = "some mnemonic words"
	req.Keycard = &KeycardData{WhisperPrivateKey: validWhisperPrivateKey}

	err := req.Validate(true)
	require.ErrorIs(t, err, ErrRestoreKeycardAccountMnemonicSet, "keycard restore derives keys from the card, a mnemonic alongside is contradictory input")
}

func TestRestoreAccountValidateRejectsMissingKeycardDataForKeycardRestore(t *testing.T) {
	req := validRestoreAccount()

	err := req.Validate(true)
	require.ErrorIs(t, err, ErrRestoreKeycardAccountKecardDetatilsMissing, "keycard restore without keycard data would proceed with nil card details")
}

func TestRestoreAccountValidateRejectsKeycardDataWithoutWhisperPrivateKey(t *testing.T) {
	req := validRestoreAccount()
	req.Keycard = &KeycardData{KeyUID: "0x1"}

	err := req.Validate(true)
	require.ErrorIs(t, err, ErrRestoreKeycardAccountNoWhisperPrivateKey, "keycard restore without the whisper private key would leave the account with no chat identity")
}

func TestRestoreAccountValidateRejectsKeycardDataForRegularRestore(t *testing.T) {
	req := validRestoreAccount()
	req.Mnemonic = "some mnemonic words"
	req.Keycard = &KeycardData{WhisperPrivateKey: validWhisperPrivateKey}

	err := req.Validate(false)
	require.ErrorIs(t, err, ErrRestoreRegularAccountKeycardSet, "a regular restore must not silently carry keycard details")
}

func TestRestoreAccountValidateRejectsMissingMnemonicForRegularRestore(t *testing.T) {
	req := validRestoreAccount()

	err := req.Validate(false)
	require.ErrorIs(t, err, ErrRestoreRegularAccountMnemonicMissing, "a regular restore has nothing to restore from without a mnemonic")
}

func TestRestoreAccountValidateAcceptsCompleteKeycardRestore(t *testing.T) {
	req := validRestoreAccount()
	req.Keycard = keycardDataFromCard()

	require.NoError(t, req.Validate(true), "a keycard restore carrying the full card details is the supported happy path")
}

// Validate only checks which fields are present, so a malformed key reaches the
// backend and fails later at key generation. Login.Validate parses its keycard
// whisper key; RestoreAccount.Validate does not.
func TestRestoreAccountValidateAcceptsUnparseableWhisperPrivateKey(t *testing.T) {
	req := validRestoreAccount()
	req.Keycard = keycardDataFromCard()
	req.Keycard.WhisperPrivateKey = "not-a-key"

	require.NoError(t, req.Validate(true), "documents that restore does not parse the whisper key, unlike login")
}
