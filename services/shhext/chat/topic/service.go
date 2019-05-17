package topic

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
)

const sskLen = 16

type Service struct {
	persistence PersistenceService
}

func NewService(persistence PersistenceService) *Service {
	return &Service{persistence: persistence}
}

func (s *Service) setupTopic(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) (*Secret, error) {
	log.Info("Setup topic called for", "installationID", installationID)
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

	return &Secret{Key: sharedKey, Identity: theirPublicKey}, err
}

// Receive will generate a shared secret for a given identity, and return it
func (s *Service) Receive(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) (*Secret, error) {
	return s.setupTopic(myPrivateKey, theirPublicKey, installationID)
}

// Send returns a shared key and whether it has been acknowledged from all the installationIDs
func (s *Service) Send(myPrivateKey *ecdsa.PrivateKey, myInstallationID string, theirPublicKey *ecdsa.PublicKey, theirInstallationIDs []string) (*Secret, bool, error) {
	sharedKey, err := s.setupTopic(myPrivateKey, theirPublicKey, myInstallationID)
	if err != nil {
		return nil, false, err
	}

	theirIdentity := crypto.CompressPubkey(theirPublicKey)
	response, err := s.persistence.Get(theirIdentity, theirInstallationIDs)
	if err != nil {
		return nil, false, err
	}

	for _, installationID := range theirInstallationIDs {
		if !response.installationIDs[installationID] {
			return sharedKey, false, nil
		}
	}

	return &Secret{
		Key:      response.secret,
		Identity: theirPublicKey,
	}, true, nil
}

type Secret struct {
	Identity *ecdsa.PublicKey
	Key      []byte
}

func (s *Service) All() ([]*Secret, error) {
	var secrets []*Secret
	tuples, err := s.persistence.All()
	if err != nil {
		return nil, err
	}

	for _, tuple := range tuples {
		key, err := crypto.DecompressPubkey(tuple[0])
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, &Secret{Identity: key, Key: tuple[1]})

	}

	return secrets, nil

}
