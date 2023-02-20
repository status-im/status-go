package keypairs

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/status-im/status-go/eth-node/types"
)

type KeyPair struct {
	KeycardUID        string          `json:"keycard-uid"`
	KeycardName       string          `json:"keycard-name"`
	KeycardLocked     bool            `json:"keycard-locked"`
	AccountsAddresses []types.Address `json:"accounts-addresses"`
	KeyUID            string          `json:"key-uid"`
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
		err := rows.Scan(&keyPair.KeycardUID, &keyPair.KeycardName, &keyPair.KeycardLocked, &addr, &keyPair.KeyUID)
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
			k.key_uid
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
			k.key_uid
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

func (kp *KeyPairs) AddMigratedKeyPairOrAddAccountsIfKeyPairIsAdded(keyPair KeyPair) (err error) {
	var (
		tx          *sql.Tx
		insertKcAcc *sql.Stmt
	)
	tx, err = kp.db.Begin()
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

	var tmpKeyUID string
	err = tx.QueryRow(`SELECT keycard_uid FROM keycards WHERE keycard_uid = ?`, keyPair.KeycardUID).Scan(&tmpKeyUID)
	if err != nil {
		if err == sql.ErrNoRows {
			insertKc, err := tx.Prepare(`
			INSERT INTO
				keycards
				(
					keycard_uid,
					keycard_name,
					keycard_locked,
					key_uid
				)
			VALUES
				(?, ?, ?, ?);
			`)

			if err != nil {
				return err
			}

			defer insertKc.Close()

			_, err = insertKc.Exec(keyPair.KeycardUID, keyPair.KeycardName, keyPair.KeycardLocked, keyPair.KeyUID)
			if err != nil {
				return err
			}

		} else {
			return err
		}
	}

	insertKcAcc, err = tx.Prepare(`
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

	for i := range keyPair.AccountsAddresses {
		addr := keyPair.AccountsAddresses[i]

		_, err = insertKcAcc.Exec(keyPair.KeycardUID, addr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kp *KeyPairs) getAccountsForKeycard(kcUID string) ([]types.Address, error) {
	var accountAddresses []types.Address

	rows, err := kp.db.Query(`SELECT account_address FROM keycards_accounts WHERE keycard_uid = ?`, kcUID)
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

func (kp *KeyPairs) RemoveMigratedAccountsForKeycard(kcUID string, accountAddresses []types.Address) (err error) {
	dbAccountAddresses, err := kp.getAccountsForKeycard(kcUID)
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
		return kp.DeleteKeycard(kcUID)
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
	delete, err := kp.db.Prepare(query)
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

func (kp *KeyPairs) execUpdateQuery(kcUID string, field string, value interface{}) (err error) {
	sql := fmt.Sprintf(`UPDATE keycards SET %s = ? WHERE keycard_uid = ?`, field) // nolint: gosec

	update, err := kp.db.Prepare(sql)

	if err != nil {
		return err
	}
	defer update.Close()

	_, err = update.Exec(value, kcUID)

	return err
}

func (kp *KeyPairs) KeycardLocked(kcUID string) (err error) {
	return kp.execUpdateQuery(kcUID, "keycard_locked", true)
}

func (kp *KeyPairs) KeycardUnlocked(kcUID string) (err error) {
	return kp.execUpdateQuery(kcUID, "keycard_locked", false)
}

func (kp *KeyPairs) UpdateKeycardUID(oldKcUID string, newKcUID string) (err error) {
	return kp.execUpdateQuery(oldKcUID, "keycard_uid", newKcUID)
}

func (kp *KeyPairs) SetKeycardName(kcUID string, kpName string) (err error) {
	return kp.execUpdateQuery(kcUID, "keycard_name", kpName)
}

func (kp *KeyPairs) DeleteKeycard(kcUID string) (err error) {
	delete, err := kp.db.Prepare(`
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
