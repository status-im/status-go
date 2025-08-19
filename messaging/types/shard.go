package types

import (
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/wakuv2"
)

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
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
	if s == nil {
		return ""
	}

	wakuv2Shard := wakuv2.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
	}

	return wakuv2Shard.PubsubTopic()
}

func FromShardProtobuff(p *protobuf.Shard) *Shard {
	if p == nil {
		return nil
	}

	return &Shard{
		Cluster: uint16(p.Cluster),
		Index:   uint16(p.Index),
	}
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

func DefaultNonProtectedPubsubTopic() string {
	return DefaultNonProtectedShard().PubsubTopic()
}
