package collectibles

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/sqlite"
)

type CollectibleDataDB struct {
	db *sql.DB
}

func NewCollectibleDataDB(sqlDb *sql.DB) *CollectibleDataDB {
	return &CollectibleDataDB{
		db: sqlDb,
	}
}

const collectibleDataColumns = "chain_id, contract_address, token_id, provider, name, description, permalink, image_url, animation_url, animation_media_type, background_color, token_uri"
const collectibleTraitsColumns = "chain_id, contract_address, token_id, trait_type, trait_value, display_type, max_value"
const selectCollectibleTraitsColumns = "trait_type, trait_value, display_type, max_value"

func rowsToCollectibleTraits(rows *sql.Rows) ([]thirdparty.CollectibleTrait, error) {
	var traits []thirdparty.CollectibleTrait = make([]thirdparty.CollectibleTrait, 0)
	for rows.Next() {
		var trait thirdparty.CollectibleTrait
		err := rows.Scan(
			&trait.TraitType,
			&trait.Value,
			&trait.DisplayType,
			&trait.MaxValue,
		)
		if err != nil {
			return nil, err
		}
		traits = append(traits, trait)
	}
	return traits, nil
}

func getCollectibleTraits(creator sqlite.StatementCreator, id thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleTrait, error) {
	// Get traits list
	selectTraits, err := creator.Prepare(fmt.Sprintf(`SELECT %s
		FROM collectible_traits_cache
		WHERE chain_id = ? AND contract_address = ? AND token_id = ?`, selectCollectibleTraitsColumns))
	if err != nil {
		return nil, err
	}

	rows, err := selectTraits.Query(
		id.ContractID.ChainID,
		id.ContractID.Address,
		(*bigint.SQLBigIntBytes)(id.TokenID.Int),
	)
	if err != nil {
		return nil, err
	}

	return rowsToCollectibleTraits(rows)
}

func upsertCollectibleTraits(creator sqlite.StatementCreator, id thirdparty.CollectibleUniqueID, traits []thirdparty.CollectibleTrait) error {
	// Remove old traits list
	deleteTraits, err := creator.Prepare(`DELETE FROM collectible_traits_cache WHERE chain_id = ? AND contract_address = ? AND token_id = ?`)
	if err != nil {
		return err
	}

	_, err = deleteTraits.Exec(
		id.ContractID.ChainID,
		id.ContractID.Address,
		(*bigint.SQLBigIntBytes)(id.TokenID.Int),
	)
	if err != nil {
		return err
	}

	// Insert new traits list
	insertTrait, err := creator.Prepare(fmt.Sprintf(`INSERT INTO collectible_traits_cache (%s)
																				VALUES (?, ?, ?, ?, ?, ?, ?)`, collectibleTraitsColumns))
	if err != nil {
		return err
	}

	for _, t := range traits {
		_, err = insertTrait.Exec(
			id.ContractID.ChainID,
			id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
			t.TraitType,
			t.Value,
			t.DisplayType,
			t.MaxValue,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertCollectiblesData(creator sqlite.StatementCreator, collectibles []thirdparty.CollectibleData) error {
	insertCollectible, err := creator.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO collectible_data_cache (%s) 
																				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, collectibleDataColumns))
	if err != nil {
		return err
	}

	for _, c := range collectibles {
		_, err = insertCollectible.Exec(
			c.ID.ContractID.ChainID,
			c.ID.ContractID.Address,
			(*bigint.SQLBigIntBytes)(c.ID.TokenID.Int),
			c.Provider,
			c.Name,
			c.Description,
			c.Permalink,
			c.ImageURL,
			c.AnimationURL,
			c.AnimationMediaType,
			c.BackgroundColor,
			c.TokenURI,
		)
		if err != nil {
			return err
		}

		err = upsertCollectibleTraits(creator, c.ID, c.Traits)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *CollectibleDataDB) SetData(collectibles []thirdparty.CollectibleData) (err error) {
	tx, err := o.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	// Insert new collectibles data
	err = upsertCollectiblesData(tx, collectibles)
	if err != nil {
		return err
	}

	return
}

func scanCollectiblesDataRow(row *sql.Row) (*thirdparty.CollectibleData, error) {
	c := thirdparty.CollectibleData{
		ID: thirdparty.CollectibleUniqueID{
			TokenID: &bigint.BigInt{Int: big.NewInt(0)},
		},
		Traits: make([]thirdparty.CollectibleTrait, 0),
	}
	err := row.Scan(
		&c.ID.ContractID.ChainID,
		&c.ID.ContractID.Address,
		(*bigint.SQLBigIntBytes)(c.ID.TokenID.Int),
		&c.Provider,
		&c.Name,
		&c.Description,
		&c.Permalink,
		&c.ImageURL,
		&c.AnimationURL,
		&c.AnimationMediaType,
		&c.BackgroundColor,
		&c.TokenURI,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (o *CollectibleDataDB) GetIDsNotInDB(ids []thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleUniqueID, error) {
	ret := make([]thirdparty.CollectibleUniqueID, 0, len(ids))

	exists, err := o.db.Prepare(`SELECT EXISTS (
			SELECT 1 FROM collectible_data_cache
			WHERE chain_id=? AND contract_address=? AND token_id=?
		)`)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		row := exists.QueryRow(
			id.ContractID.ChainID,
			id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
		)
		var exists bool
		err = row.Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			ret = append(ret, id)
		}
	}

	return ret, nil
}

func (o *CollectibleDataDB) GetData(ids []thirdparty.CollectibleUniqueID) (map[string]thirdparty.CollectibleData, error) {
	ret := make(map[string]thirdparty.CollectibleData)

	getData, err := o.db.Prepare(fmt.Sprintf(`SELECT %s
		FROM collectible_data_cache
		WHERE chain_id=? AND contract_address=? AND token_id=?`, collectibleDataColumns))
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		row := getData.QueryRow(
			id.ContractID.ChainID,
			id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
		)
		c, err := scanCollectiblesDataRow(row)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return nil, err
		} else {
			// Get traits from different table
			c.Traits, err = getCollectibleTraits(o.db, c.ID)
			if err != nil {
				return nil, err
			}
			ret[c.ID.HashKey()] = *c
		}
	}
	return ret, nil
}
