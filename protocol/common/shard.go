package common

import (
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
)

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

const MainStatusShard = 16
const UndefinedShardValue = 0
