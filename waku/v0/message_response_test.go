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

	"github.com/stretchr/testify/require"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

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
