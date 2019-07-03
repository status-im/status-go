package sharedsecret

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
)

const sskLen = 16

type Service struct {
	log         log.Logger
	persistence Persistence
}

func NewService(persistence Persistence) *Service {
	return &Service{
		log:         log.New("package", "status-go/messaging/sharedsecret.Service"),
		persistence: persistence,
	}
}

func (s *Service) setup(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) (*Secret, error) {
	s.log.Debug("Setup called for", "installationID", installationID)
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
	s.log.Debug("Received message, setting up topic", "public-key", theirPublicKey, "installation-id", installationID)
	return s.setup(myPrivateKey, theirPublicKey, installationID)
}

// Send returns a shared key and whether it has been acknowledged from all the installationIDs
func (s *Service) Send(myPrivateKey *ecdsa.PrivateKey, myInstallationID string, theirPublicKey *ecdsa.PublicKey, theirInstallationIDs []string) (*Secret, bool, error) {
	s.log.Debug("Checking against:", "installation-ids", theirInstallationIDs)
	secret, err := s.setup(myPrivateKey, theirPublicKey, myInstallationID)
	if err != nil {
		return nil, false, err
	}

	if len(theirInstallationIDs) == 0 {
		return secret, false, nil
	}

	theirIdentity := crypto.CompressPubkey(theirPublicKey)
	response, err := s.persistence.Get(theirIdentity, theirInstallationIDs)
	if err != nil {
		return nil, false, err
	}

	for _, installationID := range theirInstallationIDs {
		if !response.installationIDs[installationID] {
			s.log.Debug("no shared secret with:", "installation-id", installationID)
			return secret, false, nil
		}
	}

	s.log.Debug("shared secret found")

	return secret, true, nil
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
