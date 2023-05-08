package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const Waku2PubsubTopicPrefix = "/waku/2"
const StaticShardingPubsubTopicPrefix = Waku2PubsubTopicPrefix + "/rs"

var ErrInvalidFormat = errors.New("invalid format")
var ErrInvalidStructure = errors.New("invalid topic structure")
var ErrInvalidTopicPrefix = errors.New("must start with " + Waku2PubsubTopicPrefix)
var ErrMissingTopicName = errors.New("missing topic-name")
var ErrInvalidShardedTopicPrefix = errors.New("must start with " + StaticShardingPubsubTopicPrefix)
var ErrMissingClusterIndex = errors.New("missing shard_cluster_index")
var ErrMissingShardNumber = errors.New("missing shard_number")
var ErrInvalidNumberFormat = errors.New("only 2^16 numbers are allowed")

type ContentTopic struct {
	ApplicationName    string
	ApplicationVersion uint
	ContentTopicName   string
	Encoding           string
}

func (ct ContentTopic) String() string {
	return fmt.Sprintf("/%s/%d/%s/%s", ct.ApplicationName, ct.ApplicationVersion, ct.ContentTopicName, ct.Encoding)
}

func NewContentTopic(applicationName string, applicationVersion uint, contentTopicName string, encoding string) ContentTopic {
	return ContentTopic{
		ApplicationName:    applicationName,
		ApplicationVersion: applicationVersion,
		ContentTopicName:   contentTopicName,
		Encoding:           encoding,
	}
}

func (ct ContentTopic) Equal(ct2 ContentTopic) bool {
	return ct.ApplicationName == ct2.ApplicationName && ct.ApplicationVersion == ct2.ApplicationVersion &&
		ct.ContentTopicName == ct2.ContentTopicName && ct.Encoding == ct2.Encoding
}

func StringToContentTopic(s string) (ContentTopic, error) {
	p := strings.Split(s, "/")

	if len(p) != 5 || p[0] != "" || p[1] == "" || p[2] == "" || p[3] == "" || p[4] == "" {
		return ContentTopic{}, ErrInvalidFormat
	}

	vNum, err := strconv.ParseUint(p[2], 10, 32)
	if err != nil {
		return ContentTopic{}, ErrInvalidFormat
	}

	return ContentTopic{
		ApplicationName:    p[1],
		ApplicationVersion: uint(vNum),
		ContentTopicName:   p[3],
		Encoding:           p[4],
	}, nil
}

type NamespacedPubsubTopicKind int

const (
	StaticSharding NamespacedPubsubTopicKind = iota
	NamedSharding
)

type NamespacedPubsubTopic interface {
	String() string
	Kind() NamespacedPubsubTopicKind
	Equal(NamespacedPubsubTopic) bool
}

type NamedShardingPubsubTopic struct {
	NamespacedPubsubTopic
	kind NamespacedPubsubTopicKind
	name string
}

func NewNamedShardingPubsubTopic(name string) NamespacedPubsubTopic {
	return NamedShardingPubsubTopic{
		kind: NamedSharding,
		name: name,
	}
}

func (n NamedShardingPubsubTopic) Kind() NamespacedPubsubTopicKind {
	return n.kind
}

func (n NamedShardingPubsubTopic) Name() string {
	return n.name
}

func (s NamedShardingPubsubTopic) Equal(t2 NamespacedPubsubTopic) bool {
	return s.String() == t2.String()
}

func (n NamedShardingPubsubTopic) String() string {
	return fmt.Sprintf("%s/%s", Waku2PubsubTopicPrefix, n.name)
}

func (s *NamedShardingPubsubTopic) Parse(topic string) error {
	if !strings.HasPrefix(topic, Waku2PubsubTopicPrefix) {
		return ErrInvalidTopicPrefix
	}

	topicName := topic[8:]
	if len(topicName) == 0 {
		return ErrMissingTopicName
	}

	s.kind = NamedSharding
	s.name = topicName

	return nil
}

type StaticShardingPubsubTopic struct {
	NamespacedPubsubTopic
	kind    NamespacedPubsubTopicKind
	cluster uint16
	shard   uint16
}

func NewStaticShardingPubsubTopic(cluster uint16, shard uint16) NamespacedPubsubTopic {
	return StaticShardingPubsubTopic{
		kind:    StaticSharding,
		cluster: cluster,
		shard:   shard,
	}
}

func (n StaticShardingPubsubTopic) Cluster() uint16 {
	return n.cluster
}

func (n StaticShardingPubsubTopic) Shard() uint16 {
	return n.shard
}

func (n StaticShardingPubsubTopic) Kind() NamespacedPubsubTopicKind {
	return n.kind
}

func (s StaticShardingPubsubTopic) Equal(t2 NamespacedPubsubTopic) bool {
	return s.String() == t2.String()
}

func (n StaticShardingPubsubTopic) String() string {
	return fmt.Sprintf("%s/%d/%d", StaticShardingPubsubTopicPrefix, n.cluster, n.shard)
}

func (s *StaticShardingPubsubTopic) Parse(topic string) error {
	if !strings.HasPrefix(topic, StaticShardingPubsubTopicPrefix) {
		fmt.Println(topic, StaticShardingPubsubTopicPrefix)
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

func DefaultPubsubTopic() NamespacedPubsubTopic {
	return NewNamedShardingPubsubTopic("default-waku/proto")
}
