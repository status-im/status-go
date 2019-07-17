package sharedsecret

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

const sskLen = 16

type Secret struct {
	Identity *ecdsa.PublicKey
	Key      []byte
}

// SharedSecret generates and manages negotiated secrets.
// Identities (public keys) stored by SharedSecret
// are compressed.
// TODO: make it a part of sqlitePersistence instead of SharedSecret.
type SharedSecret struct {
	persistence *sqlitePersistence
}

func New(db *sql.DB) *SharedSecret {
	return &SharedSecret{
		persistence: newSQLitePersistence(db),
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

	log.Printf(
		"[SharedSecret::generate] saving a shared key for %#x and installation %s",
		crypto.FromECDSAPub(theirPublicKey),
		installationID,
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
	log.Printf(
		"[SharedSecret::Agreed] checking against for %#x and installations %#v",
		crypto.FromECDSAPub(theirPublicKey),
		theirInstallationIDs,
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
			log.Printf("[SharedSecret::Agreed] no shared secret with installation %s", installationID)
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
