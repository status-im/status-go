package types

import (
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"

	wakuv "github.com/status-im/status-go/pkg/messaging/waku"
)

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
}

func (s *Shard) PubsubTopic() string {
	if s == nil {
		return ""
	}

	wakuv2Shard := wakuv.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}

	return wakuv2Shard.PubsubTopic()
}

const MainStatusShardCluster = 16
const DefaultShardIndex = 32

func DefaultShardPubsubTopic() string {
	return wakuproto.NewStaticShardingPubsubTopic(MainStatusShardCluster, DefaultShardIndex).String()
}

func DefaultShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   DefaultShardIndex,
	}
}
