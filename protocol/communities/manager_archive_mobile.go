//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ArchiveManagerMobile struct{}

func (amm *ArchiveManagerMobile) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerMobile) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerMobile) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return nil
}

func (amm *ArchiveManagerMobile) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return nil, nil
}

func (amm *ArchiveManagerMobile) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return nil
}

func (amm *ArchiveManagerMobile) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	return nil, nil
}

func (amm *ArchiveManagerMobile) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	return "", nil
}

func (amm *ArchiveManagerMobile) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	return nil, nil
}
