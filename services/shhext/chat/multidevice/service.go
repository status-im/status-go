package multidevice

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/services/shhext/chat/protobuf"
)

type Installation struct {
	ID      string
	Version uint32
}

type Config struct {
	MaxInstallations int
	ProtocolVersion  uint32
	InstallationID   string
}

func New(config *Config, persistence Persistence) *Service {
	return &Service{
		config:      config,
		persistence: persistence,
	}
}

type Service struct {
	persistence Persistence
	config      *Config
}

type IdentityAndIDPair [2]string

func (s *Service) GetActiveInstallations(identity *ecdsa.PublicKey) ([]*Installation, error) {
	identityC := crypto.CompressPubkey(identity)
	return s.persistence.GetActiveInstallations(s.config.MaxInstallations, identityC)
}

func (s *Service) GetOurActiveInstallations(identity *ecdsa.PublicKey) ([]*Installation, error) {
	identityC := crypto.CompressPubkey(identity)
	installations, err := s.persistence.GetActiveInstallations(s.config.MaxInstallations-1, identityC)
	if err != nil {
		return nil, err
	}
	// Move to layer above
	installations = append(installations, &Installation{
		ID:      s.config.InstallationID,
		Version: s.config.ProtocolVersion,
	})

	return installations, nil

}

func (s *Service) EnableInstallation(identity *ecdsa.PublicKey, installationID string) error {
	identityC := crypto.CompressPubkey(identity)
	return s.persistence.EnableInstallation(identityC, installationID)
}

func (s *Service) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	myIdentityKeyC := crypto.CompressPubkey(myIdentityKey)
	return s.persistence.DisableInstallation(myIdentityKeyC, installationID)
}

// ProcessPublicBundle persists a bundle and returns a list of tuples identity/installationID
func (s *Service) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, theirIdentity *ecdsa.PublicKey, b *protobuf.Bundle) ([]IdentityAndIDPair, error) {
	signedPreKeys := b.GetSignedPreKeys()
	var response []IdentityAndIDPair
	var installations []*Installation

	myIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(&myIdentityKey.PublicKey))
	theirIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(theirIdentity))

	// Any device from other peers will be considered enabled, ours needs to
	// be explicitly enabled
	fromOurIdentity := theirIdentityStr != myIdentityStr

	for installationID, signedPreKey := range signedPreKeys {
		if installationID != s.config.InstallationID {
			installations = append(installations, &Installation{
				ID:      installationID,
				Version: signedPreKey.GetProtocolVersion(),
			})
			response = append(response, IdentityAndIDPair{theirIdentityStr, installationID})
		}
	}

	if err := s.persistence.AddInstallations(b.GetIdentity(), b.GetTimestamp(), installations, fromOurIdentity); err != nil {
		return nil, err
	}

	return response, nil
}
