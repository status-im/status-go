package push_notification_client

import (
	"crypto/ecdsa"
	"database/sql"
)

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}

func (p *Persistence) TrackPushNotification(messageID []byte) error {
	return nil
}

func (p *Persistence) ShouldSentNotificationFor(publicKey *ecdsa.PublicKey, messageID []byte) (bool, error) {
	return false, nil
}
func (p *Persistence) SentFor(publicKey *ecdsa.PublicKey, messageID []byte) error {

	return nil
}
