package server

import (
	"encoding/base64"
	"strings"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
)

func IsWakuProtocol(protocol protocol.ID) bool {
	return protocol == filter.FilterPushID_v20beta1 ||
		protocol == filter.FilterSubscribeID_v20beta1 ||
		protocol == relay.WakuRelayID_v200 ||
		protocol == lightpush.LightPushID_v20beta1 ||
		protocol == legacy_store.StoreID_v20beta4 ||
		protocol == store.StoreQueryID_v300
}

type Base64URLByte []byte

// UnmarshalText is used by json.Unmarshal to decode both url-safe and standard
// base64 encoded strings with and without padding
func (h *Base64URLByte) UnmarshalText(b []byte) error {
	inputValue := ""
	if b != nil {
		inputValue = string(b)
	}

	enc := base64.StdEncoding
	if strings.ContainsAny(inputValue, "-_") {
		enc = base64.URLEncoding
	}
	if len(inputValue)%4 != 0 {
		enc = enc.WithPadding(base64.NoPadding)
	}

	decodedBytes, err := enc.DecodeString(inputValue)
	if err != nil {
		return err
	}

	*h = decodedBytes

	return nil
}
