package accounts

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

var errKeycardDbTransactionIsNil = errors.New("keycard: database transaction is nil")

type Keycard struct {
	KeycardUID        string          `json:"keycard-uid"`
	KeycardName       string          `json:"keycard-name"`
	KeycardLocked     bool            `json:"keycard-locked"`
	AccountsAddresses []types.Address `json:"accounts-addresses"`
	KeyUID            string          `json:"key-uid"`
	LastUpdateClock   uint64
}

type KeycardAction struct {
	Action        string   `json:"action"`
	OldKeycardUID string   `json:"old-keycard-uid,omitempty"`
	Keycard       *Keycard `json:"keycard"`
}

func (kp *Keycard) ToSyncKeycard() *protobuf.SyncKeycard {
	kc := &protobuf.SyncKeycard{
		Uid:    kp.KeycardUID,
		Name:   kp.KeycardName,
		Locked: kp.KeycardLocked,
		KeyUid: kp.KeyUID,
		Clock:  kp.LastUpdateClock,
	}

	for _, addr := range kp.AccountsAddresses {
		kc.Addresses = append(kc.Addresses, addr.Bytes())
	}

	return kc
}

func (kp *Keycard) FromSyncKeycard(kc *protobuf.SyncKeycard) {
	kp.KeycardUID = kc.Uid
	kp.KeycardName = kc.Name
	kp.KeycardLocked = kc.Locked
	kp.KeyUID = kc.KeyUid
	kp.LastUpdateClock = kc.Clock

	for _, addr := range kc.Addresses {
		kp.AccountsAddresses = append(kp.AccountsAddresses, types.BytesToAddress(addr))
	}
}

func removeElementAtIndex[T any](s []T, index int) []T {
	if index < 0 || index >= len(s) {
		panic("keycard: index out of the range")
	}
	return append(s[:index], s[index+1:]...)
}

type Keycards struct {
	db *sql.DB
}

func NewKeycards(db *sql.DB) *Keycards {
	return &Keycards{
		db: db,
	}
}

func containsAddress(addresses []types.Address, address types.Address) bool {
	for _, addr := range addresses {
		if addr == address {
			return true
		}
	}
	return false
}

func (kp *Keycards) processResult(rows *sql.Rows, groupByKeycard bool) ([]*Keycard, error) {
	keycards := []*Keycard{}
	for rows.Next() {
		keycard := &Keycard{}
		addr := types.Address{}
		err := rows.Scan(&keycard.KeycardUID, &keycard.KeycardName, &keycard.KeycardLocked, &addr, &keycard.KeyUID,
			&keycard.LastUpdateClock)
		if err != nil {
			return nil, err
		}

		foundAtIndex := -1
		for i := range keycards {
			if groupByKeycard {
				if keycards[i].KeycardUID == keycard.KeycardUID {
					foundAtIndex = i
					break
				}
			} else {
				if keycards[i].KeyUID == keycard.KeyUID {
					foundAtIndex = i
					break
				}
			}
		}
		if foundAtIndex == -1 {
			keycard.AccountsAddresses = append(keycard.AccountsAddresses, addr)
			keycards = append(keycards, keycard)
		} else {
			if containsAddress(keycards[foundAtIndex].AccountsAddresses, addr) {
				continue
			}
			keycards[foundAtIndex].AccountsAddresses = append(keycards[foundAtIndex].AccountsAddresses, addr)
		}
	}

	return keycards, nil
}

func (kp *Keycards) getAllRows(tx *sql.Tx, groupByKeycard bool) ([]*Keycard, error) {
	var (
		rows *sql.Rows
		err  error
	)
	query := // nolint: gosec
		`
		SELECT
			k.keycard_uid,
			k.keycard_name,
			k.keycard_locked,
			ka.account_address,
			k.key_uid,
			k.last_update_clock
		FROM
			keycards AS k
		LEFT JOIN
			keycards_accounts AS ka
		ON
			k.keycard_uid = ka.keycard_uid
		ORDER BY
			key_uid`

	if tx == nil {
		rows, err = kp.db.Query(query)
		if err != nil {
			return nil, err
		}
	} else {
		stmt, err := tx.Prepare(query)
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		rows, err = stmt.Query()
		if err != nil {
			return nil, err
		}
	}

	defer rows.Close()
	return kp.processResult(rows, groupByKeycard)
}

func (kp *Keycards) GetAllKnownKeycards() ([]*Keycard, error) {
	return kp.getAllRows(nil, true)
}

func (kp *Keycards) GetAllKnownKeycardsGroupedByKeyUID() ([]*Keycard, error) {
	return kp.getAllRows(nil, false)
}

func (kp *Keycards) GetKeycardByKeyUID(keyUID string) ([]*Keycard, error) {
	rows, err := kp.db.Query(`
		SELECT
			k.keycard_uid,
			k.keycard_name,
			k.keycard_locked,
			ka.account_address,
			k.key_uid,
			k.last_update_clock
		FROM
			keycards AS k
		LEFT JOIN
			keycards_accounts AS ka
		ON
			k.keycard_uid = ka.keycard_uid
		WHERE
			k.key_uid = ?
		ORDER BY
			k.keycard_uid
	`, keyUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*Keycard{}, nil
		}
		return nil, err
	}

	defer rows.Close()
	return kp.processResult(rows, false)
}

func (kp *Keycards) startTransactionAndCheckIfNeedToProceed(kcUID string, clock uint64) (tx *sql.Tx, proceed bool, err error) {
	tx, err = kp.db.Begin()
	if err != nil {
		return nil, false, err
	}
	var dbLastUpdateClock uint64
	err = tx.QueryRow(`SELECT last_update_clock FROM keycards WHERE keycard_uid = ?`, kcUID).Scan(&dbLastUpdateClock)
	if err != nil {
		return tx, err == sql.ErrNoRows, err
	}

	return tx, dbLastUpdateClock <= clock, nil
}

func (kp *Keycards) setLastUpdateClock(tx *sql.Tx, kcUID string, clock uint64) (err error) {
	if tx == nil {
		return errKeycardDbTransactionIsNil
	}

	_, err = tx.Exec(`
		UPDATE
			keycards
		SET
			last_update_clock = ?
		WHERE
			keycard_uid = ?`,
		clock, kcUID)

	return err
}

func (kp *Keycards) getAccountsForKeycard(tx *sql.Tx, kcUID string) ([]types.Address, error) {
	var accountAddresses []types.Address
	if tx == nil {
		return accountAddresses, errKeycardDbTransactionIsNil
	}

	rows, err := tx.Query(`SELECT account_address FROM keycards_accounts WHERE keycard_uid = ?`, kcUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var accAddress types.Address
		err = rows.Scan(&accAddress)
		if err != nil {
			return nil, err
		}
		accountAddresses = append(accountAddresses, accAddress)
	}

	return accountAddresses, nil
}

func (kp *Keycards) addAccounts(tx *sql.Tx, kcUID string, accountsAddresses []types.Address) (err error) {
	if tx == nil {
		return errKeycardDbTransactionIsNil
	}

	insertKcAcc, err := tx.Prepare(`
		INSERT INTO
			keycards_accounts
			(
				keycard_uid,
				account_address
			)
		VALUES
			(?, ?);
	`)

	if err != nil {
		return err
	}
	defer insertKcAcc.Close()

	for i := range accountsAddresses {
		addr := accountsAddresses[i]

		_, err = insertKcAcc.Exec(kcUID, addr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (kp *Keycards) deleteKeycard(tx *sql.Tx, kcUID string) (err error) {
	if tx == nil {
		return errKeycardDbTransactionIsNil
	}

	delete, err := tx.Prepare(`
		DELETE
		FROM
			keycards
		WHERE
			keycard_uid = ?
	`)
	if err != nil {
		return err
	}
	defer delete.Close()

	_, err = delete.Exec(kcUID)

	return err
}

func (kp *Keycards) AddKeycardOrAddAccountsIfKeycardIsAdded(keycard Keycard) (addedKc bool, addedAccs bool, err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(keycard.KeycardUID, keycard.LastUpdateClock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if proceed {
		// insert only if there is no such keycard, otherwise just add accounts
		if err != nil && err == sql.ErrNoRows {
			_, err = tx.Exec(`
				INSERT INTO
					keycards
					(
						keycard_uid,
						keycard_name,
						keycard_locked,
						key_uid,
						last_update_clock
					)
				VALUES
					(?, ?, ?, ?, ?);`,
				keycard.KeycardUID, keycard.KeycardName, keycard.KeycardLocked, keycard.KeyUID, keycard.LastUpdateClock)

			if err != nil {
				return false, false, err
			}

			err = kp.addAccounts(tx, keycard.KeycardUID, keycard.AccountsAddresses)
			return err == nil, false, err
		}

		err = kp.setLastUpdateClock(tx, keycard.KeycardUID, keycard.LastUpdateClock)
		if err != nil {
			return false, false, err
		}

		err = kp.addAccounts(tx, keycard.KeycardUID, keycard.AccountsAddresses)
		return false, err == nil, err
	}

	return false, false, err
}

func (kp *Keycards) ApplyKeycardsForKeypairWithKeyUID(keyUID string, keycardsToSync []*Keycard) (err error) {
	tx, err := kp.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	rows, err := tx.Query(`SELECT * FROM keycards WHERE key_uid = ?`, keyUID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	defer rows.Close()

	var dbKeycards []*Keycard
	for rows.Next() {
		keycard := &Keycard{}
		err := rows.Scan(&keycard.KeycardUID, &keycard.KeycardName, &keycard.KeycardLocked, &keycard.KeyUID,
			&keycard.LastUpdateClock)
		if err != nil {
			return err
		}

		dbKeycards = append(dbKeycards, keycard)
	}

	// apply those from `keycardsToSync` which are newer
	for _, syncKc := range keycardsToSync {
		foundAtIndex := -1
		for i := range dbKeycards {
			if dbKeycards[i].KeycardUID == syncKc.KeycardUID {
				foundAtIndex = i
				break
			}
		}

		if foundAtIndex > -1 {
			dbClock := dbKeycards[foundAtIndex].LastUpdateClock
			dbKeycards = removeElementAtIndex(dbKeycards, foundAtIndex)

			if dbClock > syncKc.LastUpdateClock {
				continue
			}
			err = kp.deleteKeycard(tx, syncKc.KeycardUID)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO
				keycards
				(
					keycard_uid,
					keycard_name,
					keycard_locked,
					key_uid,
					last_update_clock
				)
			VALUES
				(?, ?, ?, ?, ?);`,
			syncKc.KeycardUID, syncKc.KeycardName, syncKc.KeycardLocked, syncKc.KeyUID, syncKc.LastUpdateClock)

		if err != nil {
			return err
		}

		err = kp.addAccounts(tx, syncKc.KeycardUID, syncKc.AccountsAddresses)
		if err != nil {
			return err
		}
	}

	// remove those from the db if they are not in `keycardsToSync`
	for _, dbKp := range dbKeycards {
		err = kp.deleteKeycard(tx, dbKp.KeycardUID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (kp *Keycards) RemoveMigratedAccountsForKeycard(kcUID string, accountAddresses []types.Address,
	clock uint64) (err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return err
	}

	if proceed {
		err = kp.setLastUpdateClock(tx, kcUID, clock)
		if err != nil {
			return err
		}

		dbAccountAddresses, err := kp.getAccountsForKeycard(tx, kcUID)
		if err != nil {
			return err
		}
		deleteKeycard := true
		for _, dbAddr := range dbAccountAddresses {
			found := false
			for _, addr := range accountAddresses {
				if dbAddr == addr {
					found = true
				}
			}
			if !found {
				deleteKeycard = false
			}
		}

		if deleteKeycard {
			return kp.deleteKeycard(tx, kcUID)
		}

		inVector := strings.Repeat(",?", len(accountAddresses)-1)
		query := `
			DELETE
			FROM
				keycards_accounts
			WHERE
				keycard_uid = ?
			AND
				account_address	IN (?` + inVector + `)
		`
		delete, err := tx.Prepare(query)
		if err != nil {
			return err
		}

		args := make([]interface{}, len(accountAddresses)+1)
		args[0] = kcUID
		for i, addr := range accountAddresses {
			args[i+1] = addr
		}

		defer delete.Close()

		_, err = delete.Exec(args...)

		return err
	}

	return err
}

func (kp *Keycards) execUpdateQuery(kcUID string, clock uint64, field string, value interface{}) (err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return err
	}

	if proceed {
		sql := fmt.Sprintf(`UPDATE keycards SET %s = ?, last_update_clock = ?	WHERE keycard_uid = ?`, field) // nolint: gosec
		_, err = tx.Exec(sql, value, clock, kcUID)
		return err
	}

	return nil
}

func (kp *Keycards) KeycardLocked(kcUID string, clock uint64) (err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_locked", true)
}

func (kp *Keycards) KeycardUnlocked(kcUID string, clock uint64) (err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_locked", false)
}

func (kp *Keycards) UpdateKeycardUID(oldKcUID string, newKcUID string, clock uint64) (err error) {
	return kp.execUpdateQuery(oldKcUID, clock, "keycard_uid", newKcUID)
}

func (kp *Keycards) SetKeycardName(kcUID string, kpName string, clock uint64) (err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_name", kpName)
}

func (kp *Keycards) DeleteKeycard(kcUID string, clock uint64) (err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return err
	}

	if proceed {
		return kp.deleteKeycard(tx, kcUID)
	}

	return err
}

func (kp *Keycards) DeleteAllKeycardsWithKeyUID(keyUID string) (err error) {
	delete, err := kp.db.Prepare(`
		DELETE
		FROM
			keycards
		WHERE
			key_uid = ?
	`)
	if err != nil {
		return err
	}
	defer delete.Close()
	_, err = delete.Exec(keyUID)
	return err
}
