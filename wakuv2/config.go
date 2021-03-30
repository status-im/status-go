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
	MaxMessageSize         uint32   `toml:",omitempty"`
	SoftBlacklistedPeerIDs []string `toml:",omitempty"`
	Host                   string   `toml:",omitempty"`
	Port                   int      `toml:",omitempty"`
	BootNodes              []string `toml:",omitempty"`
}

var DefaultConfig = Config{
	MaxMessageSize: common.DefaultMaxMessageSize,
	Host:           "0.0.0.0",
	Port:           60000,
}
