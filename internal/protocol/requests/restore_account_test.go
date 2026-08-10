package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const validWhisperPrivateKey = "0x1111111111111111111111111111111111111111111111111111111111111111"

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
	req.Keycard = &KeycardData{
		KeyUID:            "0x1",
		WhisperPrivateKey: validWhisperPrivateKey,
	}

	require.NoError(t, req.Validate(true), "a keycard restore with card details and a whisper key is the supported happy path")
}
