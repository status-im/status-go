//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ArchiveFileManagerNop struct{}

func (amm *ArchiveFileManagerNop) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveFileManagerNop) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveFileManagerNop) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return nil
}

func (amm *ArchiveFileManagerNop) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveFileManagerNop) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return nil
}

func (amm *ArchiveFileManagerNop) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	return nil, nil
}

func (amm *ArchiveFileManagerNop) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	return "", nil
}

func (amm *ArchiveFileManagerNop) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	return nil, nil
}
