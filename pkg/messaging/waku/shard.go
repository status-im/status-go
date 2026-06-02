package wakuv2

import (
	wakuproto "github.com/waku-org/go-waku/waku/v2/protocol"
)

type Shard struct {
	Cluster uint16 `json:"cluster"`
	Index   uint16 `json:"index"`
}

func (s *Shard) PubsubTopic() string {
	if s != nil {
		return wakuproto.NewStaticShardingPubsubTopic(s.Cluster, s.Index).String()
	}
	return ""
}

const MainStatusShardCluster = 16
const DefaultShardIndex = 32

func DefaultShardPubsubTopic() string {
	return wakuproto.NewStaticShardingPubsubTopic(MainStatusShardCluster, DefaultShardIndex).String()
}
