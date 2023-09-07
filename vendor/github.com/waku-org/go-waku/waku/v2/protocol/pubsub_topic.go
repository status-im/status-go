package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const Waku2PubsubTopicPrefix = "/waku/2"
const StaticShardingPubsubTopicPrefix = Waku2PubsubTopicPrefix + "/rs"

var ErrInvalidStructure = errors.New("invalid topic structure")
var ErrInvalidTopicPrefix = errors.New("must start with " + Waku2PubsubTopicPrefix)
var ErrMissingTopicName = errors.New("missing topic-name")
var ErrInvalidShardedTopicPrefix = errors.New("must start with " + StaticShardingPubsubTopicPrefix)
var ErrMissingClusterIndex = errors.New("missing shard_cluster_index")
var ErrMissingShardNumber = errors.New("missing shard_number")
var ErrInvalidNumberFormat = errors.New("only 2^16 numbers are allowed")

// NamespacedPubsubTopicKind used to represent kind of NamespacedPubsubTopicKind
type NamespacedPubsubTopicKind int

const (
	StaticSharding NamespacedPubsubTopicKind = iota
	NamedSharding
)

// NamespacedPubsubTopic is an interface for namespace based pubSub topic
type NamespacedPubsubTopic interface {
	String() string
	Kind() NamespacedPubsubTopicKind
	Equal(NamespacedPubsubTopic) bool
}

// NamedShardingPubsubTopic is object for a NamedSharding type pubSub topic
type NamedShardingPubsubTopic struct {
	NamespacedPubsubTopic
	kind NamespacedPubsubTopicKind
	name string
}

// NewNamedShardingPubsubTopic creates a new NamedShardingPubSubTopic
func NewNamedShardingPubsubTopic(name string) NamespacedPubsubTopic {
	return NamedShardingPubsubTopic{
		kind: NamedSharding,
		name: name,
	}
}

// Kind returns the type of PubsubTopic whether it is StaticShared or NamedSharded
func (n NamedShardingPubsubTopic) Kind() NamespacedPubsubTopicKind {
	return n.kind
}

// Name is the name of the NamedSharded pubsub topic.
func (n NamedShardingPubsubTopic) Name() string {
	return n.name
}

// Equal compares NamedShardingPubsubTopic
func (n NamedShardingPubsubTopic) Equal(t2 NamespacedPubsubTopic) bool {
	return n.String() == t2.String()
}

// String formats NamedShardingPubsubTopic to RFC 23 specific string format for pubsub topic.
func (n NamedShardingPubsubTopic) String() string {
	return fmt.Sprintf("%s/%s", Waku2PubsubTopicPrefix, n.name)
}

// Parse parses a topic string into a NamedShardingPubsubTopic
func (n *NamedShardingPubsubTopic) Parse(topic string) error {
	if !strings.HasPrefix(topic, Waku2PubsubTopicPrefix) {
		return ErrInvalidTopicPrefix
	}

	topicName := topic[8:]
	if len(topicName) == 0 {
		return ErrMissingTopicName
	}

	n.kind = NamedSharding
	n.name = topicName

	return nil
}

// StaticShardingPubsubTopic describes a pubSub topic as per StaticSharding
type StaticShardingPubsubTopic struct {
	NamespacedPubsubTopic
	kind    NamespacedPubsubTopicKind
	cluster uint16
	shard   uint16
}

// NewStaticShardingPubsubTopic creates a new pubSub topic
func NewStaticShardingPubsubTopic(cluster uint16, shard uint16) StaticShardingPubsubTopic {
	return StaticShardingPubsubTopic{
		kind:    StaticSharding,
		cluster: cluster,
		shard:   shard,
	}
}

// Cluster returns the sharded cluster index
func (s StaticShardingPubsubTopic) Cluster() uint16 {
	return s.cluster
}

// Cluster returns the shard number
func (s StaticShardingPubsubTopic) Shard() uint16 {
	return s.shard
}

// Kind returns the type of PubsubTopic whether it is StaticShared or NamedSharded
func (s StaticShardingPubsubTopic) Kind() NamespacedPubsubTopicKind {
	return s.kind
}

// Equal compares StaticShardingPubsubTopic
func (s StaticShardingPubsubTopic) Equal(t2 NamespacedPubsubTopic) bool {
	return s.String() == t2.String()
}

// String formats StaticShardingPubsubTopic to RFC 23 specific string format for pubsub topic.
func (s StaticShardingPubsubTopic) String() string {
	return fmt.Sprintf("%s/%d/%d", StaticShardingPubsubTopicPrefix, s.cluster, s.shard)
}

// Parse parses a topic string into a StaticShardingPubsubTopic
func (s *StaticShardingPubsubTopic) Parse(topic string) error {
	if !strings.HasPrefix(topic, StaticShardingPubsubTopicPrefix) {
		return ErrInvalidShardedTopicPrefix
	}

	parts := strings.Split(topic[11:], "/")
	if len(parts) != 2 {
		return ErrInvalidStructure
	}

	clusterPart := parts[0]
	if len(clusterPart) == 0 {
		return ErrMissingClusterIndex
	}

	clusterInt, err := strconv.ParseUint(clusterPart, 10, 16)
	if err != nil {
		return ErrInvalidNumberFormat
	}

	shardPart := parts[1]
	if len(shardPart) == 0 {
		return ErrMissingShardNumber
	}

	shardInt, err := strconv.ParseUint(shardPart, 10, 16)
	if err != nil {
		return ErrInvalidNumberFormat
	}

	s.shard = uint16(shardInt)
	s.cluster = uint16(clusterInt)
	s.kind = StaticSharding

	return nil
}

// ToShardedPubsubTopic takes a pubSub topic string and creates a NamespacedPubsubTopic object.
func ToShardedPubsubTopic(topic string) (NamespacedPubsubTopic, error) {
	if strings.HasPrefix(topic, StaticShardingPubsubTopicPrefix) {
		s := StaticShardingPubsubTopic{}
		err := s.Parse(topic)
		if err != nil {
			return nil, err
		}
		return s, nil
	} else {
		s := NamedShardingPubsubTopic{}
		err := s.Parse(topic)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
}

// DefaultPubsubTopic is the default pubSub topic used in waku
func DefaultPubsubTopic() NamespacedPubsubTopic {
	return NewNamedShardingPubsubTopic("default-waku/proto")
}
