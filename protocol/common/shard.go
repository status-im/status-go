package common

import "github.com/status-im/status-go/protocol/transport"

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
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

const MainStatusShard = 16
const UndefinedShardValue = 0
