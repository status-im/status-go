package account

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
