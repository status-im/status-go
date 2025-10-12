package node

import (
	"database/sql"

	"github.com/status-im/mvds/state"
)

type EpochPersistence interface {
	Get(nodeID state.PeerID) (epoch int64, err error)
	Set(nodeID state.PeerID, epoch int64) error
}

type EpochSQLitePersistence struct {
	db *sql.DB
}

func NewEpochSQLitePersistence(db *sql.DB) *EpochSQLitePersistence {
	return &EpochSQLitePersistence{db: db}
}

func (p *EpochSQLitePersistence) Get(nodeID state.PeerID) (epoch int64, err error) {
	row := p.db.QueryRow(`SELECT epoch FROM mvds_epoch WHERE peer_id = ?`, nodeID[:])
	err = row.Scan(&epoch)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

func (p *EpochSQLitePersistence) Set(nodeID state.PeerID, epoch int64) error {
	_, err := p.db.Exec(`
		INSERT OR REPLACE INTO mvds_epoch (peer_id, epoch) VALUES (?, ?)`,
		nodeID[:],
		epoch,
	)
	return err
}
