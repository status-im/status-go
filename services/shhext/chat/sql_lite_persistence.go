package chat

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	_ "github.com/mutecomm/go-sqlcipher"
)

type SqlLitePersistence struct {
	db *sql.DB
}

func createPublicBundleTableStatement() string {
	return `CREATE TABLE IF NOT EXISTS public_bundles (id BLOB NOT NULL PRIMARY KEY, bundle BLOB NOT NULL)`
}

func createPrivateBundleTableStatement() string {
	return `CREATE TABLE IF NOT EXISTS private_bundles (id BLOB NOT NULL PRIMARY KEY, bundle BLOB NOT NULL)`
}

func createSymmetricKeyTableStatement() string {
	return `CREATE TABLE IF NOT EXISTS symmetric_keys (id BLOB NOT NULL PRIMARY KEY, identity BLOB NOT NULL, ephemeral BLOB NOT NULL)`
}

func (p *SqlLitePersistence) setup() error {
	_, err := p.db.Exec(createPublicBundleTableStatement())
	if err != nil {
		return err
	}

	_, err = p.db.Exec(createPrivateBundleTableStatement())
	if err != nil {
		return err
	}

	_, err = p.db.Exec(createSymmetricKeyTableStatement())
	if err != nil {
		return err
	}

	return nil
}

func NewSqlLitePersistence(path string, key string) (*SqlLitePersistence, error) {
	p := &SqlLitePersistence{}
	err := p.Open(path, key)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *SqlLitePersistence) Open(path string, key string) error {

	dbname := fmt.Sprintf("%s?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", path, key)
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		return err
	}
	p.db = db

	p.setup()

	return nil
}

func (p *SqlLitePersistence) AddPrivateBundle(b *BundleContainer) error {
	stmt, err := p.db.Prepare("insert into private_bundles(id, bundle) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	marshaledBundle, err := proto.Marshal(b)
	if err != nil {
		return err
	}

	id := b.GetBundle().GetSignedPreKey()

	_, err = stmt.Exec(id, marshaledBundle)

	return err
}

func (p *SqlLitePersistence) GetAnyPrivateBundle() (*Bundle, error) {
	stmt := "SELECT bundle FROM private_bundles LIMIT 1"

	var bundleBytes []byte
	bundle := &BundleContainer{}

	err := p.db.QueryRow(stmt).Scan(&bundleBytes)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(bundleBytes, bundle)
	if err != nil {
		return nil, err
	}

	return bundle.GetBundle(), nil
}

func (p *SqlLitePersistence) AddPublicBundle(b *Bundle) error {
	stmt, err := p.db.Prepare("insert into public_bundles(id, bundle) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	marshaledBundle, err := proto.Marshal(b)
	if err != nil {
		return err
	}

	id := b.GetIdentity()

	_, err = stmt.Exec(id, marshaledBundle)

	return err
}

func (p *SqlLitePersistence) AddSymmetricKey(identity *ecdsa.PublicKey, ephemeral *ecdsa.PublicKey, key []byte) error {
	stmt, err := p.db.Prepare("insert into symmetric_keys(id, identity, ephemeral) values(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		key,
		crypto.CompressPubkey(identity),
		crypto.CompressPubkey(ephemeral),
	)

	return err
}

func (s *SqlLitePersistence) GetAnySymmetricKey(identityKey *ecdsa.PublicKey) ([]byte, *ecdsa.PublicKey, error) {
	stmt, err := s.db.Prepare("SELECT id, ephemeral FROM symmetric_keys WHERE identity = ? LIMIT 1")
	if err != nil {
		return nil, nil, err
	}
	defer stmt.Close()

	var key []byte
	var ephemeralBytes []byte

	err = stmt.QueryRow(
		crypto.CompressPubkey(identityKey),
	).Scan(&key, &ephemeralBytes)

	if err == sql.ErrNoRows {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	ephemeral, err := crypto.DecompressPubkey(ephemeralBytes)
	if err != nil {
		return nil, nil, err
	}

	return key, ephemeral, nil

}

func (s *SqlLitePersistence) GetSymmetricKey(identityKey *ecdsa.PublicKey, ephemeralKey *ecdsa.PublicKey) ([]byte, error) {
	stmt, err := s.db.Prepare("SELECT id FROM symmetric_keys WHERE identity = ? AND ephemeral = ? LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var key []byte

	err = stmt.QueryRow(
		crypto.CompressPubkey(identityKey),
		crypto.CompressPubkey(ephemeralKey),
	).Scan(&key)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return key, nil

}

func (s *SqlLitePersistence) GetPrivateBundle(bundleID []byte) (*BundleContainer, error) {
	stmt, err := s.db.Prepare("SELECT bundle FROM private_bundles WHERE id = ? LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var bundleBytes []byte
	bundle := &BundleContainer{}

	err = stmt.QueryRow(bundleID).Scan(&bundleBytes)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(bundleBytes, bundle)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

func (s *SqlLitePersistence) GetPublicBundle(publicKey *ecdsa.PublicKey) (*Bundle, error) {
	bundleID := crypto.CompressPubkey(publicKey)
	stmt, err := s.db.Prepare("SELECT bundle FROM public_bundles WHERE id = ? LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var bundleBytes []byte
	bundle := &Bundle{}

	err = stmt.QueryRow(bundleID).Scan(&bundleBytes)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(bundleBytes, bundle)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

func (s *SqlLitePersistence) GetRatchetInfo(*ecdsa.PublicKey) (*RatchetInfo, error) {
	return nil, nil
}
