package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
)

func FromWakuShard(s *wakuv2.Shard) *types.Shard {
	if s == nil {
		return nil
	}
	return &types.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}
}

func ToWakuShard(s *types.Shard) *wakuv2.Shard {
	if s == nil {
		return nil
	}
	return &wakuv2.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}
}
