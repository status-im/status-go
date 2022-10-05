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
	"github.com/status-im/status-go/wakuv2/common"
)

// Config represents the configuration state of a waku node.
type Config struct {
	MaxMessageSize       uint32   `toml:",omitempty"`
	Host                 string   `toml:",omitempty"`
	Port                 int      `toml:",omitempty"`
	PersistPeers         bool     `toml:",omitempty"`
	PeerExchange         bool     `toml:",omitempty"`
	KeepAliveInterval    int      `toml:",omitempty"`
	MinPeersForRelay     int      `toml:",omitempty"`
	LightClient          bool     `toml:",omitempty"`
	RelayNodes           []string `toml:",omitempty"`
	StoreNodes           []string `toml:",omitempty"`
	FilterNodes          []string `toml:",omitempty"`
	LightpushNodes       []string `toml:",omitempty"`
	Rendezvous           bool     `toml:",omitempty"`
	WakuRendezvousNodes  []string `toml:",omitempty"`
	DiscV5BootstrapNodes []string `toml:",omitempty"`
	EnableDiscV5         bool     `toml:",omitempty"`
	DiscoveryLimit       int      `toml:",omitempty"`
	AutoUpdate           bool     `toml:",omitempty"`
	UDPPort              int      `toml:",omitempty"`
	EnableStore          bool     `toml:",omitempty"`
	StoreCapacity        int      `toml:",omitempty"`
	StoreSeconds         int      `toml:",omitempty"`
}

var DefaultConfig = Config{
	MaxMessageSize:    common.DefaultMaxMessageSize,
	Host:              "0.0.0.0",
	Port:              60000,
	KeepAliveInterval: 10, // second
	DiscoveryLimit:    40,
	MinPeersForRelay:  2, // TODO: determine correct value with Vac team
	UDPPort:           9000,
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

	if cfg.Port == 0 {
		cfg.Port = DefaultConfig.Port
	}

	if cfg.KeepAliveInterval == 0 {
		cfg.KeepAliveInterval = DefaultConfig.KeepAliveInterval
	}

	if cfg.DiscoveryLimit == 0 {
		cfg.DiscoveryLimit = DefaultConfig.DiscoveryLimit
	}

	if cfg.MinPeersForRelay == 0 {
		cfg.MinPeersForRelay = DefaultConfig.MinPeersForRelay
	}

	if cfg.UDPPort == 0 {
		cfg.UDPPort = DefaultConfig.UDPPort
	}

	return cfg
}
