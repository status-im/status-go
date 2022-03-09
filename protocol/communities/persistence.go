package communities

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence struct {
	db     *sql.DB
	logger *zap.Logger
}

var ErrOldRequestToJoin = errors.New("old request to join")

const communitiesBaseQuery = `SELECT c.id, c.private_key, c.description,c.joined,c.verified,c.muted,r.clock FROM communities_communities c LEFT JOIN communities_requests_to_join r ON c.id = r.community_id AND r.public_key = ?`

func (p *Persistence) SaveCommunity(community *Community) error {
	id := community.ID()
	privateKey := community.PrivateKey()
	description, err := community.ToBytes()
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO communities_communities (id, private_key, description, joined, verified) VALUES (?, ?, ?, ?, ?)`, id, crypto.FromECDSA(privateKey), description, community.config.Joined, community.config.Verified)
	return err
}

func (p *Persistence) ShouldHandleSyncCommunity(community *protobuf.SyncCommunity) (bool, error) {
	// TODO see if there is a way to make this more elegant
	// Keep the "*".
	// When the test for this function fails because the table has changed we should update sync functionality
	qr := p.db.QueryRow(`SELECT * FROM communities_communities WHERE id = ? AND synced_at > ?`, community.Id, community.Clock)
	_, err := p.scanRowToStruct(qr.Scan)

	switch err {
	case sql.ErrNoRows:
		// Query does not match, therefore synced_at value is not older than the new clock value or id was not found
		return true, nil
	case nil:
		// Error is nil, therefore query matched and synced_at is older than the new clock
		return false, nil
	default:
		// Error is not nil and is not sql.ErrNoRows, therefore pass out the error
		return false, err
	}
}

func (p *Persistence) queryCommunities(memberIdentity *ecdsa.PublicKey, query string) (response []*Community, err error) {

	rows, err := p.db.Query(query, common.PubkeyToHex(memberIdentity))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			// Don't shadow original error
			_ = rows.Close()
			return

		}
		err = rows.Close()
	}()

	for rows.Next() {
		var publicKeyBytes, privateKeyBytes, descriptionBytes []byte
		var joined, verified, muted bool
		var requestedToJoinAt sql.NullInt64
		err := rows.Scan(&publicKeyBytes, &privateKeyBytes, &descriptionBytes, &joined, &verified, &muted, &requestedToJoinAt)
		if err != nil {
			return nil, err
		}

		org, err := unmarshalCommunityFromDB(memberIdentity, publicKeyBytes, privateKeyBytes, descriptionBytes, joined, verified, muted, uint64(requestedToJoinAt.Int64), p.logger)
		if err != nil {
			return nil, err
		}
		response = append(response, org)
	}

	return response, nil

}

func (p *Persistence) AllCommunities(memberIdentity *ecdsa.PublicKey) ([]*Community, error) {
	return p.queryCommunities(memberIdentity, communitiesBaseQuery)
}

func (p *Persistence) JoinedCommunities(memberIdentity *ecdsa.PublicKey) ([]*Community, error) {
	query := communitiesBaseQuery + ` WHERE c.joined`
	return p.queryCommunities(memberIdentity, query)
}

func (p *Persistence) JoinedAndPendingCommunitiesWithRequests(memberIdentity *ecdsa.PublicKey) (comms []*Community, err error) {
	query := `SELECT
c.id, c.private_key, c.description, c.joined, c.verified, c.muted,
r.id, r.public_key, r.clock, r.ens_name, r.chat_id, r.community_id, r.state
FROM communities_communities c
LEFT JOIN communities_requests_to_join r ON c.id = r.community_id AND r.public_key = ?
WHERE c.Joined OR r.state = ?`

	rows, err := p.db.Query(query, common.PubkeyToHex(memberIdentity), RequestToJoinStatePending)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			// Don't shadow original error
			_ = rows.Close()
			return

		}
		err = rows.Close()
	}()

	for rows.Next() {
		var comm *Community

		// Community specific fields
		var publicKeyBytes, privateKeyBytes, descriptionBytes []byte
		var joined, verified, muted bool

		// Request to join specific fields
		var rtjID, rtjCommunityID []byte
		var rtjPublicKey, rtjENSName, rtjChatID sql.NullString
		var rtjClock, rtjState sql.NullInt64

		err = rows.Scan(
			&publicKeyBytes, &privateKeyBytes, &descriptionBytes, &joined, &verified, &muted,
			&rtjID, &rtjPublicKey, &rtjClock, &rtjENSName, &rtjChatID, &rtjCommunityID, &rtjState)
		if err != nil {
			return nil, err
		}

		comm, err = unmarshalCommunityFromDB(memberIdentity, publicKeyBytes, privateKeyBytes, descriptionBytes, joined, verified, muted, uint64(rtjClock.Int64), p.logger)
		if err != nil {
			return nil, err
		}

		rtj := unmarshalRequestToJoinFromDB(rtjID, rtjCommunityID, rtjPublicKey, rtjENSName, rtjChatID, rtjClock, rtjState)
		if !rtj.Empty() {
			comm.AddRequestToJoin(rtj)
		}
		comms = append(comms, comm)
	}

	return comms, nil
}

func (p *Persistence) CreatedCommunities(memberIdentity *ecdsa.PublicKey) ([]*Community, error) {
	query := communitiesBaseQuery + ` WHERE c.private_key IS NOT NULL`
	return p.queryCommunities(memberIdentity, query)
}

func (p *Persistence) GetByID(memberIdentity *ecdsa.PublicKey, id []byte) (*Community, error) {
	var publicKeyBytes, privateKeyBytes, descriptionBytes []byte
	var joined bool
	var verified bool
	var muted bool
	var requestedToJoinAt sql.NullInt64

	err := p.db.QueryRow(communitiesBaseQuery+` WHERE c.id = ?`, common.PubkeyToHex(memberIdentity), id).Scan(&publicKeyBytes, &privateKeyBytes, &descriptionBytes, &joined, &verified, &muted, &requestedToJoinAt)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return unmarshalCommunityFromDB(memberIdentity, publicKeyBytes, privateKeyBytes, descriptionBytes, joined, verified, muted, uint64(requestedToJoinAt.Int64), p.logger)
}

func unmarshalCommunityFromDB(memberIdentity *ecdsa.PublicKey, publicKeyBytes, privateKeyBytes, descriptionBytes []byte, joined, verified, muted bool, requestedToJoinAt uint64, logger *zap.Logger) (*Community, error) {

	var privateKey *ecdsa.PrivateKey
	var err error

	if privateKeyBytes != nil {
		privateKey, err = crypto.ToECDSA(privateKeyBytes)
		if err != nil {
			return nil, err
		}
	}
	metadata := &protobuf.ApplicationMetadataMessage{}

	err = proto.Unmarshal(descriptionBytes, metadata)
	if err != nil {
		return nil, err
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(metadata.Payload, description)
	if err != nil {
		return nil, err
	}

	id, err := crypto.DecompressPubkey(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	config := Config{
		PrivateKey:                    privateKey,
		CommunityDescription:          description,
		MemberIdentity:                memberIdentity,
		MarshaledCommunityDescription: descriptionBytes,
		Logger:                        logger,
		ID:                            id,
		Verified:                      verified,
		Muted:                         muted,
		RequestedToJoinAt:             requestedToJoinAt,
		Joined:                        joined,
	}
	return New(config)
}

func unmarshalRequestToJoinFromDB(ID, communityID []byte, publicKey, ensName, chatID sql.NullString, clock, state sql.NullInt64) *RequestToJoin {
	return &RequestToJoin{
		ID:          ID,
		PublicKey:   publicKey.String,
		Clock:       uint64(clock.Int64),
		ENSName:     ensName.String,
		ChatID:      chatID.String,
		CommunityID: communityID,
		State:       RequestToJoinState(state.Int64),
	}
}

func (p *Persistence) SaveRequestToJoin(request *RequestToJoin) (err error) {
	tx, err := p.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	var clock uint64
	// Fetch any existing request to join
	err = tx.QueryRow(`SELECT clock FROM communities_requests_to_join WHERE state = ? AND public_key = ? AND community_id = ?`, RequestToJoinStatePending, request.PublicKey, request.CommunityID).Scan(&clock)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// This is already processed
	if clock >= request.Clock {
		return ErrOldRequestToJoin
	}

	_, err = tx.Exec(`INSERT INTO communities_requests_to_join(id,public_key,clock,ens_name,chat_id,community_id,state) VALUES (?, ?, ?, ?, ?, ?, ?)`, request.ID, request.PublicKey, request.Clock, request.ENSName, request.ChatID, request.CommunityID, request.State)
	return err
}

func (p *Persistence) PendingRequestsToJoinForUser(pk string) ([]*RequestToJoin, error) {
	var requests []*RequestToJoin
	rows, err := p.db.Query(`SELECT id,public_key,clock,ens_name,chat_id,community_id,state FROM communities_requests_to_join WHERE state = ? AND public_key = ?`, RequestToJoinStatePending, pk)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		request := &RequestToJoin{}
		err := rows.Scan(&request.ID, &request.PublicKey, &request.Clock, &request.ENSName, &request.ChatID, &request.CommunityID, &request.State)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (p *Persistence) HasPendingRequestsToJoinForUserAndCommunity(userPk string, communityID []byte) (bool, error) {
	var count int
	err := p.db.QueryRow(`SELECT count(1) FROM communities_requests_to_join WHERE state = ? AND public_key = ? AND community_id = ?`, RequestToJoinStatePending, userPk, communityID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *Persistence) PendingRequestsToJoinForCommunity(id []byte) ([]*RequestToJoin, error) {
	var requests []*RequestToJoin
	rows, err := p.db.Query(`SELECT id,public_key,clock,ens_name,chat_id,community_id,state FROM communities_requests_to_join WHERE state = ? AND community_id = ?`, RequestToJoinStatePending, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		request := &RequestToJoin{}
		err := rows.Scan(&request.ID, &request.PublicKey, &request.Clock, &request.ENSName, &request.ChatID, &request.CommunityID, &request.State)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (p *Persistence) SetRequestToJoinState(pk string, communityID []byte, state RequestToJoinState) error {
	_, err := p.db.Exec(`UPDATE communities_requests_to_join SET state = ? WHERE community_id = ? AND public_key = ?`, state, communityID, pk)
	return err
}

func (p *Persistence) SetMuted(communityID []byte, muted bool) error {
	_, err := p.db.Exec(`UPDATE communities_communities SET muted = ? WHERE id = ?`, muted, communityID)
	return err
}

func (p *Persistence) GetRequestToJoin(id []byte) (*RequestToJoin, error) {
	request := &RequestToJoin{}
	err := p.db.QueryRow(`SELECT id,public_key,clock,ens_name,chat_id,community_id,state FROM communities_requests_to_join WHERE id = ?`, id).Scan(&request.ID, &request.PublicKey, &request.Clock, &request.ENSName, &request.ChatID, &request.CommunityID, &request.State)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (p *Persistence) SetSyncClock(id []byte, clock uint64) error {
	_, err := p.db.Exec(`UPDATE communities_communities SET synced_at = ? WHERE id = ? AND synced_at < ?`, clock, id, clock)
	return err
}

func (p *Persistence) SetPrivateKey(id []byte, privKey *ecdsa.PrivateKey) error {
	_, err := p.db.Exec(`UPDATE communities_communities SET private_key = ? WHERE id = ?`, crypto.FromECDSA(privKey), id)
	return err
}

func (p *Persistence) SaveWakuMessage(message *types.Message) error {
	_, err := p.db.Exec(`INSERT OR REPLACE INTO waku_messages (sig, timestamp, topic, payload, padding, hash) VALUES (?, ?, ?, ?, ?, ?)`,
		message.Sig,
		message.Timestamp,
		message.Topic.String(),
		message.Payload,
		message.Padding,
		types.Bytes2Hex(message.Hash),
	)
	return err
}

func (p *Persistence) GetCommunitiesSettings() ([]CommunitySettings, error) {
	rows, err := p.db.Query("SELECT community_id, message_archive_seeding_enabled, message_archive_fetching_enabled FROM communities_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	communitiesSettings := []CommunitySettings{}

	for rows.Next() {
		settings := CommunitySettings{}
		err := rows.Scan(&settings.CommunityID, &settings.HistoryArchiveSupportEnabled, &settings.HistoryArchiveSupportEnabled)
		if err != nil {
			return nil, err
		}
		communitiesSettings = append(communitiesSettings, settings)
	}
	return communitiesSettings, err
}

func (p *Persistence) CommunitySettingsExist(communityID types.HexBytes) (bool, error) {
	var count int
	err := p.db.QueryRow(`SELECT count(1) FROM communities_settings WHERE community_id = ?`, communityID.String()).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *Persistence) GetCommunitySettingsByID(communityID types.HexBytes) (*CommunitySettings, error) {
	settings := CommunitySettings{}
	err := p.db.QueryRow(`SELECT community_id, message_archive_seeding_enabled, message_archive_fetching_enabled FROM communities_settings WHERE community_id = ?`, communityID.String()).Scan(&settings.CommunityID, &settings.HistoryArchiveSupportEnabled, &settings.HistoryArchiveSupportEnabled)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (p *Persistence) DeleteCommunitySettings(communityID types.HexBytes) error {
	_, err := p.db.Exec("DELETE FROM communities_settings WHERE community_id = ?", communityID.String())
	return err
}

func (p *Persistence) SaveCommunitySettings(communitySettings CommunitySettings) error {
	_, err := p.db.Exec(`INSERT INTO communities_settings (
    community_id,
    message_archive_seeding_enabled,
    message_archive_fetching_enabled
  ) VALUES (?, ?, ?)`,
		communitySettings.CommunityID,
		communitySettings.HistoryArchiveSupportEnabled,
		communitySettings.HistoryArchiveSupportEnabled,
	)
	return err
}

func (p *Persistence) UpdateCommunitySettings(communitySettings CommunitySettings) error {
	_, err := p.db.Exec(`UPDATE communities_settings SET
    message_archive_seeding_enabled = ?,
    message_archive_fetching_enabled = ?
    WHERE community_id = ?`,
		communitySettings.HistoryArchiveSupportEnabled,
		communitySettings.HistoryArchiveSupportEnabled,
		communitySettings.CommunityID,
	)
	return err
}
