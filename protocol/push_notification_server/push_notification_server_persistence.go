package protocol

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence interface {
	GetPushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationOptions, error)
}

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) Persistence {
	return &SQLitePersistence{db: db}
}

func (p *SQLitePersistence) GetPushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationOptions, error) {
	return nil, nil
}
