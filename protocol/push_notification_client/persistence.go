package push_notification_client

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/status-im/status-go/eth-node/crypto"
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

func (p *Persistence) UpsertServer(server *PushNotificationServer) error {
	_, err := p.db.Exec(`INSERT INTO push_notification_client_servers (public_key, registered, registered_at) VALUES (?,?,?)`, crypto.CompressPubkey(server.publicKey), server.registered, server.registeredAt)
	return err

}

func (p *Persistence) GetServers() ([]*PushNotificationServer, error) {
	rows, err := p.db.Query(`SELECT public_key, registered, registered_at FROM push_notification_client_servers`)
	if err != nil {
		return nil, err
	}
	var servers []*PushNotificationServer
	for rows.Next() {
		server := &PushNotificationServer{}
		var key []byte
		err := rows.Scan(&key, &server.registered, &server.registeredAt)
		if err != nil {
			return nil, err
		}
		parsedKey, err := crypto.DecompressPubkey(key)
		if err != nil {
			return nil, err
		}
		server.publicKey = parsedKey
		servers = append(servers, server)
	}
	return servers, nil
}
