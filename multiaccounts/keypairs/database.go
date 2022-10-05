package keypairs

import (
	"database/sql"
	"fmt"

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

func (kp *KeyPairs) processResult(rows *sql.Rows) ([]*KeyPair, error) {
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
			if keyPairs[i].KeyUID == keyPair.KeyUID {
				foundAtIndex = i
				break
			}
		}
		if foundAtIndex == -1 {
			keyPair.AccountsAddresses = append(keyPair.AccountsAddresses, addr)
			keyPairs = append(keyPairs, keyPair)
		} else {
			keyPairs[foundAtIndex].AccountsAddresses = append(keyPairs[foundAtIndex].AccountsAddresses, addr)
		}
	}

	return keyPairs, nil
}

func (kp *KeyPairs) GetAllMigratedKeyPairs() ([]*KeyPair, error) {
	rows, err := kp.db.Query(`
		SELECT 
			keycard_uid, 
			keycard_name, 
			keycard_locked, 
			account_address, 
			key_uid
		FROM 
			keypairs
		ORDER BY 
			key_uid
	`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return kp.processResult(rows)
}

func (kp *KeyPairs) GetMigratedKeyPairByKeyUID(keyUID string) ([]*KeyPair, error) {
	rows, err := kp.db.Query(`
		SELECT 
			keycard_uid, 
			keycard_name, 
			keycard_locked, 
			account_address, 
			key_uid
		FROM 
			keypairs
		WHERE
			key_uid = ?
		ORDER BY 
			keycard_uid
	`, keyUID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return kp.processResult(rows)
}

func (kp *KeyPairs) AddMigratedKeyPair(kcUID string, kpName string, KeyUID string, accountAddresses []types.Address) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
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

	insert, err = tx.Prepare(`
		INSERT INTO 
			keypairs 
			(
				keycard_uid, 
				keycard_name, 
				keycard_locked, 
				account_address, 
				key_uid
			) 
		VALUES
			(?, ?, ?, ?, ?);
	`)
	if err != nil {
		return err
	}
	defer insert.Close()

	for i := range accountAddresses {
		addr := accountAddresses[i]

		_, err = insert.Exec(kcUID, kpName, false, addr, KeyUID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kp *KeyPairs) SetKeycardName(kcUID string, kpName string) (err error) {
	update, err := kp.db.Prepare(`
		UPDATE 
			keypairs 
		SET 
			keycard_name = ?
		WHERE 
			keycard_uid = ?
	`)
	if err != nil {
		return err
	}
	defer update.Close()

	_, err = update.Exec(kpName, kcUID)

	return err
}

func (kp *KeyPairs) execUpdateQuery(kcUID string, field string, value interface{}) (err error) {
	var sql string
	sql = fmt.Sprintf(`UPDATE keypairs SET %s = ? WHERE keycard_uid = ?`, field)

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

func (kp *KeyPairs) DeleteKeycard(kcUID string) (err error) {
	delete, err := kp.db.Prepare(`
		DELETE 
		FROM 
			keypairs 
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
