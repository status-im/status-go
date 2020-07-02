package push_notification_server

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
	GetPushNotificationRegistrationByPublicKeyAndInstallationID(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationRegistration, error)
	// GetPushNotificationRegistrationByPublicKey retrieve all the push notification registrations from storage given a public key
	GetPushNotificationRegistrationByPublicKeys(publicKeys [][]byte) ([]*PushNotificationIDAndRegistration, error)

	// DeletePushNotificationRegistration deletes a push notification registration from storage given a public key and installation id
	DeletePushNotificationRegistration(publicKey *ecdsa.PublicKey, installationID string) error
	// SavePushNotificationRegistration saves a push notification option to the db
	SavePushNotificationRegistration(publicKey *ecdsa.PublicKey, registration *protobuf.PushNotificationRegistration) error
}

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) Persistence {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) GetPushNotificationRegistrationByPublicKeyAndInstallationID(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationRegistration, error) {
	var marshaledRegistration []byte
	err := p.db.QueryRow(`SELECT registration FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, hashPublicKey(publicKey), installationID).Scan(&marshaledRegistration)

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

	rows, err := p.db.Query(`SELECT public_key,registration FROM push_notification_server_registrations WHERE public_key IN (`+inVector+`)`, publicKeyArgs...)
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

func (p *SQLitePersistence) SavePushNotificationRegistration(publicKey *ecdsa.PublicKey, registration *protobuf.PushNotificationRegistration) error {
	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO push_notification_server_registrations (public_key, installation_id, version, registration) VALUES (?, ?, ?, ?)`, hashPublicKey(publicKey), registration.InstallationId, registration.Version, marshaledRegistration)
	return err
}

func (p *SQLitePersistence) DeletePushNotificationRegistration(publicKey *ecdsa.PublicKey, installationID string) error {
	_, err := p.db.Exec(`DELETE FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, hashPublicKey(publicKey), installationID)
	return err
}

func hashPublicKey(pk *ecdsa.PublicKey) []byte {
	return shake256(crypto.CompressPubkey(pk))
}
