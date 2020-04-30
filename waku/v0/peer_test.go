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

package v0

import (
	mrand "math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/waku/common"
)

var seed int64

// initSingleTest should be called in the beginning of every
// test, which uses RNG, in order to make the tests
// reproduciblity independent of their sequence.
func initSingleTest() {
	seed = time.Now().Unix()
	mrand.Seed(seed)
}

var sharedTopic = common.TopicType{0xF, 0x1, 0x2, 0}
var wrongTopic = common.TopicType{0, 0, 0, 0}

//two generic waku node handshake. one don't send light flag
func TestTopicOrBloomMatch(t *testing.T) {
	p := Peer{}
	p.setTopicInterest([]common.TopicType{sharedTopic})
	envelope := &common.Envelope{Topic: sharedTopic}
	if !p.topicOrBloomMatch(envelope) {
		t.Fatal("envelope should match")
	}

	badEnvelope := &common.Envelope{Topic: wrongTopic}
	if p.topicOrBloomMatch(badEnvelope) {
		t.Fatal("envelope should not match")
	}

}

func TestTopicOrBloomMatchFullNode(t *testing.T) {
	p := Peer{}
	// Set as full node
	p.fullNode = true
	p.setTopicInterest([]common.TopicType{sharedTopic})
	envelope := &common.Envelope{Topic: sharedTopic}
	if !p.topicOrBloomMatch(envelope) {
		t.Fatal("envelope should match")
	}

	badEnvelope := &common.Envelope{Topic: wrongTopic}
	if p.topicOrBloomMatch(badEnvelope) {
		t.Fatal("envelope should not match")
	}
}

func TestPeerBasic(t *testing.T) {
	initSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d.", seed)
	}

	params.PoW = 0.001
	msg, err := common.NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		t.Fatalf("failed Wrap with seed %d.", seed)
	}

	p := NewPeer(nil, nil, nil, nil)
	p.Mark(env)
	if !p.Marked(env) {
		t.Fatalf("failed mark with seed %d.", seed)
	}
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
