package account

import (
	"strings"
	"testing"

	"github.com/status-im/status-go/extkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMnemonicPhraseLengthToEntropyStrenght(t *testing.T) {
	scenarios := []struct {
		phraseLength     int
		expectedStrength extkeys.EntropyStrength
		expectedError    error
	}{
		{12, 128, nil},
		{15, 160, nil},
		{18, 192, nil},
		{21, 224, nil},
		{24, 256, nil},
		// invalid
		{11, 0, errInvalidMnemonicPhraseLength},
		{14, 0, errInvalidMnemonicPhraseLength},
		{25, 0, errInvalidMnemonicPhraseLength},
	}

	for _, s := range scenarios {
		strength, err := mnemonicPhraseLengthToEntropyStrenght(s.phraseLength)
		assert.Equal(t, s.expectedError, err)
		assert.Equal(t, s.expectedStrength, strength)
	}
}

func TestOnboarding(t *testing.T) {
	count := 2
	wordsCount := 24
	o, _ := NewOnboarding(count, wordsCount)
	assert.Equal(t, count, len(o.accounts))

	for id, a := range o.accounts {
		words := strings.Split(a.mnemonic, " ")

		assert.Equal(t, wordsCount, len(words))
		assert.NotEmpty(t, a.Info.WalletAddress)
		assert.NotEmpty(t, a.Info.WalletPubKey)
		assert.NotEmpty(t, a.Info.ChatAddress)
		assert.NotEmpty(t, a.Info.ChatPubKey)

		retrieved, err := o.Account(id)
		require.NoError(t, err)
		assert.Equal(t, a, retrieved)
	}
}
