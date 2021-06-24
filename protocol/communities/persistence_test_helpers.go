package communities

import (
	"database/sql"

	"github.com/status-im/status-go/protocol/protobuf"
)

type rawCommunityRow struct {
	ID []byte
	PrivateKey []byte
	Description []byte
	Joined bool
	Verified bool
	SyncedAt uint64
}

func fromSyncCommunityProtobuf(syncCommProto *protobuf.SyncCommunity) rawCommunityRow {
	return rawCommunityRow{
		ID:          syncCommProto.Id,
		PrivateKey:  syncCommProto.PrivateKey,
		Description: syncCommProto.Description,
		Joined:      syncCommProto.Joined,
		Verified:    syncCommProto.Verified,
		SyncedAt:    syncCommProto.Clock,
	}
}

func (rc *rawCommunityRow) toSyncCommunityProtobuf() *protobuf.SyncCommunity {
	return &protobuf.SyncCommunity{
		Clock:                rc.SyncedAt,
		Id:                   rc.ID,
		PrivateKey:           rc.PrivateKey,
		Description:          rc.Description,
		Joined:               rc.Joined,
		Verified:             rc.Verified,
	}
}

func (p *Persistence) scanRowToStruct(rowScan func(dest ...interface{}) error) (*rawCommunityRow, error) {
	rcr := new(rawCommunityRow)
	syncedAt := sql.NullTime{}

	err := rowScan(
		&rcr.ID,
		&rcr.PrivateKey,
		&rcr.Description,
		&rcr.Joined,
		&rcr.Verified,
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

func (p *Persistence) getAllCommunitiesRaw() (rcrs []*rawCommunityRow, err error) {
	var rows *sql.Rows
	rows, err = p.db.Query(`SELECT id, private_key, description, joined, verified, synced_at FROM communities_communities`)
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

func (p *Persistence) getRawCommunityRow(id []byte) (*rawCommunityRow, error) {
	qr := p.db.QueryRow(`SELECT * FROM communities_communities WHERE id = ?`, id)
	return p.scanRowToStruct(qr.Scan)
}

func (p *Persistence) saveRawCommunityRow(rawCommRow rawCommunityRow) error {
	_, err := p.db.Exec(
		`INSERT INTO communities_communities ("id", "private_key", "description", "joined", "verified", "synced_at") VALUES (?, ?, ?, ?, ?, ?)`,
		rawCommRow.ID,
		rawCommRow.PrivateKey,
		rawCommRow.Description,
		rawCommRow.Joined,
		rawCommRow.Verified,
		rawCommRow.SyncedAt,
	)
	return err
}
