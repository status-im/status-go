package adapters

import (
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/utils"
)

func FromEncryptionSubscriptions(s *encryption.Subscriptions) *types.EncryptionSubscriptions {
	if s == nil {
		return nil
	}
	return &types.EncryptionSubscriptions{
		SendContactCode:    s.SendContactCode,
		NewHashRatchetKeys: utils.BridgeChannelsSlice(s.NewHashRatchetKeys, FromEncryptionHashRatchet),
		Quit:               s.Quit,
	}
}
