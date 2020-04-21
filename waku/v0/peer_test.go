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
	"testing"

	"github.com/status-im/status-go/waku/common"
)

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
