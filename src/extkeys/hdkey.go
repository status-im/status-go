package extkeys

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/ripemd160"
	"io"
	"math/big"
)

// Implementation of BIP32 https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
// Referencing
// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
// https://bitcoin.org/en/developer-guide#hardened-keys

// Reference Implementations
// https://github.com/btcsuite/btcutil/tree/master/hdkeychain
// https://github.com/WeMeetAgain/go-hdwallet

// https://github.com/ConsenSys/eth-lightwallet/blob/master/lib/keystore.js
// https://github.com/bitpay/bitcore-lib/tree/master/lib

// MUST CREATE HARDENED CHILDREN OF THE MASTER PRIVATE KEY (M) TO PREVENT
// A COMPROMISED CHILD KEY FROM COMPROMISING THE MASTER KEY.
// AS THERE ARE NO NORMAL CHILDREN FOR THE MASTER KEYS,
// THE MASTER PUBLIC KEY IS NOT USED IN HD WALLETS.
// ALL OTHER KEYS CAN HAVE NORMAL CHILDREN,
// SO THE CORRESPONDING EXTENDED PUBLIC KEYS MAY BE USED INSTEAD.

// TODO make sure we're doing this ^^^^ !!!!!!

const (
	HardenedKeyIndex          = 0x80000000 // 2^31
	PublicKeyCompressedLength = 33
)

var (
	InvalidKeyErr = errors.New("Key is invalid")
)

type HDKey struct {
	Key         []byte // 33 bytes, the public key or private key data (serP(K) for public keys, 0x00 || ser256(k) for private keys)
	Chain       []byte // 32 bytes, the chain code
	Depth       byte   // 1 byte,  depth: 0x00 for master nodes, 0x01 for level-1 derived keys, ....
	ChildNumber []byte // 4 bytes, This is ser32(i) for i in xi = xpar/i, with xi the key being serialized. (0x00000000 if master key)
	FingerPrint []byte // 4 bytes, fingerprint of the parent's key (0x00000000 if master key)
	IsPrivate   bool   // unserialized
}



func hash160(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	// hash := sha256.Sum256(data)
	hasher = ripemd160.New()
	io.WriteString(hasher, string(hasher.Sum(nil)))
	return hasher.Sum(nil)
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer

	// Write header; 0x2 for even y value; 0x3 for odd
	key.WriteByte(byte(0x2) + byte(y.Bit(0)))

	// Write X coord; Pad the key so x is aligned with the LSB. Pad size is key length - header size (1) - xBytes size
	xBytes := x.Bytes()
	for i := 0; i < (PublicKeyCompressedLength - 1 - len(xBytes)); i++ {
		key.WriteByte(0x0)
	}
	key.Write(xBytes)

	return key.Bytes()
}

// As described at https://bitcointa.lk/threads/compressed-keys-y-from-x.95735/
func expandPublicKey(key []byte) (*big.Int, *big.Int) {
	Y := big.NewInt(0)
	X := big.NewInt(0)
	qPlus1Div4 := big.NewInt(0)
	X.SetBytes(key[1:])

	// y^2 = x^3 + ax^2 + b
	// a = 0
	// => y^2 = x^3 + b
	ySquared := X.Exp(X, big.NewInt(3), nil)
	ySquared.Add(ySquared, curveParams.B)

	qPlus1Div4.Add(curveParams.P, big.NewInt(1))
	qPlus1Div4.Div(qPlus1Div4, big.NewInt(4))

	// sqrt(n) = n^((q+1)/4) if q = 3 mod 4
	Y.Exp(ySquared, qPlus1Div4, curveParams.P)

	if uint32(key[0])%2 == 0 {
		Y.Sub(curveParams.P, Y)
	}

	return X, Y
}

func addPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	return compressPublicKey(curve.Add(x1, y1, x2, y2))
}

// use MnemonicSeed instead
// Generate a seed byte sequence S of a chosen length
// (between 128 and 512 bits; 256 bits is advised) from a (P)RNG.
func RandSeed() ([]byte, error) {
	s := make([]byte, 32) // 256 bits
	_, err := rand.Read([]byte(s))
	return s, err
}

// Derive MasterKey
func MasterKey(seed []byte) (*HDKey, error) {

	// Ensure seed is bigger than 128 bits and smaller than 512 bits
	lseed := len(seed)
	if lseed < 16 || lseed > 64 {
		return nil, errors.New("The recommended size of seed is 128-512 bits")
	}

	// Calculate I = HMAC-SHA512(Key = "Bitcoin seed", Data = S)
	hmac := hmac.New(sha512.New, []byte(Salt)) // Salt defined in mnemonic.go
	hmac.Write([]byte(seed))
	I := hmac.Sum(nil)

	// Split I into two 32-byte sequences, IL and IR.
	// IL = master secret key
	// IR = master chain code
	key := I[:32]
	chain := I[32:]

	// In case IL is 0 or ≥n, the master key is invalid.
	keyBigInt := new(big.Int).SetBytes(key)
	if keyBigInt.Cmp(secp256k1.S256().N) >= 0 || keyBigInt.Sign() == 0 {
		return nil, InvalidKeyErr
	}

	master := &HDKey{
		Key:         key,
		Chain:       chain,
		Depth:       0x0,
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
		FingerPrint: []byte{0x00, 0x00, 0x00, 0x00},
		IsPrivate:   true,
	}

	return master, nil
}

// TODO review
func (parent *HDKey) ChildKey(i uint32) (*HDKey, error) {
	// There are four scenarios that could happen here:
	// 1) Private extended key -> Hardened child private extended key
	// 2) Private extended key -> Non-hardened child private extended key
	// 3) Public extended key -> Non-hardened child public extended key
	// 4) Public extended key -> Hardened child public extended key (INVALID!)

	isChildHardened := i >= HardenedKeyIndex
	if !parent.IsPrivate && isChildHardened {
		return nil, errors.New("Cannot create hardened key from public key")
	}

	childNumber := make([]byte, 4)
	binary.BigEndian.PutUint32(childNumber, i)

	var data []byte
	if isChildHardened {
		data = append([]byte{0x0}, parent.Key...)
	} else {
		// TODO verify
		data = compressPublicKey(secp256k1.S256().ScalarBaseMult(parent.Key))
	}
	data = append(data, childNumber...)

	hmac := hmac.New(sha512.New, parent.Chain)
	hmac.Write(data)
	I := hmac.Sum(nil)

	// Split I into two 32-byte sequences, IL and IR.
	// IL = master secret key
	// IR = master chain code
	key := I[:32]
	chain := I[32:]

	// In case IL is 0 or ≥n, the master key is invalid.
	keyBigInt := new(big.Int).SetBytes(key)
	if keyBigInt.Cmp(secp256k1.S256().N) >= 0 || keyBigInt.Sign() == 0 {
		return nil, InvalidKeyErr
	}

	child := &HDKey{
		// Key:
		Chain:       chain,
		Depth:       parent.Depth + 1,
		ChildNumber: childNumber,
		// FingerPrint:
		IsPrivate: parent.IsPrivate,
	}

	if parent.IsPrivate {
		// Case #1 or #2.
		// Add the parent private key to the intermediate private key to
		// derive the final child key.

		parentKeyBigInt := new(big.Int).SetBytes(parent.Key)
		keyBigInt.Add(keyBigInt, parentKeyBigInt)
		keyBigInt.Mod(keyBigInt, secp256k1.S256().N)
		child.Key = keyBigInt.Bytes()
		child.FingerPrint = hash160(compressPublicKey(secp256k1.S256().ScalarBaseMult(parent.Key)))[:4]
	} else {
		// Case #3.
		// Calculate the corresponding intermediate public key for
		// intermediate private key.

		keyx, keyy := secp256k1.S256().ScalarBaseMult(key)
		if keyx.Sign() == 0 || keyy.Sign() == 0 {
			return nil, InvalidKeyErr
		}

		publicKey := compressPublicKey(keyx, keyy)

		// TODO verify

		child.Key = addPublicKeys(publicKey, parent.Key)
		child.FingerPrint = hash160(parent.Key)[:4] // Not Private key so Key is public

	}
	return child, nil
}

func (hdkey *HDKey) ECKey() (*accounts.Key, error) {
	reader := bytes.NewReader(hdkey.Key)
	privateKeyECDSA, err := ecdsa.GenerateKey(secp256k1.S256(), reader)
	if err != nil {
		return nil, err
	}
	key := &accounts.Key{
		Id:         uuid.NewRandom(),
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key, nil
}
