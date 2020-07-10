package push_notification_client

import (
	"crypto/ecdsa"
	"database/sql"
	"strings"

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
	_, err := p.db.Exec(`INSERT INTO push_notification_client_servers (public_key, registered, registered_at, access_token) VALUES (?,?,?,?)`, crypto.CompressPubkey(server.PublicKey), server.Registered, server.RegisteredAt, server.AccessToken)
	return err

}

func (p *Persistence) GetServers() ([]*PushNotificationServer, error) {
	rows, err := p.db.Query(`SELECT public_key, registered, registered_at,access_token FROM push_notification_client_servers`)
	if err != nil {
		return nil, err
	}
	var servers []*PushNotificationServer
	for rows.Next() {
		server := &PushNotificationServer{}
		var key []byte
		err := rows.Scan(&key, &server.Registered, &server.RegisteredAt, &server.AccessToken)
		if err != nil {
			return nil, err
		}
		parsedKey, err := crypto.DecompressPubkey(key)
		if err != nil {
			return nil, err
		}
		server.PublicKey = parsedKey
		servers = append(servers, server)
	}
	return servers, nil
}

func (p *Persistence) GetServersByPublicKey(keys []*ecdsa.PublicKey) ([]*PushNotificationServer, error) {

	keyArgs := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		keyArgs = append(keyArgs, crypto.CompressPubkey(key))
	}

	inVector := strings.Repeat("?, ", len(keys)-1) + "?"
	rows, err := p.db.Query(`SELECT public_key, registered, registered_at,access_token FROM push_notification_client_servers WHERE public_key IN (`+inVector+")", keyArgs...) //nolint: gosec
	if err != nil {
		return nil, err
	}
	var servers []*PushNotificationServer
	for rows.Next() {
		server := &PushNotificationServer{}
		var key []byte
		err := rows.Scan(&key, &server.Registered, &server.RegisteredAt, &server.AccessToken)
		if err != nil {
			return nil, err
		}
		parsedKey, err := crypto.DecompressPubkey(key)
		if err != nil {
			return nil, err
		}
		server.PublicKey = parsedKey
		servers = append(servers, server)
	}
	return servers, nil
}
