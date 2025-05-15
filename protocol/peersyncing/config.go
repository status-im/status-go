package peersyncing

import (
	"database/sql"

	"github.com/status-im/status-go/v10/protocol/common"
)

type Config struct {
	SyncMessagePersistence SyncMessagePersistence
	Database               *sql.DB
	Timesource             common.TimeSource
}
