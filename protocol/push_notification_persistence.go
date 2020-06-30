package protocol

import (
	"crypto/ecdsa"
	"database/sql"
)

type PushNotificationPersistence struct {
	db *sql.DB
}

func NewPushNotificationPersistence(db *sql.DB) *PushNotificationPersistence {
	return &PushNotificationPersistence{db: db}
}

func (p *PushNotificationPersistence) TrackPushNotification(messageID []byte) error {
	return nil
}

func (p *PushNotificationPersistence) ShouldSentNotificationFor(publicKey *ecdsa.PublicKey, messageID []byte) (bool, error) {
	return false, nil
}
func (p *PushNotificationPersistence) PushNotificationSentFor(publicKey *ecdsa.PublicKey, messageID []byte) error {

	return nil
}
