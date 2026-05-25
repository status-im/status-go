package segmentation

import (
	"context"
	"crypto/ecdsa"
	"database/sql"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/pkg/messaging/layers/segmentation/protobuf"
)

type SQLitePersistence struct {
	db *sql.DB
}

func NewSQLitePersistence(db *sql.DB) *SQLitePersistence {
	return &SQLitePersistence{db: db}
}

func (s *SQLitePersistence) IsMessageAlreadyCompleted(hash []byte) (bool, error) {
	var alreadyCompleted int
	err := s.db.QueryRow("SELECT COUNT(*) FROM message_segments_completed WHERE hash = ?", hash).Scan(&alreadyCompleted)
	if err != nil {
		return false, err
	}
	return alreadyCompleted > 0, nil
}

func (s *SQLitePersistence) SaveMessageSegment(segment *Message, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	_, err := s.db.Exec("INSERT INTO message_segments (hash, segment_index, segments_count, parity_segment_index, parity_segments_count, sig_pub_key, payload, timestamp, transport_id, original_payload_length) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		segment.EntireMessageHash, segment.Index, segment.SegmentsCount, segment.ParitySegmentIndex, segment.ParitySegmentsCount, sigPubKeyBlob, segment.Payload, timestamp, segment.transportID, segment.OriginalPayloadLength)

	return err
}

// Get ordered message segments for given hash
func (s *SQLitePersistence) GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*Message, error) {
	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	rows, err := s.db.Query(`
		SELECT
			hash, segment_index, segments_count, parity_segment_index, parity_segments_count, payload, transport_id, original_payload_length
		FROM
			message_segments
		WHERE
			hash = ? AND sig_pub_key = ?
		ORDER BY
			(segments_count = 0) ASC, -- Prioritize segments_count > 0
			segment_index ASC,
			parity_segment_index ASC`,
		hash, sigPubKeyBlob)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []*Message
	for rows.Next() {
		segment := &Message{
			SegmentMessage: &protobuf.SegmentMessage{},
		}
		err := rows.Scan(&segment.EntireMessageHash, &segment.Index, &segment.SegmentsCount, &segment.ParitySegmentIndex, &segment.ParitySegmentsCount, &segment.Payload, &segment.transportID, &segment.OriginalPayloadLength)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return segments, nil
}

func (s *SQLitePersistence) RemoveMessageSegmentsOlderThan(timestamp int64) error {
	_, err := s.db.Exec("DELETE FROM message_segments WHERE timestamp < ?", timestamp)
	return err
}

func (s *SQLitePersistence) CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	sigPubKeyBlob := crypto.CompressPubkey(sigPubKey)

	_, err = tx.Exec("DELETE FROM message_segments WHERE hash = ? AND sig_pub_key = ?", hash, sigPubKeyBlob)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO message_segments_completed (hash, sig_pub_key, timestamp) VALUES (?,?,?)", hash, sigPubKeyBlob, timestamp)
	if err != nil {
		return err
	}

	return err
}

func (s *SQLitePersistence) RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error {
	_, err := s.db.Exec("DELETE FROM message_segments_completed WHERE timestamp < ?", timestamp)
	return err
}
