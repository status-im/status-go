// Command fix-wallet-migrations repairs the migration version marker of a
// wallet database that a newer build has migrated.
//
// A wallet database migrated by a newer status-go carries a version this build
// does not know, and the migrate library refuses to open it. Rolling the marker
// back to the newest version this build knows lets it open the file again, as
// long as the newer migrations did not drop or rename anything this build uses;
// the tool cannot check that, since those migrations only exist in the newer
// build. It only ever touches the wallet database: the app database holds seven
// independent migration sets and its history is not additive.
//
//	fix-wallet-migrations                          list the wallet migrations this build knows
//	fix-wallet-migrations <db|datadir> <password>  open the wallet database as status-go does and repair the marker
//
// The database may be given as the <keyUID>-wallet.db file or as the profile
// data dir, when it holds a single profile. The password is the login password;
// it is hashed the way status-desktop does before status-go ever sees it.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/db/walletdb/migrations"
)

func main() {
	var err error
	switch len(os.Args) {
	case 1:
		err = list()
	case 3:
		err = fix(os.Args[1], os.Args[2])
	default:
		err = fmt.Errorf("usage: %s [<db|datadir> <password>]", filepath.Base(os.Args[0]))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// list prints the migrations compiled into this build; the last one is what
// this build expects to find in the database.
func list() error {
	known := knownVersions()
	fmt.Println("migrations known to this build:")
	for _, v := range known {
		fmt.Printf("  %d\n", v)
	}
	fmt.Printf("this build expects version %d; older builds can roll back to any version above\n", known[len(known)-1])
	return nil
}

// fix reads the marker and repairs it when this build could not open the database.
func fix(dbOrDataDir, password string) error {
	dbPath, err := resolveDBPath(dbOrDataDir)
	if err != nil {
		return err
	}
	fmt.Println("database:", dbPath)

	db, err := openWallet(dbPath, password)
	if err != nil {
		return err
	}
	defer db.Close()

	version, dirty, err := readMarker(db)
	if err != nil {
		return err
	}
	fmt.Printf("database marker: version %d, dirty %t\n", version, dirty)

	known := knownVersions()
	latest := known[len(known)-1]
	switch {
	case version > latest:
		if err := writeMarker(db, latest); err != nil {
			return err
		}
		fmt.Printf("rolled back %d -> %d, the newest version this build knows\n", version, latest)
		fmt.Println("note: this build cannot inspect the newer migrations; if one of them dropped or renamed something this build uses, it will fail to start - restore the profile from the seed phrase in that case")
	case dirty:
		if err := writeMarker(db, version); err != nil {
			return err
		}
		fmt.Printf("cleared the dirty flag at version %d\n", version)
	default:
		fmt.Printf("nothing to do: this build knows version %d and the marker is clean\n", version)
	}
	return nil
}

// resolveDBPath accepts the wallet DB itself, or a data dir / `data` folder
// holding exactly one <keyUID>-wallet.db.
func resolveDBPath(input string) (string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return input, nil
	}
	dataDir := input
	if sub, err := os.Stat(filepath.Join(input, "data")); err == nil && sub.IsDir() {
		dataDir = filepath.Join(input, "data")
	}
	wallets, err := filepath.Glob(filepath.Join(dataDir, "*-wallet.db"))
	if err != nil {
		return "", err
	}
	switch len(wallets) {
	case 1:
		return wallets[0], nil
	case 0:
		return "", fmt.Errorf("no *-wallet.db in %s", dataDir)
	default:
		return "", fmt.Errorf("%s holds several profiles, pass the wallet DB explicitly:\n  %s", dataDir, strings.Join(wallets, "\n  "))
	}
}

// openWallet opens the wallet database the way the backend does. Like
// unwrap-kek it tries the password both hashed (the client convention) and
// raw (keycard profiles hand status-go the encryption public key instead).
func openWallet(dbPath, password string) (*sql.DB, error) {
	dataDir := filepath.Dir(dbPath)
	keyUID := strings.TrimSuffix(filepath.Base(dbPath), "-wallet.db")
	account := readAccount(dataDir, keyUID)
	candidates := []string{hashPassword(password), password}

	// DEK profile: the database key is wrapped in <keyUID>-profile.kek.
	if envelope.Exists(dataDir, keyUID) {
		fmt.Println("profile: DEK", envelope.Path(dataDir, keyUID))
		for _, kek := range candidates {
			dek, kdfIter, err := envelope.Unwrap(dataDir, keyUID, kek)
			if err == nil {
				return sqlite.OpenDB(dbPath, dek, kdfIter)
			}
		}
		return nil, fmt.Errorf("wrong password for this profile (envelope MAC mismatch)%s", account.hint())
	}

	// Legacy profile: the password itself is the sqlcipher passphrase, with the
	// KDF iteration count the profile was created with.
	fmt.Printf("profile: legacy, kdf iterations %d\n", account.kdfIterations)
	for _, secret := range candidates {
		if db, err := sqlite.OpenDB(dbPath, secret, account.kdfIterations); err == nil {
			return db, nil
		}
	}
	return nil, fmt.Errorf("could not open the database with this password%s", account.hint())
}

type accountInfo struct {
	kdfIterations int
	keycard       bool
	found         bool
}

func (a accountInfo) hint() string {
	switch {
	case a.keycard:
		return "; this is a keycard profile, pass the keycard encryption public key instead of a password"
	case !a.found:
		return fmt.Sprintf("; no accounts.sql entry found, assumed %d kdf iterations", a.kdfIterations)
	}
	return ""
}

// readAccount reads the profile's KDF iteration count and keycard flag from
// the unencrypted accounts.sql. It opens the file read-only on purpose: the
// multiaccounts package would run its migrations on open, and a repair tool
// must not touch anything but the marker it came for.
func readAccount(dataDir, keyUID string) accountInfo {
	info := accountInfo{kdfIterations: sqlite.ReducedKDFIterationsNumber}
	path := filepath.Join(dataDir, "accounts.sql")
	if _, err := os.Stat(path); err != nil {
		return info
	}
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		return info
	}
	defer db.Close()

	var kdf int
	var pairing string
	if err := db.QueryRow("SELECT kdfIterations, keycardPairing FROM accounts WHERE keyUid = ?", keyUID).Scan(&kdf, &pairing); err != nil {
		return info
	}
	info.found = true
	info.keycard = pairing != ""
	if kdf > 0 {
		info.kdfIterations = kdf
	}
	return info
}

// hashPassword derives the KEK the way status-desktop does: "0x" +
// keccak256(password), lowercase hex. For a migrated profile it unwraps the DEK
// from the envelope; for a legacy profile (no envelope until the next password
// change) it is the database key itself, as in the backend's
// resolveProfileSecret. A value already in that form is used as is, so the
// hashed password can be passed too, as with unwrap-kek.
func hashPassword(password string) string {
	if len(password) == 66 && strings.HasPrefix(password, "0x") {
		return strings.ToLower(password)
	}
	return fmt.Sprintf("0x%x", crypto.Keccak256([]byte(password)))
}

// knownVersions parses the <version>_<name>.up.sql names compiled into this build.
func knownVersions() []uint64 {
	var versions []uint64
	for _, name := range migrations.AssetNames() {
		prefix, _, _ := strings.Cut(filepath.Base(name), "_")
		if v, err := strconv.ParseUint(prefix, 10, 64); err == nil {
			versions = append(versions, v)
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions
}

// readMarker returns the single row of the migrations table. The migrate
// library stores dirty as the strings 'true'/'false'.
func readMarker(db *sql.DB) (version uint64, dirty bool, err error) {
	var dirtyText string
	query := fmt.Sprintf("SELECT version, CAST(dirty AS TEXT) FROM %s LIMIT 1", sqlite.StatusMigrationTableName())
	if err := db.QueryRow(query).Scan(&version, &dirtyText); err != nil {
		return 0, false, fmt.Errorf("reading %s: %w", sqlite.StatusMigrationTableName(), err)
	}
	return version, dirtyText == "true" || dirtyText == "1", nil
}

// writeMarker does exactly what the migrate library's SetVersion does:
// DELETE, then INSERT one row.
func writeMarker(db *sql.DB, version uint64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	table := sqlite.StatusMigrationTableName()
	if _, err := tx.Exec("DELETE FROM " + table); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (version, dirty) VALUES (?, 'false')", table), version); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
