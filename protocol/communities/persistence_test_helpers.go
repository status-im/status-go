package communities

import (
	"database/sql"

	"github.com/status-im/status-go/protocol/protobuf"
)

type RawCommunityRow struct {
	ID          []byte
	PrivateKey  []byte
	Description []byte
	Joined      bool
	Spectated   bool
	Verified    bool
	SyncedAt    uint64
	Muted       bool
}

func fromSyncCommunityProtobuf(syncCommProto *protobuf.SyncCommunity) RawCommunityRow {
	return RawCommunityRow{
		ID:          syncCommProto.Id,
		Description: syncCommProto.Description,
		Joined:      syncCommProto.Joined,
		Spectated:   syncCommProto.Spectated,
		Verified:    syncCommProto.Verified,
		SyncedAt:    syncCommProto.Clock,
		Muted:       syncCommProto.Muted,
	}
}

func (p *Persistence) scanRowToStruct(rowScan func(dest ...interface{}) error) (*RawCommunityRow, error) {
	rcr := new(RawCommunityRow)
	var syncedAt, muteTill sql.NullTime

	err := rowScan(
		&rcr.ID,
		&rcr.PrivateKey,
		&rcr.Description,
		&rcr.Joined,
		&rcr.Verified,
		&rcr.Spectated,
		&rcr.Muted,
		&muteTill,
		&syncedAt,
	)
	if syncedAt.Valid {
		rcr.SyncedAt = uint64(syncedAt.Time.Unix())
	}

	if err != nil {
		return nil, err
	}

	return rcr, nil
}

func (p *Persistence) getAllCommunitiesRaw() (rcrs []*RawCommunityRow, err error) {
	var rows *sql.Rows
	// Keep "*", if the db table is updated, syncing needs to match, this fail will force us to update syncing.
	rows, err = p.db.Query(`SELECT * FROM communities_communities`)
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
		rcr, err := p.scanRowToStruct(rows.Scan)
		if err != nil {
			return nil, err
		}

		rcrs = append(rcrs, rcr)
	}
	return rcrs, nil
}

func (p *Persistence) getRawCommunityRow(id []byte) (*RawCommunityRow, error) {
	// Keep "*", if the db table is updated, syncing needs to match, this fail will force us to update syncing.
	qr := p.db.QueryRow(`SELECT * FROM communities_communities WHERE id = ?`, id)
	return p.scanRowToStruct(qr.Scan)
}

func (p *Persistence) getSyncedRawCommunity(id []byte) (*RawCommunityRow, error) {
	// Keep "*", if the db table is updated, syncing needs to match, this fail will force us to update syncing.
	qr := p.db.QueryRow(`SELECT * FROM communities_communities WHERE id = ? AND synced_at > 0`, id)
	return p.scanRowToStruct(qr.Scan)
}

func (p *Persistence) saveRawCommunityRow(rawCommRow RawCommunityRow) error {
	_, err := p.db.Exec(
		`INSERT INTO communities_communities ("id", "private_key", "description", "joined", "verified", "synced_at", "muted") VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rawCommRow.ID,
		rawCommRow.PrivateKey,
		rawCommRow.Description,
		rawCommRow.Joined,
		rawCommRow.Verified,
		rawCommRow.SyncedAt,
		rawCommRow.Muted,
	)
	return err
}

func (p *Persistence) saveRawCommunityRowWithoutSyncedAt(rawCommRow RawCommunityRow) error {
	_, err := p.db.Exec(
		`INSERT INTO communities_communities ("id", "private_key", "description", "joined", "verified", "muted") VALUES (?, ?, ?, ?, ?, ?)`,
		rawCommRow.ID,
		rawCommRow.PrivateKey,
		rawCommRow.Description,
		rawCommRow.Joined,
		rawCommRow.Verified,
		rawCommRow.Muted,
	)
	return err
}
