package wakuv2

import (
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/status-im/status-go/protocol/protobuf"
)

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
}

func FromProtobuff(p *protobuf.Shard) *Shard {
	if p == nil {
		return nil
	}

	return &Shard{
		Cluster: uint16(p.Cluster),
		Index:   uint16(p.Index),
	}
}

func (s *Shard) Protobuffer() *protobuf.Shard {
	if s == nil {
		return nil
	}

	return &protobuf.Shard{
		Cluster: int32(s.Cluster),
		Index:   int32(s.Index),
	}
}
func (s *Shard) PubsubTopic() string {
	if s != nil {
		return wakuproto.NewStaticShardingPubsubTopic(s.Cluster, s.Index).String()
	}
	return ""
}

const MainStatusShardCluster = 16
const DefaultShardIndex = 32
const NonProtectedShardIndex = 64

func DefaultShardPubsubTopic() string {
	return wakuproto.NewStaticShardingPubsubTopic(MainStatusShardCluster, DefaultShardIndex).String()
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

func DefaultShard() *Shard {
	return &Shard{
		Cluster: MainStatusShardCluster,
		Index:   DefaultShardIndex,
	}
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
