package topic

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

const sskLen = 16

type Service struct {
	persistence PersistenceService
}

func NewService(persistence PersistenceService) *Service {
	return &Service{persistence: persistence}
}

// Receive will generate a shared secret for a given identity, and return it
func (s *Service) Receive(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) ([]byte, error) {
	sharedKey, err := ecies.ImportECDSA(myPrivateKey).GenerateShared(
		ecies.ImportECDSAPublic(theirPublicKey),
		sskLen,
		sskLen,
	)
	if err != nil {
		return nil, err
	}

	theirIdentity := crypto.CompressPubkey(theirPublicKey)
	if err = s.persistence.Add(theirIdentity, sharedKey, installationID); err != nil {
		return nil, err
	}

	return sharedKey, err
}

// Send returns a shared key if we have received a message from all the installationIDs
func (s *Service) Send(theirPublicKey *ecdsa.PublicKey, installationIDs []string) ([]byte, error) {
	theirIdentity := crypto.CompressPubkey(theirPublicKey)

	response, err := s.persistence.Get(theirIdentity, installationIDs)
	if err != nil {
		return nil, err
	}

	for _, installationID := range installationIDs {
		if !response.installationIDs[installationID] {
			return nil, nil
		}
	}

	return response.secret, nil
}

func (s *Service) All() ([][]byte, error) {
	return s.persistence.All()
}
