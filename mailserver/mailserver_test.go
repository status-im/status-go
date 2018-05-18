// Copyright 2017 The go-ethereum Authors
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

package mailserver

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/params"
)

const powRequirement = 0.00001
const peerID = "peerID"

var keyID string
var shh *whisper.Whisper
var seed = time.Now().Unix()

type ServerTestParams struct {
	topic whisper.TopicType
	birth uint32
	low   uint32
	upp   uint32
	key   *ecdsa.PrivateKey
}

func assert(statement bool, text string, t *testing.T) {
	if !statement {
		t.Fatal(text)
	}
}

func TestDBKey(t *testing.T) {
	var h common.Hash
	i := uint32(time.Now().Unix())
	k := NewDbKey(i, h)
	assert(len(k.raw) == common.HashLength+4, "wrong DB key length", t)
	assert(byte(i%0x100) == k.raw[3], "raw representation should be big endian", t)
	assert(byte(i/0x1000000) == k.raw[0], "big endian expected", t)
}

func TestMailServer(t *testing.T) {
	var server WMailServer

	setupServer(t, &server)
	defer server.Close()

	env := generateEnvelope(t, time.Now())
	server.Archive(env)
	deliverTest(t, &server, env)
}

func TestRateLimits(t *testing.T) {
	l := newLimiter(time.Duration(5 * time.Millisecond))
	assert(l.isAllowed(peerID), "Expected limiter not to allow with empty db", t)

	l.db[peerID] = time.Now().Add(time.Duration(-10 * time.Millisecond))
	assert(l.isAllowed(peerID), "Expected limiter to allow with peer on its db", t)

	l.db[peerID] = time.Now().Add(time.Duration(-1 * time.Millisecond))
	assert(!l.isAllowed(peerID), "Expected limiter to not allow with peer on its db", t)
}

func TestRemoveExpiredRateLimits(t *testing.T) {
	l := newLimiter(time.Duration(10) * time.Second)
	l.db[peerID] = time.Now().Add(time.Duration(-10) * time.Second)
	l.db[peerID+"A"] = time.Now().Add(time.Duration(10) * time.Second)
	l.deleteExpired()
	_, ok := l.db[peerID]
	assert(!ok, "Expired peer should not exist, but it does ", t)
	_, ok = l.db[peerID+"A"]
	assert(ok, "Non expired peer should exist, but it doesn't", t)
}

func generateEnvelope(t *testing.T, now time.Time) *whisper.Envelope {
	h := crypto.Keccak256Hash([]byte("test sample data"))
	params := &whisper.MessageParams{
		KeySym:   h[:],
		Topic:    whisper.TopicType{0x1F, 0x7E, 0xA1, 0x7F},
		Payload:  []byte("test payload"),
		PoW:      powRequirement,
		WorkTime: 2,
	}

	msg, err := whisper.NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, now)
	if err != nil {
		t.Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}

func serverParams(t *testing.T, env *whisper.Envelope) *ServerTestParams {
	id, err := shh.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair with seed %d: %s.", seed, err)
	}
	testPeerID, err := shh.GetPrivateKey(id)
	if err != nil {
		t.Fatalf("failed to retrieve new key pair with seed %d: %s.", seed, err)
	}
	birth := env.Expiry - env.TTL

	return &ServerTestParams{
		topic: env.Topic,
		birth: birth,
		low:   birth - 1,
		upp:   birth + 1,
		key:   testPeerID,
	}
}
func deliverTest(t *testing.T, server *WMailServer, env *whisper.Envelope) {
	p := serverParams(t, env)
	singleRequest(t, server, env, p, true)

	p.low = p.birth + 1
	p.upp = p.birth + 1
	singleRequest(t, server, env, p, false)

	p.low = p.birth
	p.upp = p.birth + 1
	p.topic[0] = 0xFF
	singleRequest(t, server, env, p, false)

	p.low = 0
	p.upp = p.birth - 1
	failRequest(t, server, p, "validation should fail due to negative query time range")

	p.low = 0
	p.upp = p.birth + 24
	failRequest(t, server, p, "validation should fail due to query big time range")
}

func failRequest(t *testing.T, server *WMailServer, p *ServerTestParams, err string) {
	request := createRequest(t, p)
	src := crypto.FromECDSAPub(&p.key.PublicKey)
	ok, _, _, _ := server.validateRequest(src, request)
	if ok {
		t.Fatalf(err)
	}
}

func singleRequest(t *testing.T, server *WMailServer, env *whisper.Envelope, p *ServerTestParams, expect bool) {
	request := createRequest(t, p)
	src := crypto.FromECDSAPub(&p.key.PublicKey)
	ok, lower, upper, bloom := server.validateRequest(src, request)
	if !ok {
		t.Fatalf("request validation failed, seed: %d.", seed)
	}
	if lower != p.low {
		t.Fatalf("request validation failed (lower bound), seed: %d.", seed)
	}
	if upper != p.upp {
		t.Fatalf("request validation failed (upper bound), seed: %d.", seed)
	}
	expectedBloom := whisper.TopicToBloom(p.topic)
	if !bytes.Equal(bloom, expectedBloom) {
		t.Fatalf("request validation failed (topic), seed: %d.", seed)
	}

	var exist bool
	mail := server.processRequest(nil, p.low, p.upp, bloom)
	for _, msg := range mail {
		if msg.Hash() == env.Hash() {
			exist = true
			break
		}
	}

	if exist != expect {
		t.Fatalf("error: exist = %v, seed: %d.", exist, seed)
	}

	src[0]++
	ok, lower, upper, _ = server.validateRequest(src, request)
	if !ok {
		// request should be valid regardless of signature
		t.Fatalf("request validation false negative, seed: %d (lower: %d, upper: %d).", seed, lower, upper)
	}
}

func createRequest(t *testing.T, p *ServerTestParams) *whisper.Envelope {
	bloom := whisper.TopicToBloom(p.topic)
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data, p.low)
	binary.BigEndian.PutUint32(data[4:], p.upp)
	data = append(data, bloom...)

	key, err := shh.GetSymKey(keyID)
	if err != nil {
		t.Fatalf("failed to retrieve sym key with seed %d: %s.", seed, err)
	}

	params := &whisper.MessageParams{
		KeySym:   key,
		Topic:    p.topic,
		Payload:  data,
		PoW:      powRequirement * 2,
		WorkTime: 2,
		Src:      p.key,
	}

	msg, err := whisper.NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}

func setupServer(t *testing.T, server *WMailServer) {
	const password = "password_for_this_test"
	const dbPath = "whisper-server-test"

	dir, err := ioutil.TempDir("", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	shh = whisper.New(&whisper.DefaultConfig)
	shh.RegisterServer(server)

	err = server.Init(shh, &params.WhisperConfig{DataDir: dir, Password: password, MinimumPoW: powRequirement})
	if err != nil {
		t.Fatal(err)
	}

	keyID, err = shh.AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}
}
