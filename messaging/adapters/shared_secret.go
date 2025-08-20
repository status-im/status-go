package adapters

import (
	"github.com/status-im/status-go/messaging/layers/encryption/sharedsecret"
	"github.com/status-im/status-go/messaging/types"
)

func FromEncryptionSharedSecret(s *sharedsecret.Secret) *types.SharedSecret {
	if s == nil {
		return nil
	}
	return &types.SharedSecret{
		Identity: s.Identity,
		Key:      s.Key,
	}
}

func FromEncryptionSharedSecrets(s []*sharedsecret.Secret) []*types.SharedSecret {
	if s == nil {
		return nil
	}
	result := make([]*types.SharedSecret, 0, len(s))
	for _, secret := range s {
		result = append(result, FromEncryptionSharedSecret(secret))
	}
	return result
}
