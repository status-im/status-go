package pushnotificationserver

import (
	"crypto/ecdsa"
	"database/sql"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence interface {
	// GetPushNotificationRegistrationByPublicKeyAndInstallationID retrieve a push notification registration from storage given a public key and installation id
	GetPushNotificationRegistrationByPublicKeyAndInstallationID(publicKey []byte, installationID string) (*protobuf.PushNotificationRegistration, error)
	// GetPushNotificationRegistrationByPublicKey retrieve all the push notification registrations from storage given a public key
	GetPushNotificationRegistrationByPublicKeys(publicKeys [][]byte) ([]*PushNotificationIDAndRegistration, error)
	//GetPushNotificationRegistrationPublicKeys return all the public keys stored
	GetPushNotificationRegistrationPublicKeys() ([][]byte, error)

	// DeletePushNotificationRegistration deletes a push notification registration from storage given a public key and installation id
	DeletePushNotificationRegistration(publicKey []byte, installationID string) error
	// SavePushNotificationRegistration saves a push notification option to the db
	SavePushNotificationRegistration(publicKey []byte, registration *protobuf.PushNotificationRegistration) error
	// GetIdentity returns the server identity key
	GetIdentity() (*ecdsa.PrivateKey, error)
	// SaveIdentity saves the server identity key
	SaveIdentity(*ecdsa.PrivateKey) error
}

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) Persistence {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) GetPushNotificationRegistrationByPublicKeyAndInstallationID(publicKey []byte, installationID string) (*protobuf.PushNotificationRegistration, error) {
	var marshaledRegistration []byte
	err := p.db.QueryRow(`SELECT registration FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, publicKey, installationID).Scan(&marshaledRegistration)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	registration := &protobuf.PushNotificationRegistration{}

	if err := proto.Unmarshal(marshaledRegistration, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

type PushNotificationIDAndRegistration struct {
	ID           []byte
	Registration *protobuf.PushNotificationRegistration
}

func (p *SQLitePersistence) GetPushNotificationRegistrationByPublicKeys(publicKeys [][]byte) ([]*PushNotificationIDAndRegistration, error) {
	// TODO: check for a max number of keys

	publicKeyArgs := make([]interface{}, 0, len(publicKeys))
	for _, pk := range publicKeys {
		publicKeyArgs = append(publicKeyArgs, pk)
	}

	inVector := strings.Repeat("?, ", len(publicKeys)-1) + "?"

	rows, err := p.db.Query(`SELECT public_key,registration FROM push_notification_server_registrations WHERE public_key IN (`+inVector+`)`, publicKeyArgs...) // nolint: gosec
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registrations []*PushNotificationIDAndRegistration
	for rows.Next() {
		response := &PushNotificationIDAndRegistration{}
		var marshaledRegistration []byte
		err := rows.Scan(&response.ID, &marshaledRegistration)
		if err != nil {
			return nil, err
		}

		registration := &protobuf.PushNotificationRegistration{}

		if err := proto.Unmarshal(marshaledRegistration, registration); err != nil {
			return nil, err
		}
		response.Registration = registration
		registrations = append(registrations, response)
	}
	return registrations, nil
}

func (p *SQLitePersistence) GetPushNotificationRegistrationPublicKeys() ([][]byte, error) {
	rows, err := p.db.Query(`SELECT public_key FROM push_notification_server_registrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var publicKeys [][]byte
	for rows.Next() {
		var publicKey []byte
		err := rows.Scan(&publicKey)
		if err != nil {
			return nil, err
		}

		publicKeys = append(publicKeys, publicKey)
	}
	return publicKeys, nil
}

func (p *SQLitePersistence) SavePushNotificationRegistration(publicKey []byte, registration *protobuf.PushNotificationRegistration) error {
	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO push_notification_server_registrations (public_key, installation_id, version, registration) VALUES (?, ?, ?, ?)`, publicKey, registration.InstallationId, registration.Version, marshaledRegistration)
	return err
}

func (p *SQLitePersistence) DeletePushNotificationRegistration(publicKey []byte, installationID string) error {
	_, err := p.db.Exec(`DELETE FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, publicKey, installationID)
	return err
}

func (p *SQLitePersistence) SaveIdentity(privateKey *ecdsa.PrivateKey) error {
	_, err := p.db.Exec(`INSERT INTO push_notification_server_identity (private_key) VALUES (?)`, crypto.FromECDSA(privateKey))
	return err
}

func (p *SQLitePersistence) GetIdentity() (*ecdsa.PrivateKey, error) {
	var pkBytes []byte
	err := p.db.QueryRow(`SELECT private_key FROM push_notification_server_identity LIMIT 1`).Scan(&pkBytes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pk, err := crypto.ToECDSA(pkBytes)
	if err != nil {
		return nil, err
	}
	return pk, nil
}
