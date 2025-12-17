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

// GlobalCommunityControlShard returns the shard for the global community control messages
//
// Specs: https://github.com/vacp2p/rfc-index/blob/8ee2a6d6b232838d83374c35e2413f84436ecf64/status/56/communities.md?plain=1#L329
func GlobalCommunityControlShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   128,
	}
}

// GlobalCommunityContentShard returns the shard for the global community content messages
//
// Specs: https://github.com/vacp2p/rfc-index/blob/8ee2a6d6b232838d83374c35e2413f84436ecf64/status/56/communities.md?plain=1#L330
func GlobalCommunityContentShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   256,
	}
}

func GlobalCommunityControlPubsubTopic() string {
	return GlobalCommunityControlShard().PubsubTopic()
}

func GlobalCommunityContentPubsubTopic() string {
	return GlobalCommunityContentShard().PubsubTopic()
}
