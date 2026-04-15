package types

import (
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
)

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
}

func (s *Shard) PubsubTopic() string {
	if s == nil {
		return ""
	}

	wakuv2Shard := wakuv2.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}

	return wakuv2Shard.PubsubTopic()
}

const MainStatusShardCluster = 16
const DefaultShardIndex = 32
const NonProtectedShardIndex = 64

func DefaultShardPubsubTopic() string {
	return wakuproto.NewStaticShardingPubsubTopic(MainStatusShardCluster, DefaultShardIndex).String()
}

func DefaultShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   DefaultShardIndex,
	}
}

func DefaultNonProtectedShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   NonProtectedShardIndex,
	}
}

// TODO this is used only for community control messages, we need to stop using it once migration is done
func DefaultNonProtectedPubsubTopic() string {
	return DefaultNonProtectedShard().PubsubTopic()
}
