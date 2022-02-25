package visualidentity

import (
	"errors"
	"math/big"
)

func ToBigBase(value *big.Int, base uint64) (res [](uint64)) {
	toBigBaseImpl(value, base, &res)
	return
}

func toBigBaseImpl(value *big.Int, base uint64, res *[](uint64)) {
	bigBase := new(big.Int).SetUint64(base)
	quotient := new(big.Int).Div(value, bigBase)
	if quotient.Cmp(new(big.Int).SetUint64(0)) != 0 {
		toBigBaseImpl(quotient, base, res)
	}

	*res = append(*res, new(big.Int).Mod(value, bigBase).Uint64())
}

func ToEmojiHash(value *big.Int, hashLen int, alphabet *[]string) (hash []string, err error) {
	valueBitLen := value.BitLen()
	alphabetLen := new(big.Int).SetInt64(int64(len(*alphabet)))

	indexes := ToBigBase(value, alphabetLen.Uint64())
	if hashLen == 0 {
		hashLen = len(indexes)
	} else if hashLen > len(indexes) {
		prependLen := hashLen - len(indexes)
		for i := 0; i < prependLen; i++ {
			indexes = append([](uint64){0}, indexes...)
		}
	}

	// alphabetLen^hashLen
	possibleCombinations := new(big.Int).Exp(alphabetLen, new(big.Int).SetInt64(int64(hashLen)), nil)

	// 2^valueBitLen
	requiredCombinations := new(big.Int).Exp(new(big.Int).SetInt64(2), new(big.Int).SetInt64(int64(valueBitLen)), nil)

	if possibleCombinations.Cmp(requiredCombinations) == -1 {
		return nil, errors.New("alphabet or hash length is too short to encode given value")
	}

	for _, v := range indexes {
		hash = append(hash, (*alphabet)[v])
	}

	return hash, nil
}

// compressedPubKey = |1.5 bytes chars cutoff|20 bytes emoji hash|10 bytes color hash|1.5 bytes chars cutoff|
func Slices(compressedPubkey []byte) (res [4][]byte, err error) {
	if len(compressedPubkey) != 33 {
		return res, errors.New("incorrect compressed pubkey")
	}

	getSlice := func(low, high int, and string, rsh uint) []byte {
		sliceValue := new(big.Int).SetBytes(compressedPubkey[low:high])
		andValue, _ := new(big.Int).SetString(and, 0)
		andRes := new(big.Int).And(sliceValue, andValue)
		return new(big.Int).Rsh(andRes, rsh).Bytes()
	}

	res[0] = getSlice(0, 2, "0xFFF0", 4)
	res[1] = getSlice(1, 22, "0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0", 4)
	res[2] = getSlice(21, 32, "0x0FFFFFFFFFFFFFFFFFFFF0", 4)
	res[3] = getSlice(31, 33, "0x0FFF", 0)

	return res, nil
}

// [[1 0] [1 1] [1 2] ... [units, colors-1]]
// [3 12] => 3 units length, 12 color index
func MakeColorHashAlphabet(units, colors int) (res [][]int) {
	res = make([][]int, units*colors)
	idx := 0
	for i := 0; i < units; i++ {
		for j := 0; j < colors; j++ {
			res[idx] = make([]int, 2)
			res[idx][0] = i + 1
			res[idx][1] = j
			idx++
		}
	}
	return
}

func ToColorHash(value *big.Int, alphabet *[][]int, colorsCount int) (hash [][]int) {
	alphabetLen := len(*alphabet)
	indexes := ToBigBase(value, uint64(alphabetLen))
	hash = make([][](int), len(indexes))
	for i, v := range indexes {
		hash[i] = make([](int), 2)
		hash[i][0] = (*alphabet)[v][0]
		hash[i][1] = (*alphabet)[v][1]
	}

	// colors can't repeat themselves
	// this makes color hash not fully collision resistant
	prevColorIdx := hash[0][1]
	hashLen := len(hash)
	for i := 1; i < hashLen; i++ {
		colorIdx := hash[i][1]
		if colorIdx == prevColorIdx {
			hash[i][1] = (colorIdx + 1) % colorsCount
		}
		prevColorIdx = hash[i][1]
	}

	return
}
