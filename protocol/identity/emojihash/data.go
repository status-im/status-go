package emojihash

import (
	"bufio"
	"bytes"
	_ "embed"
	"strings"
	"sync"
)

var (
	//go:embed emojis.txt
	emojisData []byte

	// once ensures that emojis alphabet is loaded only once
	once sync.Once

	// emojisAlphabet is a slice of all emojis in emojisData
	emojisAlphabet []string
)

// loadAlphabet lazy-loads emojis alphabet from emojis.txt file
func loadAlphabet() {
	once.Do(func() {
		alphabet := make([]string, 0, emojiAlphabetLen)

		scanner := bufio.NewScanner(bytes.NewReader(emojisData))
		for scanner.Scan() {
			alphabet = append(alphabet, strings.Replace(scanner.Text(), "\n", "", -1))
		}

		// current alphabet contains more emojis than needed, just in case some emojis needs to be removed
		// make sure only necessary part is loaded
		if len(alphabet) > emojiAlphabetLen {
			alphabet = alphabet[:emojiAlphabetLen]
		}

		emojisAlphabet = alphabet
	})
}
