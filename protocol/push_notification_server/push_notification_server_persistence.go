package push_notification_server

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence interface {
	// GetPushNotificationRegistration retrieve a push notification registration from storage given a public key and installation id
	GetPushNotificationRegistration(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationRegistration, error)
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

func (p *SQLitePersistence) GetPushNotificationRegistration(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationRegistration, error) {
	var marshaledRegistration []byte
	err := p.db.QueryRow(`SELECT registration FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, p.hashPublicKey(publicKey), installationID).Scan(&marshaledRegistration)

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

func (p *SQLitePersistence) SavePushNotificationRegistration(publicKey *ecdsa.PublicKey, registration *protobuf.PushNotificationRegistration) error {
	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO push_notification_server_registrations (public_key, installation_id, version, registration) VALUES (?, ?, ?, ?)`, p.hashPublicKey(publicKey), registration.InstallationId, registration.Version, marshaledRegistration)
	return err
}

func (p *SQLitePersistence) DeletePushNotificationRegistration(publicKey *ecdsa.PublicKey, installationID string) error {
	_, err := p.db.Exec(`DELETE FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, p.hashPublicKey(publicKey), installationID)
	return err
}

func (p *SQLitePersistence) hashPublicKey(pk *ecdsa.PublicKey) []byte {
	return shake256(crypto.CompressPubkey(pk))
}
