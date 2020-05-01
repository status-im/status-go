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

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/waku/common"
)

func TestEncodeDecodeVersionedResponse(t *testing.T) {
	response := NewMessagesResponse(gethcommon.Hash{1}, []common.EnvelopeError{{Code: 1}})
	bytes, err := rlp.EncodeToBytes(response)
	require.NoError(t, err)

	var mresponse MultiVersionResponse
	require.NoError(t, rlp.DecodeBytes(bytes, &mresponse))
	v1resp, err := mresponse.DecodeResponse1()
	require.NoError(t, err)
	require.Equal(t, response.Response.Hash, v1resp.Hash)
}

func TestSendBundle(t *testing.T) {
	rw1, rw2 := p2p.MsgPipe()
	defer func() { handleError(t, rw1.Close()) }()
	defer func() { handleError(t, rw2.Close()) }()
	envelopes := []*common.Envelope{{
		Expiry: 0,
		TTL:    30,
		Topic:  common.TopicType{1},
		Data:   []byte{1, 1, 1},
	}}

	errc := make(chan error)
	go func() {
		_, err := sendBundle(rw1, envelopes)
		errc <- err
	}()
	require.NoError(t, p2p.ExpectMsg(rw2, messagesCode, envelopes))
	require.NoError(t, <-errc)
}

func handleError(t *testing.T, err error) {
	if err != nil {
		t.Logf("deferred function error: '%s'", err)
	}
}
