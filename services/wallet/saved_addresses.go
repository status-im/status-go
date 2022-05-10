package wallet

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
)

type SavedAddress struct {
	Address common.Address `json:"address"`
	// TODO: Add Emoji and Networks
	// Emoji    string         `json:"emoji"`
	Name    string `json:"name"`
	ChainID uint64 `json:"chainId"`
}

type SavedAddressesManager struct {
	db *sql.DB
}

func (sam *SavedAddressesManager) GetSavedAddresses(chainID uint64) ([]SavedAddress, error) {
	rows, err := sam.db.Query("SELECT address, name, network_id FROM saved_addresses WHERE network_id = ?", chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []SavedAddress
	for rows.Next() {
		sa := SavedAddress{}
		err := rows.Scan(&sa.Address, &sa.Name, &sa.ChainID)
		if err != nil {
			return nil, err
		}

		rst = append(rst, sa)
	}

	return rst, nil
}

func (sam *SavedAddressesManager) AddSavedAddress(sa SavedAddress) error {
	insert, err := sam.db.Prepare("INSERT OR REPLACE INTO saved_addresses (network_id, address, name) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(sa.ChainID, sa.Address, sa.Name)
	return err
}

func (sam *SavedAddressesManager) DeleteSavedAddress(chainID uint64, address common.Address) error {
	_, err := sam.db.Exec(`DELETE FROM saved_addresses WHERE address = ? AND network_id = ?`, address, chainID)
	return err
}
