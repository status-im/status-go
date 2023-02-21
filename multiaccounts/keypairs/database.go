package keypairs

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
)

const (
	dbTransactionIsNil = "database transaction is nil"
)

type KeyPair struct {
	KeycardUID        string          `json:"keycard-uid"`
	KeycardName       string          `json:"keycard-name"`
	KeycardLocked     bool            `json:"keycard-locked"`
	AccountsAddresses []types.Address `json:"accounts-addresses"`
	KeyUID            string          `json:"key-uid"`
	LastUpdateClock   uint64
}

type KeyPairs struct {
	db *sql.DB
}

func NewKeyPairs(db *sql.DB) *KeyPairs {
	return &KeyPairs{
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

func (kp *KeyPairs) processResult(rows *sql.Rows, groupByKeycard bool) ([]*KeyPair, error) {
	keyPairs := []*KeyPair{}
	for rows.Next() {
		keyPair := &KeyPair{}
		addr := types.Address{}
		err := rows.Scan(&keyPair.KeycardUID, &keyPair.KeycardName, &keyPair.KeycardLocked, &addr, &keyPair.KeyUID,
			&keyPair.LastUpdateClock)
		if err != nil {
			return nil, err
		}

		foundAtIndex := -1
		for i := range keyPairs {
			if groupByKeycard {
				if keyPairs[i].KeycardUID == keyPair.KeycardUID {
					foundAtIndex = i
					break
				}
			} else {
				if keyPairs[i].KeyUID == keyPair.KeyUID {
					foundAtIndex = i
					break
				}
			}
		}
		if foundAtIndex == -1 {
			keyPair.AccountsAddresses = append(keyPair.AccountsAddresses, addr)
			keyPairs = append(keyPairs, keyPair)
		} else {
			if containsAddress(keyPairs[foundAtIndex].AccountsAddresses, addr) {
				continue
			}
			keyPairs[foundAtIndex].AccountsAddresses = append(keyPairs[foundAtIndex].AccountsAddresses, addr)
		}
	}

	return keyPairs, nil
}

func (kp *KeyPairs) getAllRows(groupByKeycard bool) ([]*KeyPair, error) {
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
		ORDER BY
			key_uid
	`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return kp.processResult(rows, groupByKeycard)
}

func (kp *KeyPairs) GetAllKnownKeycards() ([]*KeyPair, error) {
	return kp.getAllRows(true)
}

func (kp *KeyPairs) GetAllMigratedKeyPairs() ([]*KeyPair, error) {
	return kp.getAllRows(false)
}

func (kp *KeyPairs) GetMigratedKeyPairByKeyUID(keyUID string) ([]*KeyPair, error) {
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
		return nil, err
	}

	defer rows.Close()
	return kp.processResult(rows, false)
}

func (kp *KeyPairs) startTransactionAndCheckIfNeedToProceed(kcUID string, clock uint64) (tx *sql.Tx, proceed bool, err error) {
	tx, err = kp.db.Begin()
	if err != nil {
		return nil, false, err
	}
	var dbLastUpdateClock uint64
	err = tx.QueryRow(`SELECT last_update_clock FROM keycards WHERE keycard_uid = ?`, kcUID).Scan(&dbLastUpdateClock)
	if err != nil {
		return tx, false, err
	}

	return tx, dbLastUpdateClock <= clock, nil
}

func (kp *KeyPairs) setLastUpdateClock(tx *sql.Tx, kcUID string, clock uint64) (err error) {
	if tx == nil {
		return errors.New(dbTransactionIsNil)
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

func (kp *KeyPairs) getAccountsForKeycard(tx *sql.Tx, kcUID string) ([]types.Address, error) {
	var accountAddresses []types.Address
	if tx == nil {
		return accountAddresses, errors.New(dbTransactionIsNil)
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

func (kp *KeyPairs) addAccounts(tx *sql.Tx, kcUID string, accountsAddresses []types.Address) (err error) {
	if tx == nil {
		return errors.New(dbTransactionIsNil)
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

func (kp *KeyPairs) deleteKeycard(tx *sql.Tx, kcUID string) (err error) {
	if tx == nil {
		return errors.New(dbTransactionIsNil)
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

func (kp *KeyPairs) AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair KeyPair) (added bool, err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(keyPair.KeycardUID, keyPair.LastUpdateClock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		if err == sql.ErrNoRows {
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
				keyPair.KeycardUID, keyPair.KeycardName, keyPair.KeycardLocked, keyPair.KeyUID, keyPair.LastUpdateClock)

			if err != nil {
				return false, err
			}

			err = kp.addAccounts(tx, keyPair.KeycardUID, keyPair.AccountsAddresses)
			return err == nil, err
		}

		return false, err
	}

	if proceed {
		err = kp.setLastUpdateClock(tx, keyPair.KeycardUID, keyPair.LastUpdateClock)
		if err != nil {
			return false, err
		}

		err = kp.addAccounts(tx, keyPair.KeycardUID, keyPair.AccountsAddresses)
		return err == nil, err
	}

	return false, nil
}

func (kp *KeyPairs) RemoveMigratedAccountsForKeycard(kcUID string, accountAddresses []types.Address,
	clock uint64) (removed bool, err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return false, err
	}

	if proceed {
		err = kp.setLastUpdateClock(tx, kcUID, clock)
		if err != nil {
			return false, err
		}

		dbAccountAddresses, err := kp.getAccountsForKeycard(tx, kcUID)
		if err != nil {
			return false, err
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
			err = kp.deleteKeycard(tx, kcUID)
			return err == nil, err
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
			return false, err
		}

		args := make([]interface{}, len(accountAddresses)+1)
		args[0] = kcUID
		for i, addr := range accountAddresses {
			args[i+1] = addr
		}

		defer delete.Close()

		_, err = delete.Exec(args...)

		return true, err
	}

	return false, nil
}

func (kp *KeyPairs) execUpdateQuery(kcUID string, clock uint64, field string, value interface{}) (updated bool, err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return false, err
	}

	if proceed {
		sql := fmt.Sprintf(`UPDATE keycards SET %s = ?, last_update_clock = ?	WHERE keycard_uid = ?`, field) // nolint: gosec
		_, err = tx.Exec(sql, value, clock, kcUID)
		return err == nil, err
	}

	return false, nil
}

func (kp *KeyPairs) KeycardLocked(kcUID string, clock uint64) (updated bool, err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_locked", true)
}

func (kp *KeyPairs) KeycardUnlocked(kcUID string, clock uint64) (updated bool, err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_locked", false)
}

func (kp *KeyPairs) UpdateKeycardUID(oldKcUID string, newKcUID string, clock uint64) (updated bool, err error) {
	return kp.execUpdateQuery(oldKcUID, clock, "keycard_uid", newKcUID)
}

func (kp *KeyPairs) SetKeycardName(kcUID string, kpName string, clock uint64) (updated bool, err error) {
	return kp.execUpdateQuery(kcUID, clock, "keycard_name", kpName)
}

func (kp *KeyPairs) DeleteKeycard(kcUID string, clock uint64) (deleted bool, err error) {
	tx, proceed, err := kp.startTransactionAndCheckIfNeedToProceed(kcUID, clock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if err != nil {
		return false, err
	}

	if proceed {
		err = kp.deleteKeycard(tx, kcUID)
		return err == nil, err
	}

	return false, nil
}

func (kp *KeyPairs) DeleteKeypair(keyUID string) (err error) {
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
