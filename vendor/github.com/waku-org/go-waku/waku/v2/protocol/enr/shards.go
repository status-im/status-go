package enr

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

func WithWakuRelayShardingIndicesList(rs protocol.RelayShards) ENROption {
	return func(localnode *enode.LocalNode) error {
		value, err := rs.IndicesList()
		if err != nil {
			return err
		}
		localnode.Set(enr.WithEntry(ShardingIndicesListEnrField, value))
		return nil
	}
}

func WithWakuRelayShardingBitVector(rs protocol.RelayShards) ENROption {
	return func(localnode *enode.LocalNode) error {
		localnode.Set(enr.WithEntry(ShardingBitVectorEnrField, rs.BitVector()))
		return nil
	}
}

func WithtWakuRelaySharding(rs protocol.RelayShards) ENROption {
	return func(localnode *enode.LocalNode) error {
		if len(rs.Indices) >= 64 {
			return WithWakuRelayShardingBitVector(rs)(localnode)
		} else {
			return WithWakuRelayShardingIndicesList(rs)(localnode)
		}
	}
}

// ENR record accessors

func RelayShardingIndicesList(localnode *enode.LocalNode) (*protocol.RelayShards, error) {
	var field []byte
	if err := localnode.Node().Record().Load(enr.WithEntry(ShardingIndicesListEnrField, field)); err != nil {
		return nil, nil
	}

	res, err := protocol.FromIndicesList(field)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func RelayShardingBitVector(localnode *enode.LocalNode) (*protocol.RelayShards, error) {
	var field []byte
	if err := localnode.Node().Record().Load(enr.WithEntry(ShardingBitVectorEnrField, field)); err != nil {
		return nil, nil
	}

	res, err := protocol.FromBitVector(field)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func RelaySharding(localnode *enode.LocalNode) (*protocol.RelayShards, error) {
	res, err := RelayShardingIndicesList(localnode)
	if err != nil {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	return RelayShardingBitVector(localnode)
}

// Utils

func ContainsShard(localnode *enode.LocalNode, cluster uint16, index uint16) bool {
	if index > protocol.MaxShardIndex {
		return false
	}

	rs, err := RelaySharding(localnode)
	if err != nil {
		return false
	}

	return rs.Contains(cluster, index)
}

func ContainsShardWithNsTopic(localnode *enode.LocalNode, topic protocol.NamespacedPubsubTopic) bool {
	if topic.Kind() != protocol.StaticSharding {
		return false
	}
	shardTopic := topic.(protocol.StaticShardingPubsubTopic)
	return ContainsShard(localnode, shardTopic.Cluster(), shardTopic.Shard())

}

func ContainsShardTopic(localnode *enode.LocalNode, topic string) bool {
	shardTopic, err := protocol.ToShardedPubsubTopic(topic)
	if err != nil {
		return false
	}
	return ContainsShardWithNsTopic(localnode, shardTopic)
}
