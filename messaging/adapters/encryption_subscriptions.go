package adapters

import (
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/messaging/utils"
)

func FromEncryptionSubscriptions(s *encryption.Subscriptions) *types.EncryptionSubscriptions {
	if s == nil {
		return nil
	}
	return &types.EncryptionSubscriptions{
		SharedSecrets:      FromEncryptionSharedSecrets(s.SharedSecrets),
		SendContactCode:    s.SendContactCode,
		NewHashRatchetKeys: utils.BridgeChannelsSlice(s.NewHashRatchetKeys, FromEncryptionHashRatchet),
		Quit:               s.Quit,
	}
}
