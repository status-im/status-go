//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ArchiveManagerNop struct{}

func (amm *ArchiveManagerNop) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerNop) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerNop) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return nil
}

func (amm *ArchiveManagerNop) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerNop) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return nil
}

func (amm *ArchiveManagerNop) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	return nil, nil
}

func (amm *ArchiveManagerNop) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	return "", nil
}

func (amm *ArchiveManagerNop) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	return nil, nil
}
