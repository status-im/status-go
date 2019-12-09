package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/whisper/v6"
)

// NewGethSyncEventResponseWrapper returns a types.SyncEventResponse object that mimics Geth's SyncEventResponse
func NewGethSyncEventResponseWrapper(syncEventResponse whisper.SyncEventResponse) types.SyncEventResponse {
	return types.SyncEventResponse{
		Cursor: syncEventResponse.Cursor,
		Error:  syncEventResponse.Error,
	}
}
