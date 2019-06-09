package mailserver

import (
	"database/sql"
	"fmt"
	"time"

	// Import postgres driver
	_ "github.com/lib/pq"
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/postgres"
	"github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/status-im/status-go/mailserver/migrations"
	"github.com/status-im/status-go/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/status-im/whisper/whisperv6"
)

func NewPostgresDB(config *params.WhisperConfig) (*PostgresDB, error) {
	db, err := sql.Open("postgres", config.DatabaseConfig.PGConfig.URI)
	if err != nil {
		return nil, err
	}

	instance := &PostgresDB{db: db}
	if err := instance.setup(); err != nil {
		return nil, err
	}

	return instance, nil
}

type PostgresDB struct {
	db *sql.DB
}

type postgresIterator struct {
	*sql.Rows
}

func (i *postgresIterator) DBKey() (*DBKey, error) {
	var value []byte
	var id []byte
	if err := i.Scan(&id, &value); err != nil {
		return nil, err
	}
	return &DBKey{raw: id}, nil
}

func (i *postgresIterator) Error() error {
	return nil
}

func (i *postgresIterator) Release() {
	i.Close()
}

func (i *postgresIterator) GetEnvelope(bloom []byte) ([]byte, error) {
	var value []byte
	var id []byte
	if err := i.Scan(&id, &value); err != nil {
		return nil, err
	}

	return value, nil
}

func (i *PostgresDB) BuildIterator(query CursorQuery) (Iterator, error) {
	var upperLimit []byte
	var stmtString string
	if len(query.cursor) > 0 {
		// If we have a cursor, we don't want to include that envelope in the result set
		upperLimit = query.cursor

		// We disable security checks as we need to use string interpolation
		// for this, but it's converted to 0s and 1s so no injection should be possible
		/* #nosec */
		stmtString = fmt.Sprintf("SELECT id, data FROM envelopes where id >= $1 AND id < $2 AND bloom & b'%s'::bit(512) = bloom ORDER BY ID DESC LIMIT $3", toBitString(query.bloom))
	} else {
		upperLimit = query.end
		// We disable security checks as we need to use string interpolation
		// for this, but it's converted to 0s and 1s so no injection should be possible
		/* #nosec */
		stmtString = fmt.Sprintf("SELECT id, data FROM envelopes where id >= $1 AND id <= $2 AND bloom & b'%s'::bit(512) = bloom ORDER BY ID DESC LIMIT $3", toBitString(query.bloom))
	}

	stmt, err := i.db.Prepare(stmtString)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(query.start, upperLimit, query.limit)

	if err != nil {
		return nil, err
	}

	return &postgresIterator{rows}, nil
}

func (i *PostgresDB) setup() error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(i.db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"postgres",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (i *PostgresDB) Close() error {
	return i.db.Close()
}

func (i *PostgresDB) GetEnvelope(key *DBKey) ([]byte, error) {
	statement := `SELECT data FROM envelopes WHERE id = $1`

	stmt, err := i.db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var envelope []byte

	if err = stmt.QueryRow(key.Bytes()).Scan(&envelope); err != nil {
		return nil, err
	}

	return envelope, nil
}

func (i *PostgresDB) Prune(t time.Time, batch int) (int, error) {
	var zero common.Hash
	var emptyTopic whisper.TopicType
	kl := NewDBKey(0, emptyTopic, zero)
	ku := NewDBKey(uint32(t.Unix()), emptyTopic, zero)
	statement := "DELETE FROM envelopes WHERE id BETWEEN $1 AND $2"

	stmt, err := i.db.Prepare(statement)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	if _, err = stmt.Exec(kl.Bytes(), ku.Bytes()); err != nil {
		return 0, err
	}

	return 0, nil
}

func (i *PostgresDB) SaveEnvelope(env *whisper.Envelope) error {
	key := NewDBKey(env.Expiry-env.TTL, env.Topic, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
		archivedErrorsCounter.Inc(1)
		return err
	}

	statement := "INSERT INTO envelopes (id, data, topic, bloom) VALUES ($1, $2, $3, B'"
	statement += toBitString(env.Bloom())
	statement += "'::bit(512)) ON CONFLICT (id) DO NOTHING;"
	stmt, err := i.db.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		key.Bytes(),
		rawEnvelope,
		topicToByte(env.Topic),
	)

	if err != nil {
		archivedErrorsCounter.Inc(1)
		return err
	}

	archivedMeter.Mark(1)
	archivedSizeMeter.Mark(int64(whisper.EnvelopeHeaderLength + len(env.Data)))

	return nil
}

func topicToByte(t whisper.TopicType) []byte {
	return []byte{t[0], t[1], t[2], t[3]}
}

func toBitString(bloom []byte) string {
	val := ""
	for _, n := range bloom {
		val += fmt.Sprintf("%08b", n)
	}
	return val
}
