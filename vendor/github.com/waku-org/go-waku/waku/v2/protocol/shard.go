package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const MaxShardIndex = uint16(1023)

type RelayShards struct {
	Cluster uint16
	Indices []uint16
}

func NewRelayShards(cluster uint16, indices ...uint16) (RelayShards, error) {
	if len(indices) > math.MaxUint8 {
		return RelayShards{}, errors.New("too many indices")
	}

	indiceSet := make(map[uint16]struct{})
	for _, index := range indices {
		if index > MaxShardIndex {
			return RelayShards{}, errors.New("invalid index")
		}
		indiceSet[index] = struct{}{} // dedup
	}

	if len(indiceSet) == 0 {
		return RelayShards{}, errors.New("invalid index count")
	}

	indices = []uint16{}
	for index := range indiceSet {
		indices = append(indices, index)
	}

	return RelayShards{Cluster: cluster, Indices: indices}, nil
}

func (rs RelayShards) Topics() []NamespacedPubsubTopic {
	var result []NamespacedPubsubTopic
	for _, i := range rs.Indices {
		result = append(result, NewStaticShardingPubsubTopic(rs.Cluster, i))
	}
	return result
}

func (rs RelayShards) Contains(cluster uint16, index uint16) bool {
	if rs.Cluster != cluster {
		return false
	}

	found := false
	for _, idx := range rs.Indices {
		if idx == index {
			found = true
		}
	}

	return found
}

func (rs RelayShards) ContainsNamespacedTopic(topic NamespacedPubsubTopic) bool {
	if topic.Kind() != StaticSharding {
		return false
	}

	shardedTopic := topic.(StaticShardingPubsubTopic)

	return rs.Contains(shardedTopic.Cluster(), shardedTopic.Shard())
}

func TopicsToRelayShards(topic ...string) ([]RelayShards, error) {
	result := make([]RelayShards, 0)
	dict := make(map[uint16]map[uint16]struct{})
	for _, t := range topic {
		var ps StaticShardingPubsubTopic
		err := ps.Parse(t)
		if err != nil {
			return nil, err
		}

		indices, ok := dict[ps.cluster]
		if !ok {
			indices = make(map[uint16]struct{})
		}

		indices[ps.shard] = struct{}{}
		dict[ps.cluster] = indices
	}

	for cluster, indices := range dict {
		idx := make([]uint16, 0, len(indices))
		for index := range indices {
			idx = append(idx, index)
		}

		rs, err := NewRelayShards(cluster, idx...)
		if err != nil {
			return nil, err
		}

		result = append(result, rs)
	}

	return result, nil
}

func (rs RelayShards) ContainsTopic(topic string) bool {
	nsTopic, err := ToShardedPubsubTopic(topic)
	if err != nil {
		return false
	}
	return rs.ContainsNamespacedTopic(nsTopic)
}

func (rs RelayShards) IndicesList() ([]byte, error) {
	if len(rs.Indices) > math.MaxUint8 {
		return nil, errors.New("indices list too long")
	}

	var result []byte

	result = binary.BigEndian.AppendUint16(result, rs.Cluster)
	result = append(result, uint8(len(rs.Indices)))
	for _, index := range rs.Indices {
		result = binary.BigEndian.AppendUint16(result, index)
	}

	return result, nil
}

func FromIndicesList(buf []byte) (RelayShards, error) {
	if len(buf) < 3 {
		return RelayShards{}, fmt.Errorf("insufficient data: expected at least 3 bytes, got %d bytes", len(buf))
	}

	cluster := binary.BigEndian.Uint16(buf[0:2])
	length := int(buf[2])

	if len(buf) != 3+2*length {
		return RelayShards{}, fmt.Errorf("invalid data: `length` field is %d but %d bytes were provided", length, len(buf))
	}

	var indices []uint16
	for i := 0; i < length; i++ {
		indices = append(indices, binary.BigEndian.Uint16(buf[3+2*i:5+2*i]))
	}

	return NewRelayShards(cluster, indices...)
}

func setBit(n byte, pos uint) byte {
	n |= (1 << pos)
	return n
}

func hasBit(n byte, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}

func (rs RelayShards) BitVector() []byte {
	// The value is comprised of a two-byte shard cluster index in network byte
	// order concatenated with a 128-byte wide bit vector. The bit vector
	// indicates which shards of the respective shard cluster the node is part
	// of. The right-most bit in the bit vector represents shard 0, the left-most
	// bit represents shard 1023.
	var result []byte
	result = binary.BigEndian.AppendUint16(result, rs.Cluster)

	vec := make([]byte, 128)
	for _, index := range rs.Indices {
		n := vec[index/8]
		vec[index/8] = byte(setBit(n, uint(index%8)))
	}

	return append(result, vec...)
}

func FromBitVector(buf []byte) (RelayShards, error) {
	if len(buf) != 130 {
		return RelayShards{}, errors.New("invalid data: expected 130 bytes")
	}

	cluster := binary.BigEndian.Uint16(buf[0:2])
	var indices []uint16

	for i := uint16(0); i < 128; i++ {
		for j := uint(0); j < 8; j++ {
			if !hasBit(buf[2+i], j) {
				continue
			}

			indices = append(indices, uint16(j)+8*i)
		}
	}

	return RelayShards{Cluster: cluster, Indices: indices}, nil
}
