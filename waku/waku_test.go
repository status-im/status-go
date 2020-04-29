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

package waku

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"math"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/pbkdf2"

	"github.com/status-im/status-go/waku/common"
	v0 "github.com/status-im/status-go/waku/v0"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"go.uber.org/zap"
)

var seed int64

// InitSingleTest should be called in the beginning of every
// test, which uses RNG, in order to make the tests
// reproducibility independent of their sequence.
func InitSingleTest() {
	seed = time.Now().Unix()
	mrand.Seed(seed)
}

func TestBasic(t *testing.T) {
	w := New(nil, nil)
	p := w.Protocols()
	shh := p[0]
	if shh.Name != v0.Name {
		t.Fatalf("failed Peer Name: %v.", shh.Name)
	}
	if uint64(shh.Version) != v0.Version {
		t.Fatalf("failed Peer Version: %v.", shh.Version)
	}
	if shh.Length != v0.NumberOfMessageCodes {
		t.Fatalf("failed Peer Length: %v.", shh.Length)
	}
	if shh.Run == nil {
		t.Fatalf("failed shh.Run.")
	}
	if uint64(w.Version()) != v0.Version {
		t.Fatalf("failed waku Version: %v.", shh.Version)
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

	derived := pbkdf2.Key(peerID, nil, 65356, common.AESKeyLength, sha256.New)
	if !common.ValidateDataIntegrity(derived, common.AESKeyLength) {
		t.Fatalf("failed validateSymmetricKey with param = %v.", derived)
	}
	if common.ContainsOnlyZeros(derived) {
		t.Fatalf("failed containsOnlyZeros with param = %v.", derived)
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := common.BytesToUintLittleEndian(buf)
	be := common.BytesToUintBigEndian(buf)
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
	if !common.ValidatePublicKey(&pk.PublicKey) {
		t.Fatalf("failed ValidatePublicKey: %v.", pk)
	}
}

func TestAsymmetricKeyImport(t *testing.T) {
	var (
		w           = New(nil, nil)
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

func TestWakuIdentityManagement(t *testing.T) {
	w := New(nil, nil)
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

func TestSymKeyManagement(t *testing.T) {
	InitSingleTest()

	var err error
	var k1, k2 []byte
	w := New(nil, nil)
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
	randomKey := make([]byte, common.AESKeyLength)
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
	if len(k1) != common.AESKeyLength {
		t.Fatalf("wrong length of k1.")
	}
	if len(k2) != common.AESKeyLength {
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

	randomKey = make([]byte, common.AESKeyLength+1)
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
	if !common.ValidateDataIntegrity(k2, common.AESKeyLength) {
		t.Fatalf("key validation failed.")
	}
	if !bytes.Equal(k1, k2) {
		t.Fatalf("k1 != k2.")
	}
}

func TestExpiry(t *testing.T) {
	InitSingleTest()

	w := New(nil, nil)
	err := w.SetMinimumPoW(0.0000001, false)
	if err != nil {
		t.Fatal("failed to set min pow")
	}

	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	err = w.Start(nil)
	if err != nil {
		t.Fatal("failed to start waku")
	}
	defer func() {
		handleError(t, w.Stop())
	}()

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
		msg, err := common.NewSentMessage(params)
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

	w := New(nil, nil)
	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	defer func() {
		handleError(t, w.SetMaxMessageSize(common.DefaultMaxMessageSize))
	}()
	if err := w.Start(nil); err != nil {
		t.Fatal("failed to start node")
	}
	defer func() {
		handleError(t, w.Stop())
	}()

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
	params.Topic = common.BytesToTopic(f.Topics[2])
	params.PoW = smallPoW
	params.TTL = 3600 * 24 // one day
	msg, err := common.NewSentMessage(params)
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

	_ = w.SetMinimumPoW(smallPoW/2, true)
	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}

	params.TTL++
	msg, err = common.NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err = msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}
	_ = w.SetMaxMessageSize(uint32(env.Size() - 1))
	err = w.Send(env)
	if err == nil {
		t.Fatalf("successfully sent oversized envelope (seed %d): false positive.", seed)
	}

	_ = w.SetMaxMessageSize(common.DefaultMaxMessageSize)
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

	w := New(nil, nil)
	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	defer func() {
		handleError(t, w.SetMaxMessageSize(common.DefaultMaxMessageSize))
	}()
	err := w.Start(nil)
	if err != nil {
		t.Fatal("failed to start node")
	}
	defer func() {
		handleError(t, w.Stop())
	}()

	filter1, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter1.PoW = common.DefaultMinimumPoW

	// Copy the first filter since some of its fields
	// are randomly generated.
	filter2 := &common.Filter{
		KeySym:   filter1.KeySym,
		Topics:   filter1.Topics,
		PoW:      filter1.PoW,
		AllowP2P: filter1.AllowP2P,
		Messages: common.NewMemoryMessageStore(),
	}

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter1.Src = &params.Src.PublicKey
	filter2.Src = &params.Src.PublicKey

	params.KeySym = filter1.KeySym
	params.Topic = common.BytesToTopic(filter1.Topics[2])
	params.PoW = filter1.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := common.NewSentMessage(params)
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

func TestSymmetricSendCycleWithTopicInterest(t *testing.T) {
	InitSingleTest()

	w := New(nil, nil)
	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	defer func() {
		handleError(t, w.SetMaxMessageSize(common.DefaultMaxMessageSize))
	}()
	if err := w.Start(nil); err != nil {
		t.Fatal("could not start node")
	}
	defer func() {
		handleError(t, w.Stop())
	}()

	filter1, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter1.PoW = common.DefaultMinimumPoW

	// Copy the first filter since some of its fields
	// are randomly generated.
	filter2 := &common.Filter{
		KeySym:   filter1.KeySym,
		Topics:   filter1.Topics,
		PoW:      filter1.PoW,
		AllowP2P: filter1.AllowP2P,
		Messages: common.NewMemoryMessageStore(),
	}

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter1.Src = &params.Src.PublicKey
	filter2.Src = &params.Src.PublicKey

	params.KeySym = filter1.KeySym
	params.Topic = common.BytesToTopic(filter1.Topics[2])
	params.PoW = filter1.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := common.NewSentMessage(params)
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

	w := New(nil, nil)
	if err := w.Start(nil); err != nil {
		t.Errorf("failed to start waku: '%s'", err)
	}

	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	defer func() {
		handleError(t, w.SetMaxMessageSize(common.DefaultMaxMessageSize))
	}()
	defer func() {
		handleError(t, w.Stop())
	}()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = common.DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter.Src = nil

	params.KeySym = filter.KeySym
	params.Topic = common.BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := common.NewSentMessage(params)
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

	w := New(nil, nil)
	if err := w.Start(nil); err != nil {
		t.Errorf("failed to start waku: '%s'", err)
	}
	defer func() {
		handleError(t, w.SetMinimumPoW(common.DefaultMinimumPoW, false))
	}()
	defer func() {
		handleError(t, w.SetMaxMessageSize(common.DefaultMaxMessageSize))
	}()
	defer func() {
		handleError(t, w.Stop())
	}()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = common.DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.KeySym = filter.KeySym
	params.Topic = common.BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := common.NewSentMessage(params)
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
	topic := common.TopicType{0, 0, 255, 6}
	b := common.TopicToBloom(topic)
	x := make([]byte, common.BloomFilterSize)
	x[0] = byte(1)
	x[32] = byte(1)
	x[common.BloomFilterSize-1] = byte(128)
	if !common.BloomFilterMatch(x, b) || !common.BloomFilterMatch(b, x) {
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
	if !common.BloomFilterMatch(b, b) {
		t.Fatalf("bloom filter does not match self")
	}
	x = addBloom(x, b)
	if !common.BloomFilterMatch(x, b) {
		t.Fatalf("bloom filter does not match combined bloom")
	}
	if !common.IsFullNode(nil) {
		t.Fatalf("common.IsFullNode did not recognize nil as full node")
	}
	x[17] = 254
	if common.IsFullNode(x) {
		t.Fatalf("common.IsFullNode false positive")
	}
	for i := 0; i < common.BloomFilterSize; i++ {
		b[i] = byte(255)
	}
	if !common.IsFullNode(b) {
		t.Fatalf("common.IsFullNode false negative")
	}
	if common.BloomFilterMatch(x, b) {
		t.Fatalf("bloomFilterMatch false positive")
	}
	if !common.BloomFilterMatch(b, x) {
		t.Fatalf("bloomFilterMatch false negative")
	}

	w := New(nil, nil)
	f := w.BloomFilter()
	if f != nil {
		t.Fatalf("wrong bloom on creation")
	}
	err = w.SetBloomFilter(x)
	if err != nil {
		t.Fatalf("failed to set bloom filter: %s", err)
	}
	f = w.BloomFilter()
	if !common.BloomFilterMatch(f, x) || !common.BloomFilterMatch(x, f) {
		t.Fatalf("retireved wrong bloom filter")
	}
}

func TestTopicInterest(t *testing.T) {
	w := New(nil, nil)
	topicInterest := w.TopicInterest()
	if topicInterest != nil {
		t.Fatalf("wrong topic on creation")
	}

	filter1, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter1)
	if err != nil {
		t.Fatalf("failed subscribe with seed %d: %s.", seed, err)
	}

	topicInterest = w.TopicInterest()
	if len(topicInterest) != len(filter1.Topics) {
		t.Fatalf("wrong number of topics created")
	}

	filter2, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter2)
	if err != nil {
		t.Fatalf("failed subscribe with seed %d: %s.", seed, err)
	}

	topicInterest = w.TopicInterest()
	if len(topicInterest) != len(filter1.Topics)+len(filter2.Topics) {
		t.Fatalf("wrong number of topics created")
	}

}

// TODO: Fix this to use protcol instead of stubbing
func TestHandleP2PMessageCode(t *testing.T) {
	InitSingleTest()

	w1 := New(nil, nil)
	if err := w1.SetMinimumPoW(0.0000001, false); err != nil {
		t.Error(err)
	}
	if err := w1.Start(nil); err != nil {
		t.Error(err)
	}

	defer func() {
		handleError(t, w1.Stop())
	}()

	w2 := New(nil, nil)
	if err := w2.SetMinimumPoW(0.0000001, false); err != nil {
		t.Error(err)
	}
	if err := w2.Start(nil); err != nil {
		t.Error(err)
	}
	defer func() {
		handleError(t, w2.Stop())
	}()

	envelopeEvents := make(chan common.EnvelopeEvent, 10)
	sub := w1.SubscribeEnvelopeEvents(envelopeEvents)
	defer sub.Unsubscribe()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params.TTL = 1

	msg, err := common.NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	rw1, rw2 := p2p.MsgPipe()

	errorc := make(chan error, 1)
	go func() {
		err := w1.HandlePeer(p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw1)
		errorc <- err
	}()
	go func() {
		select {
		case err := <-errorc:
			t.Log(err)
		case <-time.After(time.Second * 5):
			if err := rw1.Close(); err != nil {
				t.Error(err)
			}
			if err := rw2.Close(); err != nil {
				t.Error(err)
			}
		}
	}()

	peer1 := v0.NewPeer(w2, p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{}), rw2, nil)
	peer1.SetPeerTrusted(true)

	err = peer1.Start()
	require.NoError(t, err, "failed run message loop")

	// Simulate receiving the new envelope
	_, err = w2.add(env, true)
	require.NoError(t, err)

	if e := <-envelopeEvents; e.Hash != env.Hash() {
		t.Fatalf("received envelope %s while expected %s", e.Hash, env.Hash())
	}
	peer1.Stop()
}

func testConfirmationsHandshake(t *testing.T, expectConfirmations bool) {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		EnableConfirmations: expectConfirmations,
	}
	w := New(conf, nil)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	// so that actual read won't hang forever
	time.AfterFunc(5*time.Second, func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
	})
	require.NoError(
		t,
		p2p.ExpectMsg(
			rw1,
			v0.StatusCode,
			[]interface{}{
				v0.Version,
				v0.StatusOptionsFromHost(w),
			},
		),
	)
}

func TestConfirmationHadnshakeExtension(t *testing.T) {
	testConfirmationsHandshake(t, true)
}

func TestHandshakeWithConfirmationsDisabled(t *testing.T) {
	testConfirmationsHandshake(t, false)
}

func TestConfirmationReceived(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	conf := &Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      10 << 20,
		EnableConfirmations: true,
	}
	w := New(conf, logger)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	go func() {
		select {
		case err := <-errorc:
			t.Log(err)
		case <-time.After(time.Second * 5):
			if err := rw1.Close(); err != nil {
				t.Errorf("error closing MsgPipe, '%s'", err)
			}
		}
	}()
	pow := math.Float64bits(w.MinPow())
	confirmationsEnabled := true
	lightNodeEnabled := true
	require.NoError(
		t,
		p2p.ExpectMsg(
			rw1,
			v0.StatusCode,
			[]interface{}{
				v0.Version,
				v0.StatusOptionsFromHost(w),
			},
		),
	)
	require.NoError(
		t,
		p2p.SendItems(
			rw1,
			v0.StatusCode,
			v0.Version,
			v0.StatusOptions{
				PoWRequirement:       &pow,
				BloomFilter:          w.BloomFilter(),
				ConfirmationsEnabled: &confirmationsEnabled,
				LightNodeEnabled:     &lightNodeEnabled,
			},
		),
	)

	e := common.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	data, err := rlp.EncodeToBytes([]*common.Envelope{&e})
	require.NoError(t, err)
	hash := crypto.Keccak256Hash(data)
	require.NoError(t, p2p.SendItems(rw1, v0.MessagesCode, &e))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.MessageResponseCode, nil))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.BatchAcknowledgedCode, hash))
}

func TestMessagesResponseWithError(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      10 << 20,
		EnableConfirmations: true,
	}
	w := New(conf, nil)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}})
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe 1, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			t.Errorf("error closing MsgPipe 2, '%s'", err)
		}
	}()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()

	pow := math.Float64bits(w.MinPow())
	confirmationsEnabled := true
	lightNodeEnabled := true
	require.NoError(
		t,
		p2p.ExpectMsg(
			rw1,
			v0.StatusCode,
			[]interface{}{
				v0.Version,
				v0.StatusOptionsFromHost(w),
			},
		),
	)
	require.NoError(
		t,
		p2p.SendItems(
			rw1,
			v0.StatusCode,
			v0.Version,
			v0.StatusOptions{
				PoWRequirement:       &pow,
				BloomFilter:          w.BloomFilter(),
				ConfirmationsEnabled: &confirmationsEnabled,
				LightNodeEnabled:     &lightNodeEnabled,
			},
		),
	)

	failed := common.Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	normal := common.Envelope{
		Expiry: uint32(time.Now().Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}

	data, err := rlp.EncodeToBytes([]*common.Envelope{&failed, &normal})
	require.NoError(t, err)
	hash := crypto.Keccak256Hash(data)
	require.NoError(t, p2p.SendItems(rw1, v0.MessagesCode, &failed, &normal))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.MessageResponseCode, v0.NewMessagesResponse(hash, []common.EnvelopeError{
		{Hash: failed.Hash(), Code: common.EnvelopeTimeNotSynced, Description: "envelope from future"},
	})))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.BatchAcknowledgedCode, hash))
}

func testConfirmationEvents(t *testing.T, envelope common.Envelope, envelopeErrors []common.EnvelopeError) {
	conf := &Config{
		MinimumAcceptedPoW:  0,
		MaxMessageSize:      10 << 20,
		EnableConfirmations: true,
	}
	w := New(conf, nil)
	events := make(chan common.EnvelopeEvent, 2)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	time.AfterFunc(5*time.Second, func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
	})

	pow := math.Float64bits(w.MinPow())
	confirmationsEnabled := true
	lightNodeEnabled := true

	require.NoError(t, p2p.ExpectMsg(
		rw1,
		v0.StatusCode,
		[]interface{}{
			v0.Version,
			v0.StatusOptionsFromHost(w),
		},
	))
	require.NoError(t, p2p.SendItems(
		rw1,
		v0.StatusCode,
		v0.Version,
		v0.StatusOptions{
			PoWRequirement:       &pow,
			BloomFilter:          w.BloomFilter(),
			ConfirmationsEnabled: &confirmationsEnabled,
			LightNodeEnabled:     &lightNodeEnabled,
		},
	))
	require.NoError(t, w.Send(&envelope))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.MessagesCode, []*common.Envelope{&envelope}))

	var hash gethcommon.Hash
	select {
	case ev := <-events:
		require.Equal(t, common.EventEnvelopeSent, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.NotEqual(t, gethcommon.Hash{}, ev.Batch)
		hash = ev.Batch
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for an envelope.sent event")
	}
	require.NoError(t, p2p.Send(rw1, v0.MessageResponseCode, v0.NewMessagesResponse(hash, envelopeErrors)))
	require.NoError(t, p2p.Send(rw1, v0.BatchAcknowledgedCode, hash))
	select {
	case ev := <-events:
		require.Equal(t, common.EventBatchAcknowledged, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.Equal(t, hash, ev.Batch)
		require.Equal(t, envelopeErrors, ev.Data)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for an batch.acknowledged event")
	}
}

func TestConfirmationEventsReceived(t *testing.T) {
	e := common.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	testConfirmationEvents(t, e, []common.EnvelopeError{})
}

func TestConfirmationEventsExtendedWithErrors(t *testing.T) {
	e := common.Envelope{
		Expiry: uint32(time.Now().Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	testConfirmationEvents(t, e, []common.EnvelopeError{
		{
			Hash:        e.Hash(),
			Code:        common.EnvelopeTimeNotSynced,
			Description: "test error",
		}},
	)
}

func TestEventsWithoutConfirmation(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPoW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf, nil)
	events := make(chan common.EnvelopeEvent, 2)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}})
	rw1, rw2 := p2p.MsgPipe()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err
	}()
	time.AfterFunc(5*time.Second, func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
	})

	pow := math.Float64bits(w.MinPow())
	lightNodeEnabled := true

	require.NoError(
		t,
		p2p.ExpectMsg(
			rw1,
			v0.StatusCode,
			[]interface{}{
				v0.Version,
				v0.StatusOptionsFromHost(w),
			},
		),
	)
	require.NoError(
		t,
		p2p.SendItems(
			rw1,
			v0.StatusCode,
			v0.Version,
			v0.StatusOptions{
				PoWRequirement:   &pow,
				BloomFilter:      w.BloomFilter(),
				LightNodeEnabled: &lightNodeEnabled,
			},
		),
	)

	e := common.Envelope{
		Expiry: uint32(time.Now().Add(10 * time.Second).Unix()),
		TTL:    10,
		Topic:  common.TopicType{1},
		Data:   make([]byte, 1<<10),
		Nonce:  1,
	}
	require.NoError(t, w.Send(&e))
	require.NoError(t, p2p.ExpectMsg(rw1, v0.MessagesCode, []*common.Envelope{&e}))

	select {
	case ev := <-events:
		require.Equal(t, common.EventEnvelopeSent, ev.Event)
		require.Equal(t, p.ID(), ev.Peer)
		require.Equal(t, gethcommon.Hash{}, ev.Batch)
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
			msg.Discard() // nolint: errcheck
		}
	}()
	return rw2
}

func TestWakuTimeDesyncEnvelopeIgnored(t *testing.T) {
	c := &Config{
		MaxMessageSize:     common.DefaultMaxMessageSize,
		MinimumAcceptedPoW: 0,
	}
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
	}()
	p1 := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	p2 := p2p.NewPeer(enode.ID{2}, "2", []p2p.Cap{{"shh", 6}})
	w1, w2 := New(c, nil), New(c, nil)
	errc := make(chan error)
	go func() {
		if err := w1.HandlePeer(p2, rw2); err != nil {
			t.Errorf("error handling peer, '%s'", err)
		}
	}()
	go func() {
		errc <- w2.HandlePeer(p1, rw1)
	}()
	w1.SetTimeSource(func() time.Time {
		return time.Now().Add(time.Hour)
	})
	env := &common.Envelope{
		Expiry: uint32(time.Now().Add(time.Hour).Unix()),
		TTL:    30,
		Topic:  common.TopicType{1},
		Data:   []byte{1, 1, 1},
	}
	require.NoError(t, w1.Send(env))
	select {
	case err := <-errc:
		require.NoError(t, err)
	case <-time.After(time.Second):
	}
	if err := rw2.Close(); err != nil {
		t.Errorf("error closing MsgPipe, '%s'", err)
	}
	select {
	case err := <-errc:
		require.Error(t, err, "p2p: read or write on closed message pipe")
	case <-time.After(time.Second):
		require.FailNow(t, "connection wasn't closed in expected time")
	}
}

func TestRequestSentEventWithExpiry(t *testing.T) {
	w := New(nil, nil)
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"shh", 6}})
	rw := discardPipe()
	defer func() {
		handleError(t, rw.Close())
	}()
	w.peers[v0.NewPeer(w, p, rw, nil)] = struct{}{}
	events := make(chan common.EnvelopeEvent, 1)
	sub := w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	e := &common.Envelope{Nonce: 1}
	require.NoError(t, w.RequestHistoricMessagesWithTimeout(p.ID().Bytes(), e, time.Millisecond))
	verifyEvent := func(etype common.EventType) {
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "error waiting for a event type %s", etype)
		case ev := <-events:
			require.Equal(t, etype, ev.Event)
			require.Equal(t, p.ID(), ev.Peer)
			require.Equal(t, e.Hash(), ev.Hash)
		}
	}
	verifyEvent(common.EventMailServerRequestSent)
	verifyEvent(common.EventMailServerRequestExpired)
}

func TestSendMessagesRequest(t *testing.T) {
	validMessagesRequest := common.MessagesRequest{
		ID:    make([]byte, 32),
		From:  0,
		To:    10,
		Bloom: []byte{0x01},
	}

	t.Run("InvalidID", func(t *testing.T) {
		w := New(nil, nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, common.MessagesRequest{})
		require.EqualError(t, err, "invalid 'ID', expected a 32-byte slice")
	})

	t.Run("WithoutPeer", func(t *testing.T) {
		w := New(nil, nil)
		err := w.SendMessagesRequest([]byte{0x01, 0x02}, validMessagesRequest)
		require.EqualError(t, err, "could not find peer with ID: 0102")
	})

	t.Run("AllGood", func(t *testing.T) {
		p := p2p.NewPeer(enode.ID{0x01}, "peer01", nil)
		rw1, rw2 := p2p.MsgPipe()
		w := New(nil, nil)
		w.peers[v0.NewPeer(w, p, rw1, nil)] = struct{}{}

		go func() {
			err := w.SendMessagesRequest(p.ID().Bytes(), validMessagesRequest)
			require.NoError(t, err)
		}()

		require.NoError(t, p2p.ExpectMsg(rw2, v0.P2PRequestCode, nil))
	})
}

func TestRateLimiterIntegration(t *testing.T) {
	conf := &Config{
		MinimumAcceptedPoW: 0,
		MaxMessageSize:     10 << 20,
	}
	w := New(conf, nil)
	w.RegisterRateLimiter(common.NewPeerRateLimiter(nil, &common.MetricsRateLimiterHandler{}))
	p := p2p.NewPeer(enode.ID{1}, "1", []p2p.Cap{{"waku", 0}})
	rw1, rw2 := p2p.MsgPipe()
	defer func() {
		if err := rw1.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
		if err := rw2.Close(); err != nil {
			t.Errorf("error closing MsgPipe, '%s'", err)
		}
	}()
	errorc := make(chan error, 1)
	go func() {
		err := w.HandlePeer(p, rw2)
		errorc <- err

	}()

	require.NoError(
		t,
		p2p.ExpectMsg(
			rw1,
			v0.StatusCode,
			[]interface{}{
				v0.Version,
				v0.StatusOptionsFromHost(w),
			},
		),
	)
	select {
	case err := <-errorc:
		require.NoError(t, err)
	default:
	}
}

func TestMailserverCompletionEvent(t *testing.T) {
	w1 := New(nil, nil)
	require.NoError(t, w1.Start(nil))
	defer func() {
		handleError(t, w1.Stop())
	}()

	rw1, rw2 := p2p.MsgPipe()
	peer1 := v0.NewPeer(w1, p2p.NewPeer(enode.ID{1}, "1", nil), rw1, nil)
	peer1.SetPeerTrusted(true)
	w1.peers[peer1] = struct{}{}

	w2 := New(nil, nil)
	require.NoError(t, w2.Start(nil))
	defer func() {
		handleError(t, w2.Stop())
	}()

	peer2 := v0.NewPeer(w2, p2p.NewPeer(enode.ID{1}, "1", nil), rw2, nil)
	peer2.SetPeerTrusted(true)
	w2.peers[peer2] = struct{}{}

	events := make(chan common.EnvelopeEvent)
	sub := w1.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	envelopes := []*common.Envelope{{Data: []byte{1}}, {Data: []byte{2}}}
	go func() {
		require.NoError(t, peer2.Start())
		require.NoError(t, p2p.Send(rw2, v0.P2PMessageCode, envelopes))
		require.NoError(t, p2p.Send(rw2, v0.P2PRequestCompleteCode, [100]byte{})) // 2 hashes + cursor size
		require.NoError(t, rw2.Close())
	}()

	require.NoError(t, peer1.Start(), "p2p: read or write on closed message pipe")
	require.EqualError(t, peer1.Run(), "p2p: read or write on closed message pipe")

	after := time.After(2 * time.Second)
	count := 0
	for {
		select {
		case <-after:
			require.FailNow(t, "timed out waiting for all events")
		case ev := <-events:
			switch ev.Event {
			case common.EventEnvelopeAvailable:
				count++
			case common.EventMailServerRequestCompleted:
				require.Equal(t, count, len(envelopes),
					"all envelope.available events mut be received before request is completed")
				return
			}
		}
	}
}

func handleError(t *testing.T, err error) {
	if err != nil {
		t.Logf("deferred function error: '%s'", err)
	}
}

func generateFilter(t *testing.T, symmetric bool) (*common.Filter, error) {
	var f common.Filter
	f.Messages = common.NewMemoryMessageStore()

	const topicNum = 8
	f.Topics = make([][]byte, topicNum)
	for i := 0; i < topicNum; i++ {
		f.Topics[i] = make([]byte, 4)
		mrand.Read(f.Topics[i]) // nolint: gosec
		f.Topics[i][0] = 0x01
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generateFilter 1 failed with seed %d.", seed)
		return nil, err
	}
	f.Src = &key.PublicKey

	if symmetric {
		f.KeySym = make([]byte, common.AESKeyLength)
		mrand.Read(f.KeySym) // nolint: gosec
		f.SymKeyHash = crypto.Keccak256Hash(f.KeySym)
	} else {
		f.KeyAsym, err = crypto.GenerateKey()
		if err != nil {
			t.Fatalf("generateFilter 2 failed with seed %d.", seed)
			return nil, err
		}
	}

	// AcceptP2P & PoW are not set
	return &f, nil
}

func generateMessageParams() (*common.MessageParams, error) {
	// set all the parameters except p.Dst and p.Padding

	buf := make([]byte, 4)
	mrand.Read(buf) // nolint: gosec
	sz := mrand.Intn(400)

	var p common.MessageParams
	p.PoW = 0.01
	p.WorkTime = 1
	p.TTL = uint32(mrand.Intn(1024))
	p.Payload = make([]byte, sz)
	p.KeySym = make([]byte, common.AESKeyLength)
	mrand.Read(p.Payload) // nolint: gosec
	mrand.Read(p.KeySym)  // nolint: gosec
	p.Topic = common.BytesToTopic(buf)

	var err error
	p.Src, err = crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &p, nil
}
