package requests

import (
	"errors"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type SetCommunityShard struct {
	CommunityID types.HexBytes  `json:"communityId"`
	Shard       *common.Shard   `json:"shard,omitempty"`
	PrivateKey  *types.HexBytes `json:"privateKey,omitempty"`
}

func (s *SetCommunityShard) Validate() error {
	if s == nil {
		return errors.New("invalid request")
	}
	if s.Shard != nil {
		// TODO: for now only MainStatusShard(16) is accepted
		if s.Shard.Cluster != common.MainStatusShard {
			return errors.New("invalid shard cluster")
		}
		if s.Shard.Index > protocol.MaxShardIndex {
			return fmt.Errorf("invalid shard index. Max index is '%d'", protocol.MaxShardIndex)
		}
	}
	return nil
}
