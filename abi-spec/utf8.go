package abispec

import (
	"fmt"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func stringToRunes(str string) []rune {
	var runes []rune
	bytes := []byte(str)
	for len(bytes) > 0 {
		r, size := utf8.DecodeRune(bytes)
		if r == utf8.RuneError {
			for i := 0; i < size; i++ {
				runes = append(runes, rune(bytes[i]))
			}
		} else {
			runes = append(runes, r)
		}
		bytes = bytes[size:]
	}
	return runes
}

// Taken from https://mths.be/punycode
func ucs2decode(str string) []rune {
	var runes = stringToRunes(str)
	var output []rune
	var counter = 0
	var length = len(runes)
	var value rune
	var extra rune
	for counter < length {
		value = runes[counter]
		counter++
		if value >= 0xD800 && value <= 0xDBFF && counter < length {
			// high surrogate, and there is a next character
			extra = runes[counter]
			counter++
			if (extra & 0xFC00) == 0xDC00 { // low surrogate
				output = append(output, ((value&0x3FF)<<10)+(extra&0x3FF)+0x10000)
			} else {
				// unmatched surrogate; only append this code unit, in case the next
				// code unit is the high surrogate of a surrogate pair
				output = append(output, value)
				counter--
			}
		} else {
			output = append(output, value)
		}
	}
	return output
}

// Taken from https://mths.be/punycode
func ucs2encode(array []rune) []byte {
	var length = len(array)
	var index = 0
	var value rune
	var output []byte
	for index < length {
		value = array[index]
		if value > 0xFFFF {
			value -= 0x10000
			codePoint := rune(uint32(value)>>10&0x3FF | 0xD800)
			output = appendBytes(output, stringFromCharCode(codePoint))
			value = 0xDC00 | value&0x3FF
		}
		output = appendBytes(output, stringFromCharCode(value))
		index++
	}
	return output
}

func appendBytes(dest []byte, bytes []byte) []byte {
	for i := 0; i < len(bytes); i++ {
		dest = append(dest, bytes[i])
	}
	return dest
}

func checkScalarValue(codePoint rune) error {
	if codePoint >= 0xD800 && codePoint <= 0xDFFF {
		return fmt.Errorf("lone surrogate U+%s is not a scalar value", hexutil.EncodeUint64(uint64(codePoint)))
	}
	return nil
}

func stringFromCharCode(codePoint rune) []byte {
	var buf = make([]byte, 4)
	n := utf8.EncodeRune(buf, codePoint)
	return buf[0:n]
}

func createByte(codePoint rune, shift uint32) []byte {
	return stringFromCharCode(((codePoint >> shift) & 0x3F) | 0x80)
}

func encodeCodePoint(codePoint rune) ([]byte, error) {
	if (uint32(codePoint) & uint32(0xFFFFFF80)) == 0 { // 1-byte sequence
		return stringFromCharCode(codePoint), nil
	}
	var symbol []byte
	if uint32(codePoint)&0xFFFFF800 == 0 { // 2-byte sequence
		symbol = stringFromCharCode(((codePoint >> 6) & 0x1F) | 0xC0)
	} else if (uint32(codePoint) & 0xFFFF0000) == 0 { // 3-byte sequence
		err := checkScalarValue(codePoint)
		if err != nil {
			return nil, err
		}
		symbol = stringFromCharCode(((codePoint >> 12) & 0x0F) | 0xE0)
		symbol = appendBytes(symbol, createByte(codePoint, 6))
	} else if (uint32(codePoint) & 0xFFE00000) == 0 { // 4-byte sequence
		symbol = stringFromCharCode(((codePoint >> 18) & 0x07) | 0xF0)
		symbol = appendBytes(symbol, createByte(codePoint, 12))
		symbol = appendBytes(symbol, createByte(codePoint, 6))
	}
	symbol = appendBytes(symbol, stringFromCharCode((codePoint&0x3F)|0x80))
	return symbol, nil
}

// implementation referenced from https://github.com/mathiasbynens/utf8.js/blob/2ce09544b62f2a274dbcd249473c0986e3660849/utf8.js
func Utf8encode(str string) (string, error) {
	var codePoints = ucs2decode(str)
	var length = len(codePoints)
	var index = 0
	var codePoint rune
	var bytes []byte
	for index < length {
		codePoint = codePoints[index]
		cps, err := encodeCodePoint(codePoint)
		if err != nil {
			return "", err
		}
		bytes = appendBytes(bytes, cps)
		index++
	}
	return string(bytes), nil
}

func readContinuationByte(byteArray []rune, byteCount int, pByteIndex *int) (rune, error) {
	if *pByteIndex >= byteCount {
		return utf8.RuneError, fmt.Errorf("invalid byte index")
	}

	var continuationByte = byteArray[*pByteIndex] & 0xFF
	*pByteIndex = *pByteIndex + 1

	if (continuationByte & 0xC0) == 0x80 {
		return continuationByte & 0x3F, nil
	}

	// If we end up here, it’s not a continuation byte
	return utf8.RuneError, fmt.Errorf("invalid continuation byte")
}

func decodeSymbol(byteArray []rune, byteCount int, pByteIndex *int) (rune, bool, error) {
	var byte1 rune
	var codePoint rune

	if *pByteIndex > byteCount {
		return utf8.RuneError, false, fmt.Errorf("invalid byte index")
	}

	if *pByteIndex == byteCount {
		return utf8.RuneError, false, nil
	}

	// Read first byte
	byte1 = byteArray[*pByteIndex] & 0xFF
	*pByteIndex = *pByteIndex + 1

	// 1-byte sequence (no continuation bytes)
	if (byte1 & 0x80) == 0 {
		return byte1, true, nil
	}

	// 2-byte sequence
	if (byte1 & 0xE0) == 0xC0 {
		byte2, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		codePoint = ((byte1 & 0x1F) << 6) | byte2
		if codePoint >= 0x80 {
			return codePoint, true, nil
		}
		return utf8.RuneError, false, fmt.Errorf("invalid continuation byte")
	}

	// 3-byte sequence (may include unpaired surrogates)
	if (byte1 & 0xF0) == 0xE0 {
		byte2, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		byte3, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		codePoint = ((byte1 & 0x0F) << 12) | (byte2 << 6) | byte3
		if codePoint >= 0x0800 {
			err := checkScalarValue(codePoint)
			if err != nil {
				return utf8.RuneError, false, err
			}
			return codePoint, true, nil
		}
		return utf8.RuneError, false, fmt.Errorf("invalid continuation byte")
	}

	// 4-byte sequence
	if (byte1 & 0xF8) == 0xF0 {
		byte2, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		byte3, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		byte4, err := readContinuationByte(byteArray, byteCount, pByteIndex)
		if err != nil {
			return utf8.RuneError, false, err
		}
		codePoint = ((byte1 & 0x07) << 0x12) | (byte2 << 0x0C) |
			(byte3 << 0x06) | byte4
		if codePoint >= 0x010000 && codePoint <= 0x10FFFF {
			return codePoint, true, nil
		}
	}

	return utf8.RuneError, false, fmt.Errorf("invalid UTF-8 detected")
}

// implementation referenced from https://github.com/mathiasbynens/utf8.js/blob/2ce09544b62f2a274dbcd249473c0986e3660849/utf8.js
func Utf8decode(str string) ([]byte, error) {
	byteArray := ucs2decode(str)
	byteCount := len(byteArray)
	byteIndex := 0
	var codePoints []rune
	for {
		codePoint, goOn, err := decodeSymbol(byteArray, byteCount, &byteIndex)
		if err != nil {
			return nil, err
		}
		if !goOn {
			break
		}
		codePoints = append(codePoints, codePoint)
	}
	return ucs2encode(codePoints), nil
}
