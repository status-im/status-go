package statusaccounts

import (
	"errors"

	"github.com/status-im/status-go/extkeys"
)

var errInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length")

func mnemonicPhraseLengthToEntropyStrenght(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, errInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
