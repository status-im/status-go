package chain

import (
	"database/sql"
	"time"
)

type RPCLimiterDB struct {
	db *sql.DB
}

func NewRPCLimiterDB(db *sql.DB) *RPCLimiterDB {
	return &RPCLimiterDB{
		db: db,
	}
}

func (r *RPCLimiterDB) CreateRPCLimit(limit LimitData) error {
	query := `INSERT INTO rpc_limits (tag, created_at, period, max_requests, counter) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, limit.Tag, limit.CreatedAt.Unix(), limit.Period, limit.MaxReqs, limit.NumReqs)
	if err != nil {
		return err
	}
	return nil
}

func (r *RPCLimiterDB) GetRPCLimit(tag string) (*LimitData, error) {
	query := `SELECT tag, created_at, period, max_requests, counter FROM rpc_limits WHERE tag = ?`
	row := r.db.QueryRow(query, tag)
	limit := &LimitData{}
	createdAtSecs := int64(0)
	err := row.Scan(&limit.Tag, &createdAtSecs, &limit.Period, &limit.MaxReqs, &limit.NumReqs)
	if err != nil {
		return nil, err
	}

	limit.CreatedAt = time.Unix(createdAtSecs, 0)
	return limit, nil
}

func (r *RPCLimiterDB) UpdateRPCLimit(limit LimitData) error {
	query := `UPDATE rpc_limits SET created_at = ?, period = ?, max_requests = ?, counter = ? WHERE tag = ?`
	_, err := r.db.Exec(query, limit.CreatedAt.Unix(), limit.Period, limit.MaxReqs, limit.NumReqs, limit.Tag)
	if err != nil {
		return err
	}
	return nil
}

func (r *RPCLimiterDB) DeleteRPCLimit(tag string) error {
	query := `DELETE FROM rpc_limits WHERE tag = ?`
	_, err := r.db.Exec(query, tag)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}
