package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/status-im/extkeys"
)

// URCryptoHDKeyToXPub converts a UR:CRYPTO-HDKEY encoded extended public key string to a standard BIP32 xpub string
//
// The UR:CRYPTO-HDKEY format follows the Blockchain Commons specification:
// https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-007-hdkey.md
//
// Only public keys (xpub) are supported. Returns an error if the UR contains a private key.
func URCryptoHDKeyToXPub(urString string) (string, error) {
	const urPrefix = "UR:CRYPTO-HDKEY/"
	if !strings.HasPrefix(strings.ToUpper(urString), urPrefix) {
		return "", errors.New("invalid UR: expected UR:CRYPTO-HDKEY/ prefix")
	}

	encoded := strings.ToLower(urString[len(urPrefix):])

	raw, err := bytewordsDecode(encoded)
	if err != nil {
		return "", fmt.Errorf("bytewords decode: %w", err)
	}

	if len(raw) < 5 {
		return "", errors.New("decoded data too short")
	}

	payload := raw[:len(raw)-4]
	storedChecksum := raw[len(raw)-4:]

	var computedChecksum [4]byte
	binary.BigEndian.PutUint32(computedChecksum[:], crc32.ChecksumIEEE(payload))
	if !bytes.Equal(storedChecksum, computedChecksum[:]) {
		return "", errors.New("CRC32 checksum mismatch")
	}

	keyData, chainCode, depth, parentFP, childNum, isPrivate, err := parseCryptoHDKeyCBOR(payload)
	if err != nil {
		return "", fmt.Errorf("CBOR parse: %w", err)
	}

	if isPrivate {
		return "", errors.New("UR contains a private key; only extended public keys (xpub) are supported")
	}

	fingerPrint := make([]byte, 4)
	binary.BigEndian.PutUint32(fingerPrint, parentFP)

	extKey := &extkeys.ExtendedKey{
		Version:     extkeys.PublicKeyVersion,
		KeyData:     keyData,
		ChainCode:   chainCode,
		FingerPrint: fingerPrint,
		Depth:       depth,
		ChildNumber: childNum,
		IsPrivate:   false,
	}

	return extKey.String(), nil
}

// bytewordsDecode decodes a bytewords minimal-encoded string to bytes.
// Each pair of characters (first+last letter of a bytewords word) represents one byte.
// Word list: https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-012-bytewords.md
func bytewordsDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errors.New("bytewords: odd-length input")
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		a, b := s[i], s[i+1]
		if a < 'a' || a > 'z' || b < 'a' || b > 'z' {
			return nil, fmt.Errorf("bytewords: invalid characters at position %d: %q", i, s[i:i+2])
		}
		val, ok := bytewordsLookup[int(a-'a')*26+int(b-'a')]
		if !ok {
			return nil, fmt.Errorf("bytewords: unknown word pair %q at position %d", s[i:i+2], i)
		}
		result[i/2] = val
	}

	return result, nil
}

// bytewordsLookup maps (first-'a')*26+(last-'a') → byte value.
// Built from the official 256-word BC bytewords list (each word is exactly 4 letters).
var bytewordsLookup = func() map[int]byte {
	words := [256]string{
		"able", "acid", "also", "apex", "aqua", "arch", "atom", "aunt",
		"away", "axis", "back", "bald", "barn", "belt", "beta", "bias",
		"blue", "body", "brag", "brew", "bulb", "buzz", "calm", "cash",
		"cats", "chef", "city", "claw", "code", "cola", "cook", "cost",
		"crux", "curl", "cusp", "cyan", "dark", "data", "days", "deli",
		"dice", "diet", "door", "down", "draw", "drop", "drum", "dull",
		"duty", "each", "easy", "echo", "edge", "epic", "even", "exam",
		"exit", "eyes", "fact", "fair", "fern", "figs", "film", "fish",
		"fizz", "flap", "flew", "flux", "foxy", "free", "frog", "fuel",
		"fund", "gala", "game", "gear", "gems", "gift", "girl", "glow",
		"good", "gray", "grim", "guru", "gush", "gyro", "half", "hang",
		"hard", "hawk", "heat", "help", "high", "hill", "holy", "hope",
		"horn", "huts", "iced", "idea", "idle", "inch", "inky", "into",
		"iris", "iron", "item", "jade", "jazz", "join", "jolt", "jowl",
		"judo", "jugs", "jump", "junk", "jury", "keep", "keno", "kept",
		"keys", "kick", "kiln", "king", "kite", "kiwi", "knob", "lamb",
		"lava", "lazy", "leaf", "legs", "liar", "limp", "lion", "list",
		"logo", "loud", "love", "luau", "luck", "lung", "main", "many",
		"math", "maze", "memo", "menu", "meow", "mild", "mint", "miss",
		"monk", "nail", "navy", "need", "news", "next", "noon", "note",
		"numb", "obey", "oboe", "omit", "onyx", "open", "oval", "owls",
		"paid", "part", "peck", "play", "plus", "poem", "pool", "pose",
		"puff", "puma", "purr", "quad", "quiz", "race", "ramp", "real",
		"redo", "rich", "road", "rock", "roof", "ruby", "ruin", "runs",
		"rust", "safe", "saga", "scar", "sets", "silk", "skew", "slot",
		"soap", "solo", "song", "stub", "surf", "swan", "taco", "task",
		"taxi", "tent", "tied", "time", "tiny", "toil", "tomb", "toys",
		"trip", "tuna", "twin", "ugly", "undo", "unit", "urge", "user",
		"vast", "very", "veto", "vial", "vibe", "view", "visa", "void",
		"vows", "wall", "wand", "warm", "wasp", "wave", "waxy", "webs",
		"what", "when", "whiz", "wolf", "work", "yank", "yawn", "yell",
		"yoga", "yurt", "zaps", "zero", "zest", "zinc", "zone", "zoom",
	}

	m := make(map[int]byte, 256)
	for i, w := range words {
		key := int(w[0]-'a')*26 + int(w[3]-'a')
		m[key] = byte(i)
	}
	return m
}()

// cborReader is a minimal CBOR reader for the crypto-hdkey structure.
type cborReader struct {
	data []byte
	pos  int
}

func (r *cborReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("CBOR: unexpected end of data")
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *cborReader) readUintArg(info byte) (uint64, error) {
	switch info {
	case 24:
		b, err := r.readByte()
		return uint64(b), err
	case 25:
		if r.pos+2 > len(r.data) {
			return 0, errors.New("CBOR: unexpected end reading 2-byte uint")
		}
		v := binary.BigEndian.Uint16(r.data[r.pos:])
		r.pos += 2
		return uint64(v), nil
	case 26:
		if r.pos+4 > len(r.data) {
			return 0, errors.New("CBOR: unexpected end reading 4-byte uint")
		}
		v := binary.BigEndian.Uint32(r.data[r.pos:])
		r.pos += 4
		return uint64(v), nil
	default:
		return uint64(info), nil
	}
}

func (r *cborReader) readBytes() ([]byte, error) {
	b, err := r.readByte()
	if err != nil {
		return nil, err
	}
	if b>>5 != 2 {
		return nil, fmt.Errorf("CBOR: expected bytes type, got major type %d", b>>5)
	}
	n, err := r.readUintArg(b & 0x1f)
	if err != nil {
		return nil, err
	}
	if r.pos+int(n) > len(r.data) {
		return nil, errors.New("CBOR: unexpected end reading bytes")
	}
	v := r.data[r.pos : r.pos+int(n)]
	r.pos += int(n)
	return v, nil
}

func (r *cborReader) readUint() (uint64, error) {
	b, err := r.readByte()
	if err != nil {
		return 0, err
	}
	if b>>5 != 0 {
		return 0, fmt.Errorf("CBOR: expected uint type, got major type %d (byte 0x%02x)", b>>5, b)
	}
	return r.readUintArg(b & 0x1f)
}

func (r *cborReader) skipItem() error {
	b, err := r.readByte()
	if err != nil {
		return err
	}
	majorType := b >> 5
	info := b & 0x1f

	switch majorType {
	case 0, 1: // uint, negint
		_, err = r.readUintArg(info)
		return err
	case 2, 3: // bytes, text
		n, err := r.readUintArg(info)
		if err != nil {
			return err
		}
		r.pos += int(n)
	case 4: // array
		n, err := r.readUintArg(info)
		if err != nil {
			return err
		}
		for i := uint64(0); i < n; i++ {
			if err := r.skipItem(); err != nil {
				return err
			}
		}
	case 5: // map
		n, err := r.readUintArg(info)
		if err != nil {
			return err
		}
		for i := uint64(0); i < n*2; i++ {
			if err := r.skipItem(); err != nil {
				return err
			}
		}
	case 6: // tag
		_, err := r.readUintArg(info)
		if err != nil {
			return err
		}
		return r.skipItem()
	case 7: // float/simple
		// only handle false(0xf4) and true(0xf5) which have no extra bytes
	}
	return nil
}

// parseCryptoHDKeyCBOR parses the CBOR payload of a crypto-hdkey UR and extracts
// the fields needed to reconstruct a BIP32 xpub.
func parseCryptoHDKeyCBOR(data []byte) (keyData, chainCode []byte, depth byte, parentFP, childNum uint32, isPrivate bool, err error) {
	r := &cborReader{data: data}

	// Outer structure must be a CBOR map
	b, err := r.readByte()
	if err != nil {
		return nil, nil, 0, 0, 0, false, err
	}
	if b>>5 != 5 {
		return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: expected map at root, got major type %d", b>>5)
	}
	mapLen, err := r.readUintArg(b & 0x1f)
	if err != nil {
		return nil, nil, 0, 0, 0, false, err
	}

	for i := uint64(0); i < mapLen; i++ {
		key, err := r.readUint()
		if err != nil {
			return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: reading map key %d: %w", i, err)
		}

		switch key {
		case 2: // is-private (bool)
			vb, err := r.readByte()
			if err != nil {
				return nil, nil, 0, 0, 0, false, err
			}
			isPrivate = vb == 0xf5 // true

		case 3: // key-data (bytes, 33-byte compressed public key)
			keyData, err = r.readBytes()
			if err != nil {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: reading key-data: %w", err)
			}
			if len(keyData) != 33 {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: key-data must be 33 bytes, got %d", len(keyData))
			}

		case 4: // chain-code (bytes, 32 bytes)
			chainCode, err = r.readBytes()
			if err != nil {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: reading chain-code: %w", err)
			}
			if len(chainCode) != 32 {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: chain-code must be 32 bytes, got %d", len(chainCode))
			}

		case 6: // origin (tag(304) crypto-keypath)
			depth, childNum, err = readCryptoKeypath(r)
			if err != nil {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: reading origin: %w", err)
			}

		case 8: // parent-fingerprint (uint32)
			v, err := r.readUint()
			if err != nil {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: reading parent-fingerprint: %w", err)
			}
			parentFP = uint32(v)

		default:
			if err := r.skipItem(); err != nil {
				return nil, nil, 0, 0, 0, false, fmt.Errorf("CBOR: skipping value for key %d: %w", key, err)
			}
		}
	}

	if keyData == nil {
		return nil, nil, 0, 0, 0, false, errors.New("CBOR: missing key-data (field 3)")
	}
	if chainCode == nil {
		return nil, nil, 0, 0, 0, false, errors.New("CBOR: missing chain-code (field 4)")
	}

	return keyData, chainCode, depth, parentFP, childNum, isPrivate, nil
}

// readCryptoKeypath reads a tag(304) crypto-keypath from the CBOR stream and returns
// the depth and last child number (used as the child number in the BIP32 serialization).
func readCryptoKeypath(r *cborReader) (depth byte, childNum uint32, err error) {
	// Expect tag(304)
	b, err := r.readByte()
	if err != nil {
		return 0, 0, err
	}
	if b>>5 != 6 {
		return 0, 0, fmt.Errorf("CBOR: expected tag for crypto-keypath, got major type %d", b>>5)
	}
	tag, err := r.readUintArg(b & 0x1f)
	if err != nil {
		return 0, 0, err
	}
	if tag != 304 {
		return 0, 0, fmt.Errorf("CBOR: expected tag 304 (crypto-keypath), got %d", tag)
	}

	// Inner map
	mb, err := r.readByte()
	if err != nil {
		return 0, 0, err
	}
	if mb>>5 != 5 {
		return 0, 0, fmt.Errorf("CBOR: expected map inside crypto-keypath, got major type %d", mb>>5)
	}
	innerLen, err := r.readUintArg(mb & 0x1f)
	if err != nil {
		return 0, 0, err
	}

	for i := uint64(0); i < innerLen; i++ {
		key, err := r.readUint()
		if err != nil {
			return 0, 0, err
		}

		switch key {
		case 1: // path components: flat array alternating [index, hardened, index, hardened, ...]
			ab, err := r.readByte()
			if err != nil {
				return 0, 0, err
			}
			if ab>>5 != 4 {
				return 0, 0, fmt.Errorf("CBOR: expected array for path components, got major type %d", ab>>5)
			}
			arrLen, err := r.readUintArg(ab & 0x1f)
			if err != nil {
				return 0, 0, err
			}
			if arrLen == 0 || arrLen%2 != 0 {
				return 0, 0, fmt.Errorf("CBOR: path components array length must be even and non-zero, got %d", arrLen)
			}

			var lastIndex uint64
			var lastHardened bool
			for j := uint64(0); j < arrLen; j++ {
				if j%2 == 0 {
					lastIndex, err = r.readUint()
					if err != nil {
						return 0, 0, fmt.Errorf("CBOR: reading path index at %d: %w", j, err)
					}
				} else {
					hb, err := r.readByte()
					if err != nil {
						return 0, 0, err
					}
					lastHardened = hb == 0xf5 // true
				}
			}

			childNum = uint32(lastIndex)
			if lastHardened {
				childNum += 0x80000000
			}

		case 3: // depth (uint)
			v, err := r.readUint()
			if err != nil {
				return 0, 0, err
			}
			depth = byte(v)

		default:
			if err := r.skipItem(); err != nil {
				return 0, 0, fmt.Errorf("CBOR: skipping keypath value for key %d: %w", key, err)
			}
		}
	}

	return depth, childNum, nil
}
