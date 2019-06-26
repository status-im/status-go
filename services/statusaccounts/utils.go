package statusaccounts

import (
	"errors"

	"github.com/status-im/status-go/extkeys"
)

// ErrInvalidMnemonicPhraseLength is returned if the requested mnemonic length is invalid.
// Valid lengths are 12, 15, 18, 21, and 24.
var ErrInvalidMnemonicPhraseLength = errors.New("mnemonic phrase length; valid lengths are 12, 15, 18, 21, and 24")

func mnemonicPhraseLengthToEntropyStrenght(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, errInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
