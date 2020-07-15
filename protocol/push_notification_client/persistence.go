package push_notification_client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/gob"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}

func (p *Persistence) GetLastPushNotificationRegistration() (*protobuf.PushNotificationRegistration, []*ecdsa.PublicKey, error) {
	var registrationBytes []byte
	var contactIDsBytes []byte
	err := p.db.QueryRow(`SELECT registration,contact_ids FROM push_notification_client_registrations LIMIT 1`).Scan(&registrationBytes, &contactIDsBytes)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	var publicKeyBytes [][]byte
	var contactIDs []*ecdsa.PublicKey
	// Restore contactIDs
	contactIDsDecoder := gob.NewDecoder(bytes.NewBuffer(contactIDsBytes))
	err = contactIDsDecoder.Decode(&publicKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	for _, pkBytes := range publicKeyBytes {
		pk, err := crypto.UnmarshalPubkey(pkBytes)
		if err != nil {
			return nil, nil, err
		}
		contactIDs = append(contactIDs, pk)
	}

	registration := &protobuf.PushNotificationRegistration{}

	err = proto.Unmarshal(registrationBytes, registration)
	if err != nil {
		return nil, nil, err
	}

	return registration, contactIDs, nil
}

func (p *Persistence) SaveLastPushNotificationRegistration(registration *protobuf.PushNotificationRegistration, contactIDs []*ecdsa.PublicKey) error {
	var encodedContactIDs bytes.Buffer
	var contactIDsBytes [][]byte
	for _, pk := range contactIDs {
		contactIDsBytes = append(contactIDsBytes, crypto.FromECDSAPub(pk))
	}
	pkEncoder := gob.NewEncoder(&encodedContactIDs)
	if err := pkEncoder.Encode(contactIDsBytes); err != nil {
		return err
	}

	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`INSERT INTO push_notification_client_registrations (registration,contact_ids) VALUES (?, ?)`, marshaledRegistration, encodedContactIDs.Bytes())
	return err
}

func (p *Persistence) TrackPushNotification(chatID string, messageID []byte) error {
	trackedAt := time.Now().Unix()
	_, err := p.db.Exec(`INSERT INTO push_notification_client_tracked_messages (chat_id, message_id, tracked_at) VALUES (?,?,?)`, chatID, messageID, trackedAt)
	return err
}

func (p *Persistence) TrackedMessage(messageID []byte) (bool, error) {
	var count uint64
	err := p.db.QueryRow(`SELECT COUNT(1) FROM push_notification_client_tracked_messages WHERE message_id = ?`, messageID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func (p *Persistence) SavePushNotificationQuery(publicKey *ecdsa.PublicKey, queryID []byte) error {
	queriedAt := time.Now().Unix()
	_, err := p.db.Exec(`INSERT INTO push_notification_client_queries (public_key, query_id, queried_at) VALUES (?,?,?)`, crypto.CompressPubkey(publicKey), queryID, queriedAt)
	return err
}

func (p *Persistence) GetQueriedAt(publicKey *ecdsa.PublicKey) (int64, error) {
	var queriedAt int64
	err := p.db.QueryRow(`SELECT queried_at FROM push_notification_client_queries WHERE public_key = ? ORDER BY queried_at DESC LIMIT 1`, crypto.CompressPubkey(publicKey)).Scan(&queriedAt)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return queriedAt, nil
}

func (p *Persistence) GetQueryPublicKey(queryID []byte) (*ecdsa.PublicKey, error) {
	var publicKeyBytes []byte
	err := p.db.QueryRow(`SELECT public_key FROM push_notification_client_queries WHERE query_id = ?`, queryID).Scan(&publicKeyBytes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.DecompressPubkey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

func (p *Persistence) SavePushNotificationInfo(infos []*PushNotificationInfo) error {
	tx, err := p.db.BeginTx(context.Background(), &sql.TxOptions{})
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()
	for _, info := range infos {
		_, err := tx.Exec(`INSERT INTO push_notification_client_info (public_key, server_public_key, installation_id, access_token, retrieved_at) VALUES (?, ?, ?, ?, ?)`, crypto.CompressPubkey(info.PublicKey), crypto.CompressPubkey(info.ServerPublicKey), info.InstallationID, info.AccessToken, info.RetrievedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Persistence) GetPushNotificationInfo(publicKey *ecdsa.PublicKey, installationIDs []string) ([]*PushNotificationInfo, error) {
	queryArgs := make([]interface{}, 0, len(installationIDs)+1)
	queryArgs = append(queryArgs, crypto.CompressPubkey(publicKey))
	for _, installationID := range installationIDs {
		queryArgs = append(queryArgs, installationID)
	}

	inVector := strings.Repeat("?, ", len(installationIDs)-1) + "?"

	rows, err := p.db.Query(`SELECT server_public_key, installation_id, access_token, retrieved_at FROM push_notification_client_info WHERE public_key = ? AND installation_id IN (`+inVector+`)`, queryArgs...)
	if err != nil {
		return nil, err
	}
	var infos []*PushNotificationInfo
	for rows.Next() {
		var serverPublicKeyBytes []byte
		info := &PushNotificationInfo{PublicKey: publicKey}
		err := rows.Scan(&serverPublicKeyBytes, &info.InstallationID, &info.AccessToken, &info.RetrievedAt)
		if err != nil {
			return nil, err
		}

		serverPublicKey, err := crypto.DecompressPubkey(serverPublicKeyBytes)
		if err != nil {
			return nil, err
		}

		info.ServerPublicKey = serverPublicKey
		infos = append(infos, info)
	}

	return infos, nil
}

func (p *Persistence) GetPushNotificationInfoByPublicKey(publicKey *ecdsa.PublicKey) ([]*PushNotificationInfo, error) {
	rows, err := p.db.Query(`SELECT server_public_key, installation_id, access_token, retrieved_at FROM push_notification_client_info WHERE public_key = ?`, crypto.CompressPubkey(publicKey))
	if err != nil {
		return nil, err
	}
	var infos []*PushNotificationInfo
	for rows.Next() {
		var serverPublicKeyBytes []byte
		info := &PushNotificationInfo{PublicKey: publicKey}
		err := rows.Scan(&serverPublicKeyBytes, &info.InstallationID, &info.AccessToken, &info.RetrievedAt)
		if err != nil {
			return nil, err
		}

		serverPublicKey, err := crypto.DecompressPubkey(serverPublicKeyBytes)
		if err != nil {
			return nil, err
		}

		info.ServerPublicKey = serverPublicKey
		infos = append(infos, info)
	}

	return infos, nil
}

func (p *Persistence) ShouldSendNotificationFor(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) (bool, error) {
	// First we check that we are tracking this message, next we check that we haven't already sent this
	var count uint64
	err := p.db.QueryRow(`SELECT COUNT(1) FROM push_notification_client_tracked_messages WHERE message_id = ?`, messageID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	err = p.db.QueryRow(`SELECT COUNT(1) FROM push_notification_client_sent_notifications WHERE message_id = ? AND public_key = ? AND installation_id = ? `, messageID, crypto.CompressPubkey(publicKey), installationID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

func (p *Persistence) ShouldSendNotificationToAllInstallationIDs(publicKey *ecdsa.PublicKey, messageID []byte) (bool, error) {
	// First we check that we are tracking this message, next we check that we haven't already sent this
	var count uint64
	err := p.db.QueryRow(`SELECT COUNT(1) FROM push_notification_client_tracked_messages WHERE message_id = ?`, messageID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	err = p.db.QueryRow(`SELECT COUNT(1) FROM push_notification_client_sent_notifications WHERE message_id = ? AND public_key = ? `, messageID, crypto.CompressPubkey(publicKey)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

func (p *Persistence) NotifiedOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) error {
	sentAt := time.Now().Unix()
	_, err := p.db.Exec(`INSERT INTO push_notification_client_sent_notifications (public_key, installation_id, message_id, sent_at) VALUES (?, ?, ?, ?)`, crypto.CompressPubkey(publicKey), installationID, messageID, sentAt)
	return err
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
