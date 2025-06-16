package personal

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/crypto"
	ethtypes "github.com/status-im/status-go/eth-node/types"

	"github.com/stretchr/testify/require"
)

func generateMessageToSign(t *testing.T) string {
	requestID := "0x240235eb861041795bc84ebe837569f08f8306a23f3f377a2977bc18eba76326"
	requestIDBytes, err := hexutil.Decode(requestID)
	require.NoError(t, err)

	communityID := "0x02abc53deca7213ef265487b63715d226f3f31853e08636f822d74b9ebaddefb47"
	communityIDBytes, err := hexutil.Decode(communityID)
	require.NoError(t, err)

	identityPublicKeyCompressed := "0x03c8eec7e5b7ad01693376529387ae1f097c9967f161d745bf99318b321b3e7800"
	identityPublicKeyCompressedBytes, err := hexutil.Decode(identityPublicKeyCompressed)
	require.NoError(t, err)

	return ethtypes.EncodeHex(crypto.Keccak256(identityPublicKeyCompressedBytes, communityIDBytes, requestIDBytes))
}

func generateAccountForSigning(t *testing.T) *generator.Account {
	seedPhrase := "inch describe nothing prepare salon foster market fabric bottom type trial glooom"
	account, err := generator.CreateAccountFromMnemonic(seedPhrase, "")
	require.NoError(t, err)

	derivedAccount, err := generator.DeriveChildFromAccount(account, "m/44'/60'/0'/0/0")
	require.NoError(t, err)

	return derivedAccount
}

func TestSignAndRecover(t *testing.T) {
	expectedMessageToSign := "0x32a4ed227797845663dce42d5f595ab26aab43bb5bb98fca2c4599b4c255d18b"

	generatedMessageToSign := generateMessageToSign(t)
	require.Equal(t, expectedMessageToSign, generatedMessageToSign)

	accountForSigning := generateAccountForSigning(t)

	personalService := New()

	// Sign the message as string
	signature, err := personalService.Sign(SignParams{
		Data: generatedMessageToSign,
	}, accountForSigning)
	require.NoError(t, err)

	// Recover the address from the signed string message
	recoveredAddress, err := personalService.Recover(RecoverParams{
		Message:   generatedMessageToSign,
		Signature: hexutil.Encode(signature),
	})
	require.NoError(t, err)

	require.Equal(t, accountForSigning.Address().Hex(), recoveredAddress.Hex())

	// Sign the message as bytes
	generatedMessageToSignBytes, err := hexutil.Decode(generatedMessageToSign)
	require.NoError(t, err)

	signature, err = personalService.Sign(SignParams{
		Data: generatedMessageToSignBytes,
	}, accountForSigning)
	require.NoError(t, err)

	// Recover the address from the signed bytes message
	recoveredAddress, err = personalService.Recover(RecoverParams{
		Message:   generatedMessageToSign,
		Signature: hexutil.Encode(signature),
	})
	require.NoError(t, err)

	require.Equal(t, accountForSigning.Address().Hex(), recoveredAddress.Hex())
}
