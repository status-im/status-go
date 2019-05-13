package account

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/extkeys"
	"github.com/stretchr/testify/assert"
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

func TestNew(t *testing.T) {
	o, _ := New(2, 12)
	for _, a := range o.userAccounts {
		fmt.Printf("--- %+v\n", a)
	}
}
