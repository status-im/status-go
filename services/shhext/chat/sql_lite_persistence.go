package chat

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	sqlite "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
	dr "github.com/status-im/doubleratchet"
	"github.com/status-im/migrate"
	"github.com/status-im/migrate/database/sqlcipher"
	"github.com/status-im/migrate/source/go_bindata"
	ecrypto "github.com/status-im/status-go/services/shhext/chat/crypto"
	"github.com/status-im/status-go/services/shhext/chat/migrations"
)

// A safe max number of rows
const maxNumberOfRows = 100000000

// The default number of kdf iterations in sqlcipher (from version 3.0.0)
// https://github.com/sqlcipher/sqlcipher/blob/fda4c68bb474da7e955be07a2b807bda1bb19bd2/CHANGELOG.md#300---2013-11-05
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#kdf_iter
const defaultKdfIterationsNumber = 64000

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const kdfIterationsNumber = 3200

const exportDB = "SELECT sqlcipher_export('newdb')"

// SQLLitePersistence represents a persistence service tied to an SQLite database
type SQLLitePersistence struct {
	db             *sql.DB
	keysStorage    dr.KeysStorage
	sessionStorage dr.SessionStorage
}

// SQLLiteKeysStorage represents a keys persistence service tied to an SQLite database
type SQLLiteKeysStorage struct {
	db *sql.DB
}

// SQLLiteSessionStorage represents a session persistence service tied to an SQLite database
type SQLLiteSessionStorage struct {
	db *sql.DB
}

// NewSQLLitePersistence creates a new SQLLitePersistence instance, given a path and a key
func NewSQLLitePersistence(path string, key string) (*SQLLitePersistence, error) {
	s := &SQLLitePersistence{}

	if err := s.Open(path, key); err != nil {
		return nil, err
	}

	s.keysStorage = NewSQLLiteKeysStorage(s.db)

	s.sessionStorage = NewSQLLiteSessionStorage(s.db)

	return s, nil
}

func MigrateDBFile(oldPath string, newPath string, oldKey string, newKey string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	db, err := openDB(newPath, oldKey, defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	keyString := fmt.Sprintf("PRAGMA rekey = '%s'", newKey)

	if _, err = db.Exec(keyString); err != nil {
		return err
	}

	return nil

}

// MigrateDBKeyKdfIterations changes the number of kdf iterations executed
// during the database key derivation. This change is necessary because
// of performance reasons.
// https://github.com/status-im/status-go/pull/1343
// `sqlcipher_export` is used for migration, check out this link for details:
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#sqlcipher_export
func MigrateDBKeyKdfIterations(oldPath string, newPath string, key string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	isEncrypted, err := sqlite.IsEncrypted(oldPath)
	if err != nil {
		return err
	}

	// Nothing to do, move db to the next migration
	if !isEncrypted {
		return os.Rename(oldPath, newPath)
	}

	db, err := openDB(oldPath, key, defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	attach := fmt.Sprintf(
		"ATTACH DATABASE '%s' AS newdb KEY '%s'",
		newPath,
		key)

	if _, err = db.Exec(attach); err != nil {
		return err
	}

	changeKdfIter := fmt.Sprintf(
		"PRAGMA newdb.kdf_iter = %d",
		kdfIterationsNumber)

	if _, err = db.Exec(changeKdfIter); err != nil {
		return err
	}

	if _, err = db.Exec(exportDB); err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}

	return os.Remove(oldPath)
}

// EncryptDatabase encrypts an unencrypted database with key
func EncryptDatabase(oldPath string, newPath string, key string) error {
	_, err := os.Stat(oldPath)

	// No files, nothing to do
	if os.IsNotExist(err) {
		return nil
	}

	// Any other error, throws
	if err != nil {
		return err
	}

	isEncrypted, err := sqlite.IsEncrypted(oldPath)
	if err != nil {
		return err
	}

	// Nothing to do, already encrypted
	if isEncrypted {
		return os.Rename(oldPath, newPath)
	}

	db, err := openDB(oldPath, "", defaultKdfIterationsNumber)
	if err != nil {
		return err
	}

	attach := fmt.Sprintf(
		"ATTACH DATABASE '%s' AS newdb KEY '%s'",
		newPath,
		key)

	if _, err = db.Exec(attach); err != nil {
		return err
	}

	changeKdfIter := fmt.Sprintf(
		"PRAGMA newdb.kdf_iter = %d",
		kdfIterationsNumber)

	if _, err = db.Exec(changeKdfIter); err != nil {
		return err
	}

	if _, err = db.Exec(exportDB); err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}

	return os.Remove(oldPath)
}

func openDB(path string, key string, kdfIter int) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	if _, err = db.Exec(keyString); err != nil {
		return nil, err
	}

	kdfString := fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIter)

	if _, err = db.Exec(kdfString); err != nil {
		return nil, err
	}
	return db, nil
}

// NewSQLLiteKeysStorage creates a new SQLLiteKeysStorage instance associated with the specified database
func NewSQLLiteKeysStorage(db *sql.DB) *SQLLiteKeysStorage {
	return &SQLLiteKeysStorage{
		db: db,
	}
}

// NewSQLLiteSessionStorage creates a new SQLLiteSessionStorage instance associated with the specified database
func NewSQLLiteSessionStorage(db *sql.DB) *SQLLiteSessionStorage {
	return &SQLLiteSessionStorage{
		db: db,
	}
}

// GetKeysStorage returns the associated double ratchet KeysStorage object
func (s *SQLLitePersistence) GetKeysStorage() dr.KeysStorage {
	return s.keysStorage
}

// GetSessionStorage returns the associated double ratchet SessionStorage object
func (s *SQLLitePersistence) GetSessionStorage() dr.SessionStorage {
	return s.sessionStorage
}

// Open opens a file at the specified path
func (s *SQLLitePersistence) Open(path string, key string) error {
	db, err := openDB(path, key, kdfIterationsNumber)
	if err != nil {
		return err
	}

	s.db = db

	return s.setup()
}

// AddPrivateBundle adds the specified BundleContainer to the database
func (s *SQLLitePersistence) AddPrivateBundle(bc *BundleContainer) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	for installationID, signedPreKey := range bc.GetBundle().GetSignedPreKeys() {
		var version uint32
		stmt, err := tx.Prepare(`SELECT version
					 FROM bundles
					 WHERE installation_id = ? AND identity = ?
					 ORDER BY version DESC
					 LIMIT 1`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		err = stmt.QueryRow(installationID, bc.GetBundle().GetIdentity()).Scan(&version)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		stmt, err = tx.Prepare(`INSERT INTO bundles(identity, private_key, signed_pre_key, installation_id, version, timestamp)
					VALUES(?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.Exec(
			bc.GetBundle().GetIdentity(),
			bc.GetPrivateSignedPreKey(),
			signedPreKey.GetSignedPreKey(),
			installationID,
			version+1,
			bc.GetBundle().GetTimestamp(),
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}

// AddPublicBundle adds the specified Bundle to the database
func (s *SQLLitePersistence) AddPublicBundle(b *Bundle) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	for installationID, signedPreKeyContainer := range b.GetSignedPreKeys() {
		signedPreKey := signedPreKeyContainer.GetSignedPreKey()
		version := signedPreKeyContainer.GetVersion()
		insertStmt, err := tx.Prepare(`INSERT INTO bundles(identity, signed_pre_key, installation_id, version, timestamp)
					       VALUES( ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer insertStmt.Close()

		_, err = insertStmt.Exec(
			b.GetIdentity(),
			signedPreKey,
			installationID,
			version,
			b.GetTimestamp(),
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		// Mark old bundles as expired
		updateStmt, err := tx.Prepare(`UPDATE bundles
					       SET expired = 1
					       WHERE identity = ? AND installation_id = ? AND version < ?`)
		if err != nil {
			return err
		}
		defer updateStmt.Close()

		_, err = updateStmt.Exec(
			b.GetIdentity(),
			installationID,
			version,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}

// GetAnyPrivateBundle retrieves any bundle from the database containing a private key
func (s *SQLLitePersistence) GetAnyPrivateBundle(myIdentityKey []byte, installationIDs []string) (*BundleContainer, error) {

	/* #nosec */
	statement := `SELECT identity, private_key, signed_pre_key, installation_id, timestamp, version
	              FROM bundles
		      WHERE expired = 0 AND identity = ? AND installation_id IN (?` + strings.Repeat(",?", len(installationIDs)-1) + ")"
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var timestamp int64
	var identity []byte
	var privateKey []byte
	var version uint32

	args := make([]interface{}, len(installationIDs)+1)
	args[0] = myIdentityKey
	for i, installationID := range installationIDs {
		args[i+1] = installationID
	}

	rows, err := stmt.Query(args...)
	rowCount := 0

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	bundle := &Bundle{
		SignedPreKeys: make(map[string]*SignedPreKey),
	}

	bundleContainer := &BundleContainer{
		Bundle: bundle,
	}

	for rows.Next() {
		var signedPreKey []byte
		var installationID string
		rowCount++
		err = rows.Scan(
			&identity,
			&privateKey,
			&signedPreKey,
			&installationID,
			&timestamp,
			&version,
		)
		if err != nil {
			return nil, err
		}
		// If there is a private key, we set the timestamp of the bundle container
		if privateKey != nil {
			bundle.Timestamp = timestamp
		}

		bundle.SignedPreKeys[installationID] = &SignedPreKey{SignedPreKey: signedPreKey, Version: version}
		bundle.Identity = identity
	}

	// If no records are found or no record with private key, return nil
	if rowCount == 0 || bundleContainer.GetBundle().Timestamp == 0 {
		return nil, nil
	}

	return bundleContainer, nil

}

// GetPrivateKeyBundle retrieves a private key for a bundle from the database
func (s *SQLLitePersistence) GetPrivateKeyBundle(bundleID []byte) ([]byte, error) {
	stmt, err := s.db.Prepare(`SELECT private_key
				   FROM bundles
				   WHERE expired = 0 AND signed_pre_key = ? LIMIT 1`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var privateKey []byte

	err = stmt.QueryRow(bundleID).Scan(&privateKey)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return privateKey, nil
	default:
		return nil, err
	}
}

// MarkBundleExpired expires any private bundle for a given identity
func (s *SQLLitePersistence) MarkBundleExpired(identity []byte) error {
	stmt, err := s.db.Prepare(`UPDATE bundles
				   SET expired = 1
				   WHERE identity = ? AND private_key IS NOT NULL`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(identity)

	return err
}

// GetPublicBundle retrieves an existing Bundle for the specified public key from the database
func (s *SQLLitePersistence) GetPublicBundle(publicKey *ecdsa.PublicKey, installationIDs []string) (*Bundle, error) {

	if len(installationIDs) == 0 {
		return nil, nil
	}

	identity := crypto.CompressPubkey(publicKey)

	/* #nosec */
	statement := `SELECT signed_pre_key,installation_id, version
		      FROM bundles
		      WHERE expired = 0 AND identity = ? AND installation_id IN (?` + strings.Repeat(",?", len(installationIDs)-1) + `)
		      ORDER BY version DESC`
	stmt, err := s.db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	args := make([]interface{}, len(installationIDs)+1)
	args[0] = identity
	for i, installationID := range installationIDs {
		args[i+1] = installationID
	}

	rows, err := stmt.Query(args...)
	rowCount := 0

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	bundle := &Bundle{
		Identity:      identity,
		SignedPreKeys: make(map[string]*SignedPreKey),
	}

	for rows.Next() {
		var signedPreKey []byte
		var installationID string
		var version uint32
		rowCount++
		err = rows.Scan(
			&signedPreKey,
			&installationID,
			&version,
		)
		if err != nil {
			return nil, err
		}

		bundle.SignedPreKeys[installationID] = &SignedPreKey{
			SignedPreKey: signedPreKey,
			Version:      version,
		}

	}

	if rowCount == 0 {
		return nil, nil
	}

	return bundle, nil

}

// AddRatchetInfo persists the specified ratchet info into the database
func (s *SQLLitePersistence) AddRatchetInfo(key []byte, identity []byte, bundleID []byte, ephemeralKey []byte, installationID string) error {
	stmt, err := s.db.Prepare(`INSERT INTO ratchet_info_v2(symmetric_key, identity, bundle_id, ephemeral_key, installation_id)
				   VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		key,
		identity,
		bundleID,
		ephemeralKey,
		installationID,
	)

	return err
}

// GetRatchetInfo retrieves the existing RatchetInfo for a specified bundle ID and interlocutor public key from the database
func (s *SQLLitePersistence) GetRatchetInfo(bundleID []byte, theirIdentity []byte, installationID string) (*RatchetInfo, error) {
	stmt, err := s.db.Prepare(`SELECT ratchet_info_v2.identity, ratchet_info_v2.symmetric_key, bundles.private_key, bundles.signed_pre_key, ratchet_info_v2.ephemeral_key, ratchet_info_v2.installation_id
				   FROM ratchet_info_v2 JOIN bundles ON bundle_id = signed_pre_key
				   WHERE ratchet_info_v2.identity = ? AND ratchet_info_v2.installation_id = ? AND bundle_id = ?
				   LIMIT 1`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	ratchetInfo := &RatchetInfo{
		BundleID: bundleID,
	}

	err = stmt.QueryRow(theirIdentity, installationID, bundleID).Scan(
		&ratchetInfo.Identity,
		&ratchetInfo.Sk,
		&ratchetInfo.PrivateKey,
		&ratchetInfo.PublicKey,
		&ratchetInfo.EphemeralKey,
		&ratchetInfo.InstallationID,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		ratchetInfo.ID = append(bundleID, []byte(ratchetInfo.InstallationID)...)
		return ratchetInfo, nil
	default:
		return nil, err
	}
}

// GetAnyRatchetInfo retrieves any existing RatchetInfo for a specified interlocutor public key from the database
func (s *SQLLitePersistence) GetAnyRatchetInfo(identity []byte, installationID string) (*RatchetInfo, error) {
	stmt, err := s.db.Prepare(`SELECT symmetric_key, bundles.private_key, signed_pre_key, bundle_id, ephemeral_key
				   FROM ratchet_info_v2 JOIN bundles ON bundle_id = signed_pre_key
				   WHERE expired = 0 AND ratchet_info_v2.identity = ? AND ratchet_info_v2.installation_id = ?
				   LIMIT 1`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	ratchetInfo := &RatchetInfo{
		Identity:       identity,
		InstallationID: installationID,
	}

	err = stmt.QueryRow(identity, installationID).Scan(
		&ratchetInfo.Sk,
		&ratchetInfo.PrivateKey,
		&ratchetInfo.PublicKey,
		&ratchetInfo.BundleID,
		&ratchetInfo.EphemeralKey,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		ratchetInfo.ID = append(ratchetInfo.BundleID, []byte(installationID)...)
		return ratchetInfo, nil
	default:
		return nil, err
	}
}

// RatchetInfoConfirmed clears the ephemeral key in the RatchetInfo
// associated with the specified bundle ID and interlocutor identity public key
func (s *SQLLitePersistence) RatchetInfoConfirmed(bundleID []byte, theirIdentity []byte, installationID string) error {
	stmt, err := s.db.Prepare(`UPDATE ratchet_info_v2
	                           SET ephemeral_key = NULL
				   WHERE identity = ? AND bundle_id = ? AND installation_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		theirIdentity,
		bundleID,
		installationID,
	)

	return err
}

// Get retrieves the message key for a specified public key and message number
func (s *SQLLiteKeysStorage) Get(pubKey dr.Key, msgNum uint) (dr.Key, bool, error) {
	var keyBytes []byte
	var key [32]byte
	stmt, err := s.db.Prepare(`SELECT message_key
	                           FROM keys
				   WHERE public_key = ? AND msg_num = ?
				   LIMIT 1`)

	if err != nil {
		return key, false, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(pubKey[:], msgNum).Scan(&keyBytes)
	switch err {
	case sql.ErrNoRows:
		return key, false, nil
	case nil:
		copy(key[:], keyBytes)
		return key, true, nil
	default:
		return key, false, err
	}
}

// Put stores a key with the specified public key, message number and message key
func (s *SQLLiteKeysStorage) Put(sessionID []byte, pubKey dr.Key, msgNum uint, mk dr.Key, seqNum uint) error {
	stmt, err := s.db.Prepare(`INSERT INTO keys(session_id, public_key, msg_num, message_key, seq_num)
	                           VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		sessionID,
		pubKey[:],
		msgNum,
		mk[:],
		seqNum,
	)

	return err
}

// DeleteOldMks caps remove any key < seq_num, included
func (s *SQLLiteKeysStorage) DeleteOldMks(sessionID []byte, deleteUntil uint) error {
	stmt, err := s.db.Prepare(`DELETE FROM keys
	                           WHERE session_id = ? AND seq_num <= ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		sessionID,
		deleteUntil,
	)

	return err
}

// TruncateMks caps the number of keys to maxKeysPerSession deleting them in FIFO fashion
func (s *SQLLiteKeysStorage) TruncateMks(sessionID []byte, maxKeysPerSession int) error {
	stmt, err := s.db.Prepare(`DELETE FROM keys
				   WHERE rowid IN (SELECT rowid FROM keys WHERE session_id = ? ORDER BY seq_num DESC LIMIT ? OFFSET ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		sessionID,
		// We LIMIT to the max number of rows here, as OFFSET can't be used without a LIMIT
		maxNumberOfRows,
		maxKeysPerSession,
	)

	return err
}

// DeleteMk deletes the key with the specified public key and message key
func (s *SQLLiteKeysStorage) DeleteMk(pubKey dr.Key, msgNum uint) error {
	stmt, err := s.db.Prepare(`DELETE FROM keys
				   WHERE public_key = ? AND msg_num = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		pubKey[:],
		msgNum,
	)

	return err
}

// Count returns the count of keys with the specified public key
func (s *SQLLiteKeysStorage) Count(pubKey dr.Key) (uint, error) {
	stmt, err := s.db.Prepare(`SELECT COUNT(1)
				   FROM keys
				   WHERE public_key = ?`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count uint
	err = stmt.QueryRow(pubKey[:]).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountAll returns the count of keys with the specified public key
func (s *SQLLiteKeysStorage) CountAll() (uint, error) {
	stmt, err := s.db.Prepare(`SELECT COUNT(1)
				   FROM keys`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count uint
	err = stmt.QueryRow().Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// All returns nil
func (s *SQLLiteKeysStorage) All() (map[dr.Key]map[uint]dr.Key, error) {
	return nil, nil
}

// Save persists the specified double ratchet state
func (s *SQLLiteSessionStorage) Save(id []byte, state *dr.State) error {
	dhr := state.DHr[:]
	dhs := state.DHs
	dhsPublic := dhs.PublicKey()
	dhsPrivate := dhs.PrivateKey()
	pn := state.PN
	step := state.Step
	keysCount := state.KeysCount

	rootChainKey := state.RootCh.CK[:]

	sendChainKey := state.SendCh.CK[:]
	sendChainN := state.SendCh.N

	recvChainKey := state.RecvCh.CK[:]
	recvChainN := state.RecvCh.N

	stmt, err := s.db.Prepare(`INSERT INTO sessions(id, dhr, dhs_public, dhs_private, root_chain_key, send_chain_key, send_chain_n, recv_chain_key, recv_chain_n, pn, step, keys_count)
				   VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		id,
		dhr,
		dhsPublic[:],
		dhsPrivate[:],
		rootChainKey,
		sendChainKey,
		sendChainN,
		recvChainKey,
		recvChainN,
		pn,
		step,
		keysCount,
	)

	return err
}

// Load retrieves the double ratchet state for a given ID
func (s *SQLLiteSessionStorage) Load(id []byte) (*dr.State, error) {
	stmt, err := s.db.Prepare(`SELECT dhr, dhs_public, dhs_private, root_chain_key, send_chain_key, send_chain_n, recv_chain_key, recv_chain_n, pn, step, keys_count
				   FROM sessions
				   WHERE id = ?`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	var (
		dhr          []byte
		dhsPublic    []byte
		dhsPrivate   []byte
		rootChainKey []byte
		sendChainKey []byte
		sendChainN   uint
		recvChainKey []byte
		recvChainN   uint
		pn           uint
		step         uint
		keysCount    uint
	)

	err = stmt.QueryRow(id).Scan(
		&dhr,
		&dhsPublic,
		&dhsPrivate,
		&rootChainKey,
		&sendChainKey,
		&sendChainN,
		&recvChainKey,
		&recvChainN,
		&pn,
		&step,
		&keysCount,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		state := dr.DefaultState(toKey(rootChainKey))

		state.PN = uint32(pn)
		state.Step = step
		state.KeysCount = keysCount

		state.DHs = ecrypto.DHPair{
			PrvKey: toKey(dhsPrivate),
			PubKey: toKey(dhsPublic),
		}

		state.DHr = toKey(dhr)

		state.SendCh.CK = toKey(sendChainKey)
		state.SendCh.N = uint32(sendChainN)

		state.RecvCh.CK = toKey(recvChainKey)
		state.RecvCh.N = uint32(recvChainN)

		return &state, nil
	default:
		return nil, err
	}
}

// GetActiveInstallations returns the active installations for a given identity
func (s *SQLLitePersistence) GetActiveInstallations(maxInstallations int, identity []byte) ([]string, error) {
	stmt, err := s.db.Prepare(`SELECT installation_id
				   FROM installations
				   WHERE enabled = 1 AND identity = ?
				   ORDER BY timestamp DESC
				   LIMIT ?`)
	if err != nil {
		return nil, err
	}

	var installations []string
	rows, err := stmt.Query(identity, maxInstallations)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var installationID string
		err = rows.Scan(
			&installationID,
		)
		if err != nil {
			return nil, err
		}
		installations = append(installations, installationID)
	}

	return installations, nil

}

// AddInstallations adds the installations for a given identity, maintaining the enabled flag
func (s *SQLLitePersistence) AddInstallations(identity []byte, timestamp int64, installationIDs []string, defaultEnabled bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}

	for _, installationID := range installationIDs {
		stmt, err := tx.Prepare(`SELECT enabled
					 FROM installations
					 WHERE identity = ? AND installation_id = ?
					 LIMIT 1`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		var oldEnabled bool

		err = stmt.QueryRow(identity, installationID).Scan(&oldEnabled)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// We update timestamp if present without changing enabled
		if err != sql.ErrNoRows {
			stmt, err = tx.Prepare(`UPDATE installations
					        SET timestamp = ?,  enabled = ?
						WHERE identity = ? AND installation_id = ?`)
			if err != nil {
				return err
			}

			_, err = stmt.Exec(
				timestamp,
				oldEnabled,
				identity,
				installationID,
			)
			if err != nil {
				return err
			}
			defer stmt.Close()

		} else {
			stmt, err = tx.Prepare(`INSERT INTO installations(identity, installation_id, timestamp, enabled)
						VALUES (?, ?, ?, ?)`)
			if err != nil {
				return err
			}

			_, err = stmt.Exec(
				identity,
				installationID,
				timestamp,
				defaultEnabled,
			)
			if err != nil {
				return err
			}
			defer stmt.Close()
		}

	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil

}

// EnableInstallation enables the installation
func (s *SQLLitePersistence) EnableInstallation(identity []byte, installationID string) error {
	stmt, err := s.db.Prepare(`UPDATE installations
				   SET enabled = 1
				   WHERE identity = ? AND installation_id = ?`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(identity, installationID)
	return err

}

// DisableInstallation disable the installation
func (s *SQLLitePersistence) DisableInstallation(identity []byte, installationID string) error {

	stmt, err := s.db.Prepare(`UPDATE installations
				   SET enabled = 0
				   WHERE identity = ? AND installation_id = ?`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(identity, installationID)
	return err
}

func toKey(a []byte) dr.Key {
	var k [32]byte
	copy(k[:], a)
	return k
}

func (s *SQLLitePersistence) setup() error {
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

	driver, err := sqlcipher.WithInstance(s.db, &sqlcipher.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}

	return nil
}
