package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
)

func ToWakuShard(s *types.Shard) *wakuv2.Shard {
	if s == nil {
		return nil
	}
	return &wakuv2.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}
}
