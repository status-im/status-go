// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func GenerateMessageParams() (*MessageParams, error) {
	// set all the parameters except p.Dst and p.Padding

	buf := make([]byte, 4)
	mrand.Read(buf)       // nolint: gosec
	sz := mrand.Intn(400) // nolint: gosec

	var p MessageParams
	p.Payload = make([]byte, sz)
	p.KeySym = make([]byte, AESKeyLength)
	mrand.Read(p.Payload) // nolint: gosec
	mrand.Read(p.KeySym)  // nolint: gosec
	p.Topic = BytesToTopic(buf)

	var err error
	p.Src, err = crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func singleMessageTest(t *testing.T, symmetric bool) {
	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}

	if !symmetric {
		params.KeySym = nil
		params.Dst = &key.PublicKey
	}

	text := make([]byte, 0, 512)
	text = append(text, params.Payload...)

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	var decrypted *ReceivedMessage
	if symmetric {
		decrypted, err = env.OpenSymmetric(params.KeySym)
	} else {
		decrypted, err = env.OpenAsymmetric(key)
	}

	if err != nil {
		t.Fatalf("failed to encrypt with seed %d: %s.", seed, err)
	}

	if !decrypted.ValidateAndParse() {
		t.Fatalf("failed to validate with seed %d, symmetric = %v.", seed, symmetric)
	}

	if !bytes.Equal(text, decrypted.Payload) {
		t.Fatalf("failed with seed %d: compare payload.", seed)
	}
	if !IsMessageSigned(decrypted.Raw[0]) {
		t.Fatalf("failed with seed %d: unsigned.", seed)
	}
	if len(decrypted.Signature) != signatureLength {
		t.Fatalf("failed with seed %d: signature len %d.", seed, len(decrypted.Signature))
	}
	if !IsPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		t.Fatalf("failed with seed %d: signature mismatch.", seed)
	}
}

func TestMessageEncryption(t *testing.T) {
	InitSingleTest()

	var symmetric bool
	for i := 0; i < 256; i++ {
		singleMessageTest(t, symmetric)
		symmetric = !symmetric
	}
}

func TestMessageWrap(t *testing.T) {
	seed = int64(1777444222)
	mrand.Seed(seed)
	target := 128.0

	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}

	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	pow := env.PoW()
	if pow < target {
		t.Fatalf("failed Wrap with seed %d: pow < target (%f vs. %f).", seed, pow, target)
	}

	// set PoW target too high, expect error
	msg2, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	_, err = msg2.Wrap(params, time.Now())
	if err == nil {
		t.Fatalf("unexpectedly reached the PoW target with seed %d.", seed)
	}
}

func TestMessageSeal(t *testing.T) {
	// this test depends on deterministic choice of seed (1976726903)
	seed = int64(1976726903)
	mrand.Seed(seed)

	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}

	env := NewEnvelope(params.TTL, params.Topic, msg, time.Now())

	env.Expiry = uint32(seed) // make it deterministic
	err = env.Seal(params)
	if err != nil {
		t.Logf("failed to seal envelope: %s", err)
	}

}

func TestEnvelopeOpen(t *testing.T) {
	InitSingleTest()

	var symmetric bool
	for i := 0; i < 32; i++ {
		singleEnvelopeOpenTest(t, symmetric)
		symmetric = !symmetric
	}
}

func singleEnvelopeOpenTest(t *testing.T, symmetric bool) {
	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}

	if !symmetric {
		params.KeySym = nil
		params.Dst = &key.PublicKey
	}

	text := make([]byte, 0, 512)
	text = append(text, params.Payload...)

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	var f Filter
	if symmetric {
		f = Filter{KeySym: params.KeySym}
	} else {
		f = Filter{KeyAsym: key}
	}
	decrypted := env.Open(&f)
	if decrypted == nil {
		t.Fatalf("failed to open with seed %d.", seed)
	}

	if !bytes.Equal(text, decrypted.Payload) {
		t.Fatalf("failed with seed %d: compare payload.", seed)
	}
	if !IsMessageSigned(decrypted.Raw[0]) {
		t.Fatalf("failed with seed %d: unsigned.", seed)
	}
	if len(decrypted.Signature) != signatureLength {
		t.Fatalf("failed with seed %d: signature len %d.", seed, len(decrypted.Signature))
	}
	if !IsPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		t.Fatalf("failed with seed %d: signature mismatch.", seed)
	}
	if decrypted.isAsymmetricEncryption() == symmetric {
		t.Fatalf("failed with seed %d: asymmetric %v vs. %v.", seed, decrypted.isAsymmetricEncryption(), symmetric)
	}
	if decrypted.isSymmetricEncryption() != symmetric {
		t.Fatalf("failed with seed %d: symmetric %v vs. %v.", seed, decrypted.isSymmetricEncryption(), symmetric)
	}
	if !symmetric {
		if decrypted.Dst == nil {
			t.Fatalf("failed with seed %d: dst is nil.", seed)
		}
		if !IsPubKeyEqual(decrypted.Dst, &key.PublicKey) {
			t.Fatalf("failed with seed %d: Dst.", seed)
		}
	}
}

func TestEncryptWithZeroKey(t *testing.T) {
	InitSingleTest()

	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	params.KeySym = make([]byte, AESKeyLength)
	_, err = msg.Wrap(params, time.Now())
	if err == nil {
		t.Fatalf("wrapped with zero key, seed: %d.", seed)
	}

	params, err = GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}
	msg, err = NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	params.KeySym = make([]byte, 0)
	_, err = msg.Wrap(params, time.Now())
	if err == nil {
		t.Fatalf("wrapped with empty key, seed: %d.", seed)
	}

	params, err = GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}
	msg, err = NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	params.KeySym = nil
	_, err = msg.Wrap(params, time.Now())
	if err == nil {
		t.Fatalf("wrapped with nil key, seed: %d.", seed)
	}
}

func TestRlpEncode(t *testing.T) {
	InitSingleTest()

	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d: %s.", seed, err)
	}
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("wrapped with zero key, seed: %d.", seed)
	}

	raw, err := rlp.EncodeToBytes(env)
	if err != nil {
		t.Fatalf("RLP encode failed: %s.", err)
	}

	var decoded Envelope
	err = rlp.DecodeBytes(raw, &decoded)
	if err != nil {
		t.Fatalf("RLP decode failed: %s.", err)
	}

	he := env.Hash()
	hd := decoded.Hash()

	if he != hd {
		t.Fatalf("Hashes are not equal: %x vs. %x", he, hd)
	}
}

func singlePaddingTest(t *testing.T, padSize int) {
	params, err := GenerateMessageParams()
	if err != nil {
		t.Fatalf("failed GenerateMessageParams with seed %d and sz=%d: %s.", seed, padSize, err)
	}
	params.Padding = make([]byte, padSize)
	pad := make([]byte, padSize)
	_, err = mrand.Read(pad) // nolint: gosec
	if err != nil {
		t.Fatalf("padding is not generated (seed %d): %s", seed, err)
	}
	n := copy(params.Padding, pad)
	if n != padSize {
		t.Fatalf("padding is not copied (seed %d): %s", seed, err)
	}
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed to wrap, seed: %d and sz=%d.", seed, padSize)
	}
	f := Filter{KeySym: params.KeySym}
	decrypted := env.Open(&f)
	if decrypted == nil {
		t.Fatalf("failed to open, seed and sz=%d: %d.", seed, padSize)
	}
	if !bytes.Equal(pad, decrypted.Padding) {
		t.Fatalf("padding is not retireved as expected with seed %d and sz=%d:\n[%x]\n[%x].", seed, padSize, pad, decrypted.Padding)
	}
}

func TestPadding(t *testing.T) {
	InitSingleTest()

	for i := 1; i < 260; i++ {
		singlePaddingTest(t, i)
	}

	lim := 256 * 256
	for i := lim - 5; i < lim+2; i++ {
		singlePaddingTest(t, i)
	}

	for i := 0; i < 256; i++ {
		n := mrand.Intn(256*254) + 256 // nolint: gosec
		singlePaddingTest(t, n)
	}

	for i := 0; i < 256; i++ {
		n := mrand.Intn(256*1024) + 256*256 // nolint: gosec
		singlePaddingTest(t, n)
	}
}

func TestPaddingAppendedToSymMessagesWithSignature(t *testing.T) {
	params := &MessageParams{
		Payload: make([]byte, 246),
		KeySym:  make([]byte, AESKeyLength),
	}

	pSrc, err := crypto.GenerateKey()

	if err != nil {
		t.Fatalf("Error creating the signature key %v", err)
		return
	}
	params.Src = pSrc

	// Simulate a message with a payload just under 256 so that
	// payload + flag + signature > 256. Check that the result
	// is padded on the next 256 boundary.
	msg := sentMessage{}
	const payloadSizeFieldMinSize = 1
	msg.Raw = make([]byte, flagsLength+payloadSizeFieldMinSize+len(params.Payload))

	err = msg.appendPadding(params)

	if err != nil {
		t.Fatalf("Error appending padding to message %v", err)
		return
	}

	if len(msg.Raw) != 512-signatureLength {
		t.Errorf("Invalid size %d != 512", len(msg.Raw))
	}
}

func TestAesNonce(t *testing.T) {
	key := hexutil.MustDecode("0x03ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher failed: %s", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("NewGCM failed: %s", err)
	}
	// This is the most important single test in this package.
	// If it fails, waku will not be working.
	if aesgcm.NonceSize() != aesNonceLength {
		t.Fatalf("Nonce size is wrong. This is a critical error. Apparently AES nonce size have changed in the new version of AES GCM package. Waku will not be working until this problem is resolved.")
	}
}

func TestValidateAndParseSizeOfPayloadSize(t *testing.T) {
	testCases := []struct {
		Name string
		Raw  []byte
	}{
		{
			Name: "one byte of value 1",
			Raw:  []byte{1},
		},
		{
			Name: "two bytes of values 1 and 1",
			Raw:  []byte{1, 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			msg := ReceivedMessage{Raw: tc.Raw}
			msg.ValidateAndParse()
		})
	}
}
