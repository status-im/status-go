package sharedsecret

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"go.uber.org/zap"
)

const sskLen = 16

type Secret struct {
	Identity *ecdsa.PublicKey
	Key      []byte
}

// SharedSecret generates and manages negotiated secrets.
// Identities (public keys) stored by SharedSecret
// are compressed.
// TODO: make compression of public keys a responsibility  of sqlitePersistence instead of SharedSecret.
type SharedSecret struct {
	persistence *sqlitePersistence
	logger      *zap.Logger
}

func New(db *sql.DB, logger *zap.Logger) *SharedSecret {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SharedSecret{
		persistence: newSQLitePersistence(db),
		logger:      logger.With(zap.Namespace("SharedSecret")),
	}
}

func (s *SharedSecret) generate(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) (*Secret, error) {
	sharedKey, err := ecies.ImportECDSA(myPrivateKey).GenerateShared(
		ecies.ImportECDSAPublic(theirPublicKey),
		sskLen,
		sskLen,
	)
	if err != nil {
		return nil, err
	}

	logger := s.logger.With(zap.String("site", "generate"))

	logger.Debug(
		"saving a shared key",
		zap.Binary("their-public-key", crypto.FromECDSAPub(theirPublicKey)),
		zap.String("installation-id", installationID),
	)

	theirIdentity := crypto.CompressPubkey(theirPublicKey)
	if err = s.persistence.Add(theirIdentity, sharedKey, installationID); err != nil {
		return nil, err
	}

	return &Secret{Key: sharedKey, Identity: theirPublicKey}, err
}

// Generate will generate a shared secret for a given identity, and return it.
func (s *SharedSecret) Generate(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, installationID string) (*Secret, error) {
	return s.generate(myPrivateKey, theirPublicKey, installationID)
}

// Agreed returns true if a secret has been acknowledged by all the installationIDs.
func (s *SharedSecret) Agreed(myPrivateKey *ecdsa.PrivateKey, myInstallationID string, theirPublicKey *ecdsa.PublicKey, theirInstallationIDs []string) (*Secret, bool, error) {
	logger := s.logger.With(zap.String("site", "Agreed"))

	logger.Debug(
		"checking if shared secret is acknowledged",
		zap.Binary("their-public-key", crypto.FromECDSAPub(theirPublicKey)),
		zap.Strings("their-installation-ids", theirInstallationIDs),
	)

	secret, err := s.generate(myPrivateKey, theirPublicKey, myInstallationID)
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
			logger.Debug("no shared secret for installation", zap.String("installation-id", installationID))
			return secret, false, nil
		}
	}

	if !bytes.Equal(secret.Key, response.secret) {
		return nil, false, errors.New("computed and saved secrets are different for a given identity")
	}

	return secret, true, nil
}

func (s *SharedSecret) All() ([]*Secret, error) {
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
