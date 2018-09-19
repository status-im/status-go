// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisperv6

import "time"

// RateLimitConfig defines configuration for actual rate limiter.
type RateLimitConfig struct {
	Interval uint64
	Capacity uint64
	Quantum  uint64
}

func (conf RateLimitConfig) IntervalDuration() time.Duration {
	return time.Duration(conf.Interval)
}

// Config represents the configuration state of a whisper node.
type Config struct {
	MaxMessageSize     uint32  `toml:",omitempty"`
	MinimumAcceptedPOW float64 `toml:",omitempty"`
	TimeSource         func() time.Time
	IngressRateLimit   RateLimitConfig
	EgressRateLimit    RateLimitConfig
	TopicRateLimit     RateLimitConfig
	IgnoreEgressLimit  bool // used to make a peer generate more traffic that the other peer can handle
}

// DefaultConfig represents (shocker!) the default configuration.
var DefaultConfig = Config{
	MaxMessageSize:     DefaultMaxMessageSize,
	MinimumAcceptedPOW: DefaultMinimumPoW,
	TimeSource:         time.Now,
	IngressRateLimit:   RateLimitConfig{uint64(1 * time.Minute), 1 << (10 * 3), 10 << (10 * 2)},
	EgressRateLimit:    RateLimitConfig{uint64(500 * time.Millisecond), 1 << (10 * 3), 10 << (10 * 2)},
	TopicRateLimit:     RateLimitConfig{uint64(500 * time.Millisecond), 50 << (10 * 2), 5 << (10 * 2)},
}
