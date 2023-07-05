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

package wakuv2

import (
	"bytes"
	"testing"
	"time"

	"github.com/status-im/status-go/wakuv2/common"
)

func TestMultipleTopicCopyInNewMessageFilter(t *testing.T) {
	w, err := New("", "", nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Error creating WakuV2 client: %v", err)
	}

	keyID, err := w.GenerateSymKey()
	if err != nil {
		t.Fatalf("Error generating symmetric key: %v", err)
	}
	api := PublicWakuAPI{
		w:        w,
		lastUsed: make(map[string]time.Time),
	}

	t1 := [4]byte{0xde, 0xea, 0xbe, 0xef}
	t2 := [4]byte{0xca, 0xfe, 0xde, 0xca}

	crit := Criteria{
		SymKeyID: keyID,
		Topics:   []common.TopicType{common.TopicType(t1), common.TopicType(t2)},
	}

	_, err = api.NewMessageFilter(crit)
	if err != nil {
		t.Fatalf("Error creating the filter: %v", err)
	}

	found := false
	candidates := w.filters.GetWatchersByTopic(t1)
	for _, f := range candidates {
		if len(f.Topics) == 2 {
			if bytes.Equal(f.Topics[0], t1[:]) && bytes.Equal(f.Topics[1], t2[:]) {
				found = true
			}
		}
	}

	if !found {
		t.Fatalf("Could not find filter with both topics")
	}
}
