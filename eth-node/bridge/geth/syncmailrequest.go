package gethbridge

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/whisper"
)

// GetGethSyncMailRequestFrom converts a whisper SyncMailRequest struct from a SyncMailRequest struct
func GetGethSyncMailRequestFrom(r *types.SyncMailRequest) *whisper.SyncMailRequest {
	return &whisper.SyncMailRequest{
		Lower:  r.Lower,
		Upper:  r.Upper,
		Bloom:  r.Bloom,
		Limit:  r.Limit,
		Cursor: r.Cursor,
	}
}
