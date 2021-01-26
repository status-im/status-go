// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisper

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"math"
	mrand "math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"golang.org/x/crypto/pbkdf2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestWhisperBasic(t *testing.T) {
	w := New(&DefaultConfig)
	p := w.Protocols()
	shh := p[0]
	if shh.Name != ProtocolName {
		t.Fatalf("failed Protocol Name: %v.", shh.Name)
	}
	if uint64(shh.Version) != ProtocolVersion {
		t.Fatalf("failed Protocol Version: %v.", shh.Version)
	}
	if shh.Length != NumberOfMessageCodes {
		t.Fatalf("failed Protocol Length: %v.", shh.Length)
	}
	if shh.Run == nil {
		t.Fatalf("failed shh.Run.")
	}
	if uint64(w.Version()) != ProtocolVersion {
		t.Fatalf("failed whisper Version: %v.", shh.Version)
	}
	if w.GetFilter("non-existent") != nil {
		t.Fatalf("failed GetFilter.")
	}

	peerID := make([]byte, 64)
	mrand.Read(peerID) // nolint: gosec
	peer, _ := w.getPeer(peerID)
	if peer != nil {
		t.Fatal("found peer for random key.")
	}
	if err := w.AllowP2PMessagesFromPeer(peerID); err == nil {
		t.Fatalf("failed MarkPeerTrusted.")
	}
	exist := w.HasSymKey("non-existing")
	if exist {
		t.Fatalf("failed HasSymKey.")
	}
	key, err := w.GetSymKey("non-existing")
	if err == nil {
		t.Fatalf("failed GetSymKey(non-existing): false positive.")
	}
	if key != nil {
		t.Fatalf("failed GetSymKey: false positive.")
	}
	mail := w.Envelopes()
	if len(mail) != 0 {
		t.Fatalf("failed w.Envelopes().")
	}

	derived := pbkdf2.Key(peerID, nil, 65356, aesKeyLength, sha256.New)
	if !validateDataIntegrity(derived, aesKeyLength) {
		t.Fatalf("failed validateSymmetricKey with param = %v.", derived)
	}
	if containsOnlyZeros(derived) {
		t.Fatalf("failed containsOnlyZeros with param = %v.", derived)
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := bytesToUintLittleEndian(buf)
	be := BytesToUintBigEndian(buf)
	if le != uint64(0x280e5ff) {
		t.Fatalf("failed bytesToIntLittleEndian: %d.", le)
	}
	if be != uint64(0xffe5800200) {
		t.Fatalf("failed BytesToIntBigEndian: %d.", be)
	}

	id, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	pk, err := w.GetPrivateKey(id)
	if err != nil {
		t.Fatalf("failed to retrieve new key pair: %s.", err)
	}
	if !validatePrivateKey(pk) {
		t.Fatalf("failed validatePrivateKey: %v.", pk)
	}
	if !ValidatePublicKey(&pk.PublicKey) {
		t.Fatalf("failed ValidatePublicKey: %v.", pk)
	}
}

func TestWhisperAsymmetricKeyImport(t *testing.T) {
	var (
		w           = New(&DefaultConfig)
		privateKeys []*ecdsa.PrivateKey
	)

	for i := 0; i < 50; i++ {
		id, err := w.NewKeyPair()
		if err != nil {
			t.Fatalf("could not generate key: %v", err)
		}

		pk, err := w.GetPrivateKey(id)
		if err != nil {
			t.Fatalf("could not export private key: %v", err)
		}

		privateKeys = append(privateKeys, pk)

		if !w.DeleteKeyPair(id) {
			t.Fatalf("could not delete private key")
		}
	}

	for _, pk := range privateKeys {
		if _, err := w.AddKeyPair(pk); err != nil {
			t.Fatalf("could not import private key: %v", err)
		}
	}
}

func TestWhisperIdentityManagement(t *testing.T) {
	w := New(&DefaultConfig)
	id1, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	id2, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	pk1, err := w.GetPrivateKey(id1)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	pk2, err := w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}

	if !w.HasKeyPair(id1) {
		t.Fatalf("failed HasIdentity(pk1).")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed HasIdentity(pk2).")
	}
	if pk1 == nil {
		t.Fatalf("failed GetIdentity(pk1).")
	}
	if pk2 == nil {
		t.Fatalf("failed GetIdentity(pk2).")
	}

	if !validatePrivateKey(pk1) {
		t.Fatalf("pk1 is invalid.")
	}
	if !validatePrivateKey(pk2) {
		t.Fatalf("pk2 is invalid.")
	}

	// Delete one identity
	done := w.DeleteKeyPair(id1)
	if !done {
		t.Fatalf("failed to delete id1.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed DeleteIdentity(pub1): still exist.")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed DeleteIdentity(pub1): pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed DeleteIdentity(pub1): first key still exist.")
	}
	if pk2 == nil {
		t.Fatalf("failed DeleteIdentity(pub1): second key does not exist.")
	}

	// Delete again non-existing identity
	done = w.DeleteKeyPair(id1)
	if done {
		t.Fatalf("delete id1: false positive.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed delete non-existing identity: exist.")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed delete non-existing identity: pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed delete non-existing identity: first key exist.")
	}
	if pk2 == nil {
		t.Fatalf("failed delete non-existing identity: second key does not exist.")
	}

	// Delete second identity
	done = w.DeleteKeyPair(id2)
	if !done {
		t.Fatalf("failed to delete id2.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed delete second identity: first identity exist.")
	}
	if w.HasKeyPair(id2) {
		t.Fatalf("failed delete second identity: still exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed delete second identity: first key exist.")
	}
	if pk2 != nil {
		t.Fatalf("failed delete second identity: second key exist.")
	}
}

func TestWhisperSymKeyManagement(t *testing.T) {
	InitSingleTest()

	var err error
	var k1, k2 []byte
	w := New(&DefaultConfig)
	id2 := "arbitrary-string-2"

	id1, err := w.GenerateSymKey()
	if err != nil {
		t.Fatalf("failed GenerateSymKey with seed %d: %s.", seed, err)
	}

	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed GetSymKey(id2): false positive.")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("failed HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed HasSymKey(id2): false positive.")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if k2 != nil {
		t.Fatalf("second key still exist.")
	}

	// add existing id, nothing should change
	randomKey := make([]byte, aesKeyLength)
	mrand.Read(randomKey) // nolint: gosec
	id1, err = w.AddSymKeyDirect(randomKey)
	if err != nil {
		t.Fatalf("failed AddSymKey with seed %d: %s.", seed, err)
	}

	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id2): false positive.")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("failed w.HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed w.HasSymKey(id2): false positive.")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if !bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 != randomKey.")
	}
	if k2 != nil {
		t.Fatalf("second key already exist.")
	}

	id2, err = w.AddSymKeyDirect(randomKey)
	if err != nil {
		t.Fatalf("failed AddSymKey(id2) with seed %d: %s.", seed, err)
	}
	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("HasSymKey(id1) failed.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("HasSymKey(id2) failed.")
	}
	if k1 == nil {
		t.Fatalf("k1 does not exist.")
	}
	if k2 == nil {
		t.Fatalf("k2 does not exist.")
	}
	if !bytes.Equal(k1, k2) {
		t.Fatalf("k1 != k2.")
	}
	if !bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 != randomKey.")
	}
	if len(k1) != aesKeyLength {
		t.Fatalf("wrong length of k1.")
	}
	if len(k2) != aesKeyLength {
		t.Fatalf("wrong length of k2.")
	}

	w.DeleteSymKey(id1)
	k1, err = w.GetSymKey(id1)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id1): false positive.")
	}
	if k1 != nil {
		t.Fatalf("failed GetSymKey(id1): false positive.")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
	if w.HasSymKey(id1) {
		t.Fatalf("failed to delete first key: still exist.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("failed to delete first key: second key does not exist.")
	}
	if k1 != nil {
		t.Fatalf("failed to delete first key.")
	}
	if k2 == nil {
		t.Fatalf("failed to delete first key: second key is nil.")
	}

	w.DeleteSymKey(id1)
	w.DeleteSymKey(id2)
	k1, err = w.GetSymKey(id1)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id1): false positive.")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id2): false positive.")
	}
	if k1 != nil || k2 != nil {
		t.Fatalf("k1 or k2 is not nil")
	}
	if w.HasSymKey(id1) {
		t.Fatalf("failed to delete second key: first key exist.")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed to delete second key: still exist.")
	}
	if k1 != nil {
		t.Fatalf("failed to delete second key: first key is not nil.")
	}
	if k2 != nil {
		t.Fatalf("failed to delete second key: second key is not nil.")
	}

	randomKey = make([]byte, aesKeyLength+1)
	mrand.Read(randomKey) // nolint: gosec
	_, err = w.AddSymKeyDirect(randomKey)
	if err == nil {
		t.Fatalf("added the key with wrong size, seed %d.", seed)
	}

	const password = "arbitrary data here"
	id1, err = w.AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKeyFromPassword(id1) with seed %d: %s.", seed, err)
	}
	id2, err = w.AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKeyFromPassword(id2) with seed %d: %s.", seed, err)
	}
	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("HasSymKey(id1) failed.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("HasSymKey(id2) failed.")
	}
	if !validateDataIntegrity(k2, aesKeyLength) {
		t.Fatalf("key validation failed.")
	}
	if !bytes.Equal(k1, k2) {
		t.Fatalf("k1 != k2.")
	}
}

func TestExpiry(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	w.SetMinimumPowTest(0.0000001)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params.TTL = 1

	messagesCount := 5

	// Send a few messages one after another. Due to low PoW and expiration buckets
	// with one second resolution, it covers a case when there are multiple items
	// in a single expiration bucket.
	for i := 0; i < messagesCount; i++ {
		msg, err := NewSentMessage(params)
		if err != nil {
			t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
		}
		env, err := msg.Wrap(params, time.Now())
		if err != nil {
			t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
		}

		err = w.Send(env)
		if err != nil {
			t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
		}
	}

	// wait till received or timeout
	var received, expired bool
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) == messagesCount {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// wait till expired or timeout
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) == 0 {
			expired = true
			break
		}
	}

	if !expired {
		t.Fatalf("expire failed, seed: %d.", seed)
	}
}

func TestCustomization(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer func() { _ = w.SetMaxMessageSize(DefaultMaxMessageSize) }()
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	const smallPoW = 0.00001

	f, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.KeySym = f.KeySym
	params.Topic = BytesToTopic(f.Topics[2])
	params.PoW = smallPoW
	params.TTL = 3600 * 24 // one day
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err == nil {
		t.Fatalf("successfully sent envelope with PoW %.06f, false positive (seed %d).", env.PoW(), seed)
	}

	w.SetMinimumPowTest(smallPoW / 2)
	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}

	params.TTL++
	msg, err = NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err = msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}
	_ = w.SetMaxMessageSize(uint32(env.size() - 1))
	err = w.Send(env)
	if err == nil {
		t.Fatalf("successfully sent oversized envelope (seed %d): false positive.", seed)
	}

	_ = w.SetMaxMessageSize(DefaultMaxMessageSize)
	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send second envelope with seed %d: %s.", seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) > 1 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	_, err = w.Subscribe(f)
	if err != nil {
		t.Fatalf("failed subscribe with seed %d: %s.", seed, err)
	}
	time.Sleep(5 * time.Millisecond)
	mail := f.Retrieve()
	if len(mail) > 0 {
		t.Fatalf("received premature mail")
	}
}

func TestSymmetricSendCycle(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer func() { _ = w.SetMaxMessageSize(DefaultMaxMessageSize) }()
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	filter1, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter1.PoW = DefaultMinimumPoW

	// Copy the first filter since some of its fields
	// are randomly gnerated.
	filter2 := &Filter{
		KeySym:   filter1.KeySym,
		Topics:   filter1.Topics,
		PoW:      filter1.PoW,
		AllowP2P: filter1.AllowP2P,
		Messages: NewMemoryMessageStore(),
	}

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter1.Src = &params.Src.PublicKey
	filter2.Src = &params.Src.PublicKey

	params.KeySym = filter1.KeySym
	params.Topic = BytesToTopic(filter1.Topics[2])
	params.PoW = filter1.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter1)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter2)
	if err != nil {
		t.Fatalf("failed subscribe 2 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail1 := filter1.Retrieve()
	mail2 := filter2.Retrieve()
	if len(mail2) == 0 {
		t.Fatalf("did not receive any email for filter 2")
	}
	if len(mail1) == 0 {
		t.Fatalf("did not receive any email for filter 1")
	}

}

func TestSymmetricSendWithoutAKey(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer func() { _ = w.SetMaxMessageSize(DefaultMaxMessageSize) }()
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter.Src = nil

	params.KeySym = filter.KeySym
	params.Topic = BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail := filter.Retrieve()
	if len(mail) == 0 {
		t.Fatalf("did not receive message in spite of not setting a public key")
	}
}

func TestSymmetricSendKeyMismatch(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer func() { _ = w.SetMaxMessageSize(DefaultMaxMessageSize) }()
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.KeySym = filter.KeySym
	params.Topic = BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail := filter.Retrieve()
	if len(mail) > 0 {
		t.Fatalf("received a message when keys weren't matching")
	}
}

func TestBloom(t *testing.T) {
	topic := TopicType{0, 0, 255, 6}
	b := TopicToBloom(topic)
	x := make([]byte, BloomFilterSize)
	x[0] = byte(1)
	x[32] = byte(1)
	x[BloomFilterSize-1] = byte(128)
	if !BloomFilterMatch(x, b) || !BloomFilterMatch(b, x) {
		t.Fatalf("bloom filter does not match the mask")
	}

	_, err := mrand.Read(b) // nolint: gosec
	if err != nil {
		t.Fatalf("math rand error")
	}
	_, err = mrand.Read(x) // nolint: gosec
	if err != nil {
		t.Fatalf("math rand error")
	}
	if !BloomFilterMatch(b, b) {
		t.Fatalf("bloom filter does not match self")
	}
	x = addBloom(x, b)
	if !BloomFilterMatch(x, b) {
		t.Fatalf("bloom filter does not match combined bloom")
	}
	if !isFullNode(nil) {
		t.Fatalf("isFullNode did not recognize nil as full node")
	}
	x[17] = 254
	if isFullNode(x) {
		t.Fatalf("isFullNode false positive")
	}
	for i := 0; i < BloomFilterSize; i++ {
		b[i] = byte(255)
	}
	if !isFullNode(b) {
		t.Fatalf("isFullNode false negative")
	}
	if BloomFilterMatch(x, b) {
		t.Fatalf("bloomFilterMatch false positive")
	}
	if !BloomFilterMatch(b, x) {
		t.Fatalf("bloomFilterMatch false negative")
	}

	w := New(&DefaultConfig)
	f := w.BloomFilter()
	if f != nil {
		t.Fatalf("wrong bloom on creation")
	}
	err = w.SetBloomFilter(x)
	if err != nil {
		t.Fatalf("failed to set bloom filter: %s", err)
	}
	f = w.BloomFilter()
	if !BloomFilterMatch(f, x) || !BloomFilterMatch(x, f) {
		t.Fatalf("retireved wrong bloom filter")
	}
}

func TestSendP2PDirect(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	w.SetMinimumPowTest(0.0000001)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	rwStub := &rwP2PMessagesStub{}
	peerW := newPeer(w, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rwStub)
	w.peers[peerW] = struct{}{}

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params.TTL = 1

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	// verify sending a single envelope
	err = w.SendP2PDirect(peerW.ID(), env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}
	if len(rwStub.messages) != 1 {
		t.Fatalf("invalid number of messages sent to peer: %d, expected 1", len(rwStub.messages))
	}
	var envelope Envelope
	if err := rwStub.messages[0].Decode(&envelope); err != nil {
		t.Fatalf("failed to decode envelopes: %s", err)
	}
	if envelope.Hash() != env.Hash() {
		t.Fatalf("invalid envelope %d, expected %d", envelope.Hash(), env.Hash())
	}
	rwStub.messages = nil

	// send a batch of envelopes
	err = w.SendP2PDirect(peerW.ID(), env, env, env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}
	if len(rwStub.messages) != 1 {
		t.Fatalf("invalid number of messages sent to peer: %d, expected 1", len(rwStub.messages))
	}
	var envelopes []*Envelope
	if err := rwStub.messages[0].Decode(&envelopes); err != nil {
		t.Fatalf("failed to decode envelopes: %s", err)
	}
	if len(envelopes) != 3 {
		t.Fatalf("invalid number of envelopes in a message: %d, expected 3", len(envelopes))
	}
	rwStub.messages = nil
	envelopes = nil
}

func TestHandleP2PMessageCode(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	w.SetMinimumPowTest(0.0000001)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	_ = w.Start(nil)
	defer func() { _ = w.Stop() }()

	envelopeEvents := make(chan EnvelopeEvent, 10)
	sub := w.SubscribeEnvelopeEvents(envelopeEvents)
	defer sub.Unsubscribe()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params.TTL = 1

	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	// read a single envelope
	rwStub := &rwP2PMessagesStub{}
	rwStub.payload = []interface{}{env}

	peer := newPeer(nil, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), nil)
	peer.trusted = true

	err = w.runMessageLoop(peer, rwStub)
	if err != nil && err != errRWStub {
		t.Fatalf("failed run message loop: %s", err)
	}
	if e := <-envelopeEvents; e.Hash != env.Hash() {
		t.Fatalf("received envelope %s while expected %s", e.Hash, env.Hash())
	}

	// read a batch of envelopes
	rwStub = &rwP2PMessagesStub{}
	rwStub.payload = []interface{}{[]*Envelope{env, env, env}}

	err = w.runMessageLoop(peer, rwStub)
	if err != nil && err != errRWStub {
		t.Fatalf("failed run message loop: %s", err)
	}
	for i := 0; i < 3; i++ {
		if e := <-envelopeEvents; e.Hash != env.Hash() {
			t.Fatalf("received envelope %s while expected %s", e.Hash, env.Hash())
		}
	}
}

var errRWStub = errors.New("no more messages")

type rwP2PMessagesStub struct {
	// payload stores individual messages that will be sent returned
	// on ReadMsg() class
	payload  []interface{}
	messages []p2p.Msg
}

func (stub *rwP2PMessagesStub) ReadMsg() (p2p.Msg, error) {
	if len(stub.payload) == 0 {
		return p2p.Msg{}, errRWStub
	}
	size, r, err := rlp.EncodeToReader(stub.payload[0])
	if err != nil {
		return p2p.Msg{}, err
	}
	stub.payload = stub.payload[1:]
	return p2p.Msg{Code: p2pMessageCode, Size: uint32(size), Payload: r}, nil
}

func (stub *rwP2PMessagesStub) WriteMsg(m p2p.Msg) error {
	stub.messages = append(stub.messages, m)
	return nil
}

func testConfirmationsHandshake(t *testing.T, expectConfirmations bool) {
	conf := &Config{
		MinimumAcceptedPOW:   0,
		DisableConfirmations: !expectConfirmations,
	}
	w := New(conf)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	// so that actual read won't hang forever
	time.AfterFunc(5*time.Second, func() {
		rw1.Close()
	})
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, expectConfirmations, RateLimits{}},
	))
}

func TestConfirmationHadnshakeExtension(t *testing.T) {
	testConfirmationsHandshake(t, true)
}

func TestHandshakeWithConfirmationsDisabled(t *testing.T) {
	testConfirmationsHandshake(t, false)
}

func TestConfirmationReceived(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	time.AfterFunc(5*time.Second, func() {
		rw1.Close()
	})
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true, RateLimits{}},
	))
	require.NoError(t, p2p.SendItems(rw1, statusCode, ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, true))

	e := Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	data, err := rlp.EncodeToBytes([]*Envelope{&e})
	require.NoError(t, err)
	hash := crypto.Keccak256Hash(data)
	require.NoError(t, p2p.SendItems(rw1, messagesCode, &e))
	require.NoError(t, p2p.ExpectMsg(rw1, messageResponseCode, nil))
	require.NoError(t, p2p.ExpectMsg(rw1, batchAcknowledgedCode, hash))
}

func TestMessagesResponseWithError(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		rw1.Close()
		rw2.Close()
	}()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true, RateLimits{}},
	))
	require.NoError(
		t,
		p2p.SendItems(rw1, statusCode, ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, true),
	)

	failed := Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	normal := Envelope{
		Expiry: uint32(time.Now().Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}

	data, err := rlp.EncodeToBytes([]*Envelope{&failed, &normal})
	require.NoError(t, err)
	hash := crypto.Keccak256Hash(data)
	require.NoError(t, p2p.SendItems(rw1, messagesCode, &failed, &normal))
	require.NoError(t, p2p.ExpectMsg(rw1, messageResponseCode, NewMessagesResponse(hash, []EnvelopeError{
		{Hash: failed.Hash(), Code: EnvelopeTimeNotSynced, Description: "envelope from future"},
	})))
	require.NoError(t, p2p.ExpectMsg(rw1, batchAcknowledgedCode, hash))
}

func testConfirmationEvents(t *testing.T, envelope Envelope, envelopeErrors []EnvelopeError) {
	conf := &Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf)
	events := make(chan EnvelopeEvent, 2)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	time.AfterFunc(5*time.Second, func() {
		rw1.Close()
	})
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true, RateLimits{}},
	))
	require.NoError(t, p2p.SendItems(rw1, statusCode, ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, true))

	require.NoError(t, w.Send(&envelope))
	require.NoError(t, p2p.ExpectMsg(rw1, messagesCode, []*Envelope{&envelope}))

	var hash common.Hash
	select {
	case ev := <-events:
		require.Equal(t, EventEnvelopeSent, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.NotEqual(t, common.Hash{}, ev.Batch)
		hash = ev.Batch
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for an envelope.sent event")
	}
	require.NoError(t, p2p.Send(rw1, messageResponseCode, NewMessagesResponse(hash, envelopeErrors)))
	require.NoError(t, p2p.Send(rw1, batchAcknowledgedCode, hash))
	select {
	case ev := <-events:
		require.Equal(t, EventBatchAcknowledged, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.Equal(t, hash, ev.Batch)
		require.Equal(t, envelopeErrors, ev.Data)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for an batch.acknowledged event")
	}
}

func TestConfirmationEventsReceived(t *testing.T) {
	e := Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	testConfirmationEvents(t, e, []EnvelopeError{})
}

func TestConfirmationEventsExtendedWithErrors(t *testing.T) {
	e := Envelope{
		Expiry: uint32(time.Now().Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	testConfirmationEvents(t, e, []EnvelopeError{
		{
			Hash:        e.Hash(),
			Code:        EnvelopeTimeNotSynced,
			Description: "test error",
		}},
	)
}

func TestEventsWithoutConfirmation(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf)
	events := make(chan EnvelopeEvent, 2)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	time.AfterFunc(5*time.Second, func() {
		rw1.Close()
	})
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true, RateLimits{}},
	))
	require.NoError(t, p2p.SendItems(rw1, statusCode, ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, false))

	e := Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	require.NoError(t, w.Send(&e))
	require.NoError(t, p2p.ExpectMsg(rw1, messagesCode, []*Envelope{&e}))

	select {
	case ev := <-events:
		require.Equal(t, EventEnvelopeSent, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.Equal(t, common.Hash{}, ev.Batch)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for an envelope.sent event")
	}
}

func discardPipe() *p2p.MsgPipeRW {
	rw1, rw2 := p2p.MsgPipe()
	go func() {
		for {
			msg, err := rw1.ReadMsg()
			if err != nil {
				return
			}
			_ = msg.Discard()
		}
	}()
	return rw2
}

func TestWhisperTimeDesyncEnvelopeIgnored(t *testing.T) {
	c := &Config{
		MaxMessageSize:     DefaultMaxMessageSize,
		MinimumAcceptedPOW: 0,
	}
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		rw1.Close()
		rw2.Close()
	}()
	p1 := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	p2 := p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"shh", 6}})
	w1, w2 := New(c), New(c)
	errc := make(chan error)
	go func() {
		_ = w1.HandlePeer(p2, rw2)
	}()
	go func() {
		errc <- w2.HandlePeer(p1, rw1)
	}()
	w1.SetTimeSource(func() time.Time {
		return time.Now().Add(time.Hour)
	})
	env := &Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    30,
		Topic:  TopicType{1},
		Data:   []byte{1, 1, 1},
	}
	require.NoError(t, w1.Send(env))
	select {
	case err := <-errc:
		require.NoError(t, err)
	case <-time.After(time.Second):
	}
	rw2.Close()
	select {
	case err := <-errc:
		require.Error(t, err, "p2p: read or write on closed message pipe")
	case <-time.After(time.Second):
		require.FailNow(t, "connection wasn't closed in expected time")
	}
}

func TestRequestSentEventWithExpiry(t *testing.T) {
	w := New(nil)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw := discardPipe()
	defer rw.Close()
	w.peers[newPeer(w, p, rw)] = struct{}{}
	events := make(chan EnvelopeEvent, 1)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	e := &Envelope{Nonce: 1}
	require.NoError(t, w.RequestHistoricMessagesWithTimeout(p.ID().Bytes(), e, time.Millisecond))
	verifyEvent := func(etype EventType) {
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "error waiting for a event type %s", etype)
		case ev := <-events:
			require.Equal(t, etype, ev.Event)
			require.Equal(t, p.ID(), ev.Peer)
			require.Equal(t, e.Hash(), ev.Hash)
		}
	}
	verifyEvent(EventMailServerRequestSent)
	verifyEvent(EventMailServerRequestExpired)
}

func TestSendMessagesRequest(t *testing.T) {
	validMessagesRequest := MessagesRequest{
		ID:    make([]byte, 32),
		From:  0,
		To:    10,
		Bloom: []byte{0x01},
	}

	t.Run("InvalidID", func(t *testing.T) {
		w := New(nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, MessagesRequest{})
		require.EqualError(t, err, "invalid 'ID', expected a 32-byte slice")
	})

	t.Run("WithoutPeer", func(t *testing.T) {
		w := New(nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, validMessagesRequest)
		require.EqualError(t, err, "Could not find peer with ID: 0102")
	})

	t.Run("AllGood", func(t *testing.T) {
		p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
		rw1, rw2 := p2p.MsgPipe()
		w := New(nil)
		w.peers[newPeer(w, p, rw1)] = struct{}{}

		go func() {
			err := w.SendMessagesRequest(p.ID().Bytes(), validMessagesRequest)
			require.NoError(t, err)
		}()

		require.NoError(t, p2p.ExpectMsg(rw2, p2pRequestCode, nil))
	})
}

func TestSyncMessages(t *testing.T) {
	t.Run("WithoutMailServer", func(t *testing.T) {
		w := New(nil)
		err := w.SyncMessages(nil, SyncMailRequest{})
		require.EqualError(t, err, "can not sync messages if Mail Server is not configured")
	})

	t.Run("WithoutPeer", func(t *testing.T) {
		w := New(nil)
		w.RegisterMailServer(&stubMailServer{})
		err := w.SyncMessages([]byte{0x01, 0x02}, SyncMailRequest{})
		require.EqualError(t, err, "Could not find peer with ID: 0102")
	})

	t.Run("WithInvalidRequest", func(t *testing.T) {
		w := New(nil)
		w.RegisterMailServer(&stubMailServer{})

		p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
		rw1, _ := p2p.MsgPipe()

		w.peers[newPeer(w, p, rw1)] = struct{}{}

		err := w.SyncMessages(p.ID().Bytes(), SyncMailRequest{Limit: 10, Lower: 10, Upper: 5})
		require.EqualError(t, err, "invalid 'Lower' value, can't be greater than 'Upper'")
	})

	t.Run("AllGood", func(t *testing.T) {
		w := New(nil)
		w.RegisterMailServer(&stubMailServer{})

		p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
		rw1, rw2 := p2p.MsgPipe()

		w.peers[newPeer(w, p, rw1)] = struct{}{}

		go func() {
			err := w.SyncMessages(p.ID().Bytes(), SyncMailRequest{Limit: 10})
			require.NoError(t, err)
		}()

		require.NoError(t, p2p.ExpectMsg(rw2, p2pSyncRequestCode, nil))
	})
}

func TestSendSyncResponse(t *testing.T) {
	w := New(nil)
	p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
	rw1, rw2 := p2p.MsgPipe()
	whisperPeer := newPeer(w, p, rw1)
	w.peers[whisperPeer] = struct{}{}
	errC := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		errC <- w.SendSyncResponse(whisperPeer.ID(), SyncResponse{})
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		errC <- p2p.ExpectMsg(rw2, p2pSyncResponseCode, nil)
		wg.Done()
	}()

	require.NoError(t, <-errC)
	require.NoError(t, <-errC)
}

func TestHandleP2PSyncRequestCode(t *testing.T) {
	rw1, rw2 := p2p.MsgPipe()
	peer := newPeer(nil, p2p.NewPeer(enode.ID{}, "test", nil), nil)

	mailMock := &mockMailServer{}
	mailMock.On("SyncMail", peer.ID(), mock.Anything).Return(nil)

	w := New(nil)
	w.RegisterMailServer(mailMock)

	go func() {
		err := p2p.Send(rw1, p2pSyncRequestCode, SyncMailRequest{Limit: 10})
		require.NoError(t, err)
		require.NoError(t, rw1.Close())
	}()

	err := w.runMessageLoop(peer, rw2)
	if err != nil && err != p2p.ErrPipeClosed {
		require.FailNow(t, "invalid err", err.Error())
	}

	mailMock.AssertNumberOfCalls(t, "SyncMail", 1)
}

func TestHandleP2PSyncRequestCodeWithInvalidRequest(t *testing.T) {
	rw1, rw2 := p2p.MsgPipe()
	peer := newPeer(nil, p2p.NewPeer(enode.ID{}, "test", nil), nil)

	mailMock := &mockMailServer{}
	mailMock.On("SyncMail", peer, mock.Anything).Return(nil)

	w := New(nil)
	w.RegisterMailServer(mailMock)

	// create an invalid request
	req := SyncMailRequest{Limit: 10, Lower: 10, Upper: 5}
	require.Error(t, req.Validate())

	go func() {
		err := p2p.Send(rw1, p2pSyncRequestCode, req)
		require.NoError(t, err)
		require.NoError(t, rw1.Close())
	}()

	err := w.runMessageLoop(peer, rw2)
	require.EqualError(t, err, "sync mail request was invalid: invalid 'Lower' value, can't be greater than 'Upper'")

	mailMock.AssertNumberOfCalls(t, "SyncMail", 0)
}

func TestHandleP2PSyncResponseCode(t *testing.T) {
	rw1, rw2 := p2p.MsgPipe()
	peer := newPeer(nil, p2p.NewPeer(enode.ID{}, "test", nil), nil)
	peer.trusted = true

	mailMock := &mockMailServer{}
	mailMock.On("Archive", mock.Anything)

	w := New(nil)
	w.RegisterMailServer(mailMock)

	envelopesCount := 3

	go func() {
		params, err := generateMessageParams()
		require.NoError(t, err)
		msg, err := NewSentMessage(params)
		require.NoError(t, err)

		var envelopes []*Envelope
		for i := 0; i < envelopesCount; i++ {
			env, err := msg.Wrap(params, time.Now())
			require.NoError(t, err)
			envelopes = append(envelopes, env)
		}

		err = p2p.Send(rw1, p2pSyncResponseCode, SyncResponse{
			Envelopes: envelopes,
		})
		require.NoError(t, err)
		require.NoError(t, rw1.Close())
	}()

	err := w.runMessageLoop(peer, rw2)
	if err != nil && err != p2p.ErrPipeClosed {
		require.FailNow(t, "invalid err", err)
	}

	mailMock.AssertNumberOfCalls(t, "Archive", envelopesCount)
}

func TestRateLimiterIntegration(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPOW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf)
	w.SetRateLimiter(NewPeerRateLimiter(nil, &MetricsRateLimiterHandler{}))
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		rw1.Close()
		rw2.Close()
	}()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	require.NoError(t, p2p.ExpectMsg(
		rw1,
		statusCode,
		[]interface{}{ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), false, true, w.RateLimits()},
	))
	require.NoError(t, p2p.SendItems(rw1, statusCode, ProtocolVersion, math.Float64bits(w.MinPow()), w.BloomFilter(), true, true))

	envelope := Envelope{
		Expiry: uint32(time.Now().Unix()),
		TTL:    10,
		Topic:  TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}

	data, err := rlp.EncodeToBytes([]*Envelope{&envelope})
	require.NoError(t, err)
	hash := crypto.Keccak256Hash(data)
	require.NoError(t, p2p.SendItems(rw1, messagesCode, &envelope))
	require.NoError(t, p2p.ExpectMsg(rw1, messageResponseCode, NewMessagesResponse(hash, nil)))
}

type stubMailServer struct{}

func (stubMailServer) Archive(*Envelope)                      {}
func (stubMailServer) DeliverMail([]byte, *Envelope)          {}
func (stubMailServer) Deliver([]byte, MessagesRequest)        {}
func (stubMailServer) SyncMail([]byte, SyncMailRequest) error { return nil }

type mockMailServer struct {
	mock.Mock
}

func (m *mockMailServer) Archive(env *Envelope) {
	m.Called(env)
}

func (m *mockMailServer) DeliverMail(p []byte, env *Envelope) {
	m.Called(p, env)
}

func (m *mockMailServer) Deliver(p []byte, r MessagesRequest) {
	m.Called(p, r)
}

func (m *mockMailServer) SyncMail(p []byte, r SyncMailRequest) error {
	args := m.Called(p, r)
	return args.Error(0)
}

func TestMailserverCompletionEvent(t *testing.T) {
	w := New(nil)
	require.NoError(t, w.Start(nil))
	defer func() { _ = w.Stop() }()

	rw1, rw2 := p2p.MsgPipe()
	peer := newPeer(w, p2p.NewPeer(enode.ID{1}, "1", nil), rw1)
	peer.trusted = true
	w.peers[peer] = struct{}{}

	events := make(chan EnvelopeEvent)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	envelopes := []*Envelope{{Data: []byte{1}}, {Data: []byte{2}}}
	go func() {
		require.NoError(t, p2p.Send(rw2, p2pMessageCode, envelopes))
		require.NoError(t, p2p.Send(rw2, p2pRequestCompleteCode, [100]byte{})) // 2 hashes + cursor size
		rw2.Close()
	}()
	require.EqualError(t, w.runMessageLoop(peer, rw1), "p2p: read or write on closed message pipe")

	after := time.After(2 * time.Second)
	count := 0
	for {
		select {
		case <-after:
			require.FailNow(t, "timed out waiting for all events")
		case ev := <-events:
			switch ev.Event {
			case EventEnvelopeAvailable:
				count++
			case EventMailServerRequestCompleted:
				require.Equal(t, count, len(envelopes),
					"all envelope.avaiable events mut be recevied before request is compelted")
				return
			}
		}
	}
}
