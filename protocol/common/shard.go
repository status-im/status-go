package common

import (
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

const MainStatusShardCluster = 16
const NonProtectedShardIndex = 64
const UndefinedShardValue = 0

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
}

func ShardFromProtobuff(p *protobuf.Shard) *Shard {
	if p == nil {
		return nil
	}

	return &Shard{
		Cluster: uint16(p.Cluster),
		Index:   uint16(p.Index),
	}
}

func (s *Shard) TransportShard() *transport.Shard {
	if s == nil {
		return nil
	}

	return &transport.Shard{
		Cluster: s.Cluster,
		Index:   s.Index,
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

func DefaultNonProtectedPubsubTopic(shard *Shard) string {
	// TODO: remove the condition once DefaultWakuTopic usage
	// is removed
	if shard != nil {
		return transport.GetPubsubTopic(&transport.Shard{
			Cluster: MainStatusShardCluster,
			Index:   NonProtectedShardIndex,
		})
	}

	return relay.DefaultWakuTopic
}
