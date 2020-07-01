package push_notification_server

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence interface {
	// GetPushNotificationOptions retrieve a push notification options from storage given a public key and installation id
	GetPushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationOptions, error)
	// DeletePushNotificationOptions deletes a push notification options from storage given a public key and installation id
	DeletePushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) error
	// SavePushNotificationOptions saves a push notification option to the db
	SavePushNotificationOptions(publicKey *ecdsa.PublicKey, options *protobuf.PushNotificationOptions) error
}

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) Persistence {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) GetPushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationOptions, error) {
	var marshaledOptions []byte
	err := p.db.QueryRow(`SELECT registration FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, crypto.CompressPubkey(publicKey), installationID).Scan(&marshaledOptions)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	options := &protobuf.PushNotificationOptions{}

	if err := proto.Unmarshal(marshaledOptions, options); err != nil {
		return nil, err
	}
	return options, nil
}

func (p *SQLitePersistence) SavePushNotificationOptions(publicKey *ecdsa.PublicKey, options *protobuf.PushNotificationOptions) error {
	compressedPublicKey := crypto.CompressPubkey(publicKey)
	marshaledOptions, err := proto.Marshal(options)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO push_notification_server_registrations (public_key, installation_id, version, registration) VALUES (?, ?, ?, ?)`, compressedPublicKey, options.InstallationId, options.Version, marshaledOptions)
	return err
}

func (p *SQLitePersistence) DeletePushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) error {
	_, err := p.db.Exec(`DELETE FROM push_notification_server_registrations WHERE public_key = ? AND installation_id = ?`, crypto.CompressPubkey(publicKey), installationID)
	return err
}
