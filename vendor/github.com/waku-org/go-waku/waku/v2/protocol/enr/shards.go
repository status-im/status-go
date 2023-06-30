package enr

import (
	"errors"

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

func WithWakuRelaySharding(rs protocol.RelayShards) ENROption {
	return func(localnode *enode.LocalNode) error {
		if len(rs.Indices) >= 64 {
			return WithWakuRelayShardingBitVector(rs)(localnode)
		} else {
			return WithWakuRelayShardingIndicesList(rs)(localnode)
		}
	}
}

func WithWakuRelayShardingTopics(topics ...string) ENROption {
	return func(localnode *enode.LocalNode) error {
		rs, err := protocol.TopicsToRelayShards(topics...)
		if err != nil {
			return err
		}

		if len(rs) != 1 {
			return errors.New("expected a single RelayShards")
		}

		return WithWakuRelaySharding(rs[0])(localnode)
	}
}

// ENR record accessors

func RelayShardingIndicesList(record *enr.Record) (*protocol.RelayShards, error) {
	var field []byte
	if err := record.Load(enr.WithEntry(ShardingIndicesListEnrField, field)); err != nil {
		return nil, nil
	}

	res, err := protocol.FromIndicesList(field)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func RelayShardingBitVector(record *enr.Record) (*protocol.RelayShards, error) {
	var field []byte
	if err := record.Load(enr.WithEntry(ShardingBitVectorEnrField, field)); err != nil {
		if enr.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	res, err := protocol.FromBitVector(field)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func RelaySharding(record *enr.Record) (*protocol.RelayShards, error) {
	res, err := RelayShardingIndicesList(record)
	if err != nil {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	return RelayShardingBitVector(record)
}

// Utils

func ContainsShard(record *enr.Record, cluster uint16, index uint16) bool {
	if index > protocol.MaxShardIndex {
		return false
	}

	rs, err := RelaySharding(record)
	if err != nil {
		return false
	}

	return rs.Contains(cluster, index)
}

func ContainsShardWithNsTopic(record *enr.Record, topic protocol.NamespacedPubsubTopic) bool {
	if topic.Kind() != protocol.StaticSharding {
		return false
	}
	shardTopic := topic.(protocol.StaticShardingPubsubTopic)
	return ContainsShard(record, shardTopic.Cluster(), shardTopic.Shard())
}

func ContainsRelayShard(record *enr.Record, topic protocol.StaticShardingPubsubTopic) bool {
	return ContainsShardWithNsTopic(record, topic)
}

func ContainsShardTopic(record *enr.Record, topic string) bool {
	shardTopic, err := protocol.ToShardedPubsubTopic(topic)
	if err != nil {
		return false
	}
	return ContainsShardWithNsTopic(record, shardTopic)
}
