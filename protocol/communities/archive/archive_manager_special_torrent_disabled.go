//go:build !disable_history_archives && !use_torrent
// +build !disable_history_archives,!use_torrent

package archive

import (
	"crypto/ecdsa"
	"errors"
	"time"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type torrentBackendForTests interface {
	CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error)
	LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes) (*protobuf.WakuMessageArchiveIndex, error)
}

// GetTorrentBackend returns an error when Torrent support is not built in.
func (m *ArchiveManager) GetTorrentBackend() (torrentBackendForTests, error) {
	return nil, errors.New("backend is not ArchiveManagerTorrent")
}
