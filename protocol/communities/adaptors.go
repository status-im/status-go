package communities

import (
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToSyncCommunityProtobuf(clock uint64, communitySettings *CommunitySettings) (*protobuf.SyncCommunity, error) {
	var pkb []byte
	pk := o.PrivateKey()
	if pk != nil {
		pkb = crypto.FromECDSA(pk)
	}

	md, err := o.ToBytes()
	if err != nil {
		return nil, err
	}

	var rtjs []*protobuf.SyncCommunityRequestsToJoin
	reqs := o.RequestsToJoin()
	for _, req := range reqs {
		rtjs = append(rtjs, req.ToSyncProtobuf())
	}

	settings := &protobuf.SyncCommunitySettings{
		Clock:                        clock,
		CommunityId:                  o.IDString(),
		HistoryArchiveSupportEnabled: true,
	}

	if communitySettings != nil {
		settings.HistoryArchiveSupportEnabled = communitySettings.HistoryArchiveSupportEnabled
	}

	return &protobuf.SyncCommunity{
		Clock:          clock,
		Id:             o.ID(),
		PrivateKey:     pkb,
		Description:    md,
		Joined:         o.Joined(),
		Verified:       o.Verified(),
		Muted:          o.Muted(),
		RequestsToJoin: rtjs,
		Settings:       settings,
	}, nil
}
