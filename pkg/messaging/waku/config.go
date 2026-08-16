// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package wakuv2

import (
	"go.uber.org/zap"

	ethdisc "github.com/ethereum/go-ethereum/p2p/dnsdisc"

	"github.com/status-im/status-go/pkg/messaging/waku/common"
	"github.com/status-im/status-go/pkg/messaging/waku/fleets"
)

// Config represents the configuration state of a waku node.
type Config struct {
	MaxMessageSize           uint32 `toml:",omitempty"` // Maximal message length allowed by the waku node
	Host                     string `toml:",omitempty"`
	Port                     int    `toml:",omitempty"`
	EnablePeerExchangeServer bool   `toml:",omitempty"` // PeerExchange server makes sense only when discv5 is running locally as it will have a cache of peers that it can respond to in case a PeerExchange request comes from the PeerExchangeClient
	EnablePeerExchangeClient bool   `toml:",omitempty"`
	MinPeersForRelay         int    `toml:",omitempty"` // Indicates the minimum number of peers required for using Relay Protocol
	MaxPeersForFilter        int    `toml:",omitempty"` // Indicates the minimum number of peers required for using Filter Protocol
	// Fleet, when set, is the source of truth for the peer configuration: the
	// waku node resolves WakuNodes / DiscV5BootstrapNodes / ClusterID from the
	// fleet registry (see setDefaults). Leave empty to configure those fields
	// directly (used by tests pointing at ephemeral nodes).
	Fleet string `toml:",omitempty"`
	// Mode selects Core (full/relay, the default) vs Edge (light). It derives the
	// peer-exchange / discv5 flags below and is the single source of truth for
	// the light-vs-full distinction (see IsLightClient).
	Mode                       Mode             `toml:",omitempty"`
	WakuNodes                  []string         `toml:",omitempty"`
	DiscV5BootstrapNodes       []string         `toml:",omitempty"`
	Nameserver                 string           `toml:",omitempty"` // Optional nameserver to use for dns discovery
	Resolver                   ethdisc.Resolver `toml:",omitempty"` // Optional resolver to use for dns discovery
	EnableDiscV5               bool             `toml:",omitempty"` // Indicates whether discv5 is enabled or not
	DiscoveryLimit             int              `toml:",omitempty"` // Indicates the number of nodes to discover with peer exchange client
	AutoUpdate                 bool             `toml:",omitempty"`
	UDPPort                    int              `toml:",omitempty"`
	MetricsEnabled             bool             `toml:",omitempty"`
	DefaultShardPubsubTopic    string           `toml:",omitempty"` // Pubsub topic to be used by default for messages that do not have a topic assigned (depending whether sharding is used or not)
	DefaultShardedPubsubTopics []string         `toml:", omitempty"`
	ClusterID                  uint16           `toml:",omitempty"`
	EnableConfirmations        bool             `toml:",omitempty"` // Enable sending message confirmations
	SkipPublishToTopic         bool             `toml:",omitempty"` // Used in testing
	UseThrottledPublish        bool             `toml:",omitempty"` // Flag that indicates whether a rate limited priority queue will be used to send messages or not
}

// IsLightClient reports whether the node is a light (Edge) node. Mode is the
// single source of truth; there is no separate LightClient flag.
func (c *Config) IsLightClient() bool {
	return c.Mode.IsLightClient()
}

func (c *Config) Validate(logger *zap.Logger) error {
	// The peer-exchange / discv5 flags are derived from Mode in setDefaults, so
	// the only thing to reject here is an unknown mode value.
	return c.Mode.Validate()
}

var DefaultConfig = Config{
	MaxMessageSize:    common.DefaultMaxMessageSize,
	Host:              "0.0.0.0",
	Port:              0,
	DiscoveryLimit:    20,
	MinPeersForRelay:  1, // TODO: determine correct value with Vac team
	MaxPeersForFilter: 3, // TODO: determine correct value with Vac team and via testing
	AutoUpdate:        false,
}

func setDefaults(cfg *Config) *Config {
	if cfg == nil {
		cfg = new(Config)
	}

	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = DefaultConfig.MaxMessageSize
	}

	if cfg.Host == "" {
		cfg.Host = DefaultConfig.Host
	}

	if cfg.DiscoveryLimit == 0 {
		cfg.DiscoveryLimit = DefaultConfig.DiscoveryLimit
	}

	if cfg.MinPeersForRelay == 0 {
		cfg.MinPeersForRelay = DefaultConfig.MinPeersForRelay
	}

	if cfg.MaxPeersForFilter == 0 {
		cfg.MaxPeersForFilter = DefaultConfig.MaxPeersForFilter
	}

	if cfg.DefaultShardPubsubTopic == "" {
		cfg.DefaultShardPubsubTopic = DefaultShardPubsubTopic()
		//For now populating with both used shards, but this can be populated from user subscribed communities etc once community sharding is implemented
		cfg.DefaultShardedPubsubTopics = append(cfg.DefaultShardedPubsubTopics, DefaultShardPubsubTopic())
		cfg.DefaultShardedPubsubTopics = append(cfg.DefaultShardedPubsubTopics, DefaultNonProtectedPubsubTopic())
	}

	// Derive the peer-exchange / discv5 flags from the mode (Core is the default).
	cfg.Mode.applyTo(cfg)

	// Resolve the peer configuration from the fleet registry. When a fleet is
	// given it is the source of truth, so the waku node owns fleet resolution
	// and callers pass only a fleet name. An empty fleet leaves WakuNodes /
	// DiscV5BootstrapNodes / ClusterID as set directly on the config.
	if cfg.Fleet != "" {
		cfg.WakuNodes = fleets.WakuNodes(cfg.Fleet)
		cfg.DiscV5BootstrapNodes = fleets.DiscV5Nodes(cfg.Fleet)
		cfg.ClusterID = fleets.ClusterID(cfg.Fleet)
	}

	return cfg
}
