package visualidentity

import (
	"bufio"
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/static"
)

const (
	emojiAlphabetLen       = 2757 // 20bytes of data described by 14 emojis requires at least 2757 length alphabet
	emojiHashLen           = 14
	colorHashSegmentMaxLen = 5
	colorHashColorsCount   = 32
)

func NewAPI() *API {
	colorHashAlphabet := MakeColorHashAlphabet(colorHashSegmentMaxLen, colorHashColorsCount)
	return &API{
		emojisAlphabet:    &[]string{},
		colorHashAlphabet: &colorHashAlphabet,
	}
}

type API struct {
	emojisAlphabet    *[]string
	colorHashAlphabet *[][]int
}

func (api *API) EmojiHashOf(pubkey string) (hash []string, err error) {
	log.Info("[VisualIdentityAPI::EmojiHashOf]")

	slices, err := slices(pubkey)
	if err != nil {
		return nil, err
	}

	return ToEmojiHash(new(big.Int).SetBytes(slices[1]), emojiHashLen, api.emojisAlphabet)
}

func (api *API) ColorHashOf(pubkey string) (hash [][]int, err error) {
	log.Info("[VisualIdentityAPI::ColorHashOf]")

	slices, err := slices(pubkey)
	if err != nil {
		return nil, err
	}

	return ToColorHash(new(big.Int).SetBytes(slices[2]), api.colorHashAlphabet, colorHashColorsCount), nil
}

func LoadAlphabet() (*[]string, error) {
	data, err := static.Asset("emojis.txt")
	if err != nil {
		return nil, err
	}

	alphabet := make([]string, 0, emojiAlphabetLen)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		alphabet = append(alphabet, strings.Replace(scanner.Text(), "\n", "", -1))
	}

	// current alphabet contains more emojis than needed, just in case some emojis needs to be removed
	// make sure only necessary part is loaded
	if len(alphabet) > emojiAlphabetLen {
		alphabet = alphabet[:emojiAlphabetLen]
	}

	return &alphabet, nil
}

func slices(pubkey string) (res [4][]byte, err error) {
	pubkeyValue, ok := new(big.Int).SetString(pubkey, 0)
	if !ok {
		return res, fmt.Errorf("invalid pubkey: %s", pubkey)
	}

	x, y := secp256k1.S256().Unmarshal(pubkeyValue.Bytes())
	if x == nil || !secp256k1.S256().IsOnCurve(x, y) {
		return res, fmt.Errorf("invalid pubkey: %s", pubkey)
	}
	compressedKey := secp256k1.CompressPubkey(x, y)

	return Slices(compressedKey)
}
