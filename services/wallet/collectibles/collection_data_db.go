package collectibles

import (
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/alchemy"
	"github.com/status-im/status-go/sqlite"
)

type CollectionDataDB struct {
	db *sql.DB
}

func NewCollectionDataDB(sqlDb *sql.DB) *CollectionDataDB {
	return &CollectionDataDB{
		db: sqlDb,
	}
}

const collectionDataColumns = "chain_id, contract_address, provider, name, slug, image_url, image_payload, community_id"
const collectionTraitsColumns = "chain_id, contract_address, trait_type, min, max"
const selectCollectionTraitsColumns = "trait_type, min, max"
const collectionSocialsColumns = "chain_id, contract_address, website, twitter_handle"
const selectCollectionSocialsColumns = "website, twitter_handle"

func rowsToCollectionTraits(rows *sql.Rows) (map[string]thirdparty.CollectionTrait, error) {
	traits := make(map[string]thirdparty.CollectionTrait)
	for rows.Next() {
		var traitType string
		var trait thirdparty.CollectionTrait
		err := rows.Scan(
			&traitType,
			&trait.Min,
			&trait.Max,
		)
		if err != nil {
			return nil, err
		}
		traits[traitType] = trait
	}
	return traits, nil
}

func getCollectionTraits(creator sqlite.StatementCreator, id thirdparty.ContractID) (map[string]thirdparty.CollectionTrait, error) {
	// Get traits list
	selectTraits, err := creator.Prepare(fmt.Sprintf(`SELECT %s
		FROM collection_traits_cache
		WHERE chain_id = ? AND contract_address = ?`, selectCollectionTraitsColumns))
	if err != nil {
		return nil, err
	}

	rows, err := selectTraits.Query(
		id.ChainID,
		id.Address,
	)
	if err != nil {
		return nil, err
	}

	return rowsToCollectionTraits(rows)
}

func upsertCollectionTraits(creator sqlite.StatementCreator, id thirdparty.ContractID, traits map[string]thirdparty.CollectionTrait) error {
	// Rremove old traits list
	deleteTraits, err := creator.Prepare(`DELETE FROM collection_traits_cache WHERE chain_id = ? AND contract_address = ?`)
	if err != nil {
		return err
	}

	_, err = deleteTraits.Exec(
		id.ChainID,
		id.Address,
	)
	if err != nil {
		return err
	}

	// Insert new traits list
	insertTrait, err := creator.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO collection_traits_cache (%s)
		VALUES (?, ?, ?, ?, ?)`, collectionTraitsColumns))
	if err != nil {
		return err
	}

	for traitType, trait := range traits {
		_, err = insertTrait.Exec(
			id.ChainID,
			id.Address,
			traitType,
			trait.Min,
			trait.Max,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func idExists(creator sqlite.StatementCreator, contractID thirdparty.ContractID) bool {
	exists, err := creator.Prepare(`SELECT EXISTS (
                        SELECT 1 FROM collection_socials_cache
                        WHERE chain_id=? AND contract_address=?
                )`)
	if err != nil {
		return false
	}

	row := exists.QueryRow(
		contractID.ChainID,
		contractID.Address,
	)
	var found bool
	err = row.Scan(&found)
	if err != nil {
		return false
	}

	return found
}

func setCollectionsData(creator sqlite.StatementCreator, collections []thirdparty.CollectionData, allowUpdate bool) error {
	insertCollection, err := creator.Prepare(fmt.Sprintf(`%s INTO collection_data_cache (%s) 
																				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, insertStatement(allowUpdate), collectionDataColumns))
	if err != nil {
		return err
	}

	for _, c := range collections {
		_, err = insertCollection.Exec(
			c.ID.ChainID,
			c.ID.Address,
			c.Provider,
			c.Name,
			c.Slug,
			c.ImageURL,
			c.ImagePayload,
			c.CommunityID,
		)
		if err != nil {
			return err
		}

		err = upsertContractType(creator, c.ID, c.ContractType)
		if err != nil {
			return err
		}

		if allowUpdate {
			err = upsertCollectionTraits(creator, c.ID, c.Traits)
			if err != nil {
				return err
			}

			if c.Provider == alchemy.AlchemyID && !idExists(creator, c.ID) {
				err = upsertCollectionSocials(creator, c.ID, c.Socials)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (o *CollectionDataDB) SetData(collections []thirdparty.CollectionData, allowUpdate bool) (err error) {
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

	// Insert new collections data
	err = setCollectionsData(tx, collections, allowUpdate)
	if err != nil {
		return err
	}

	return
}

func scanCollectionsDataRow(row *sql.Row) (*thirdparty.CollectionData, error) {
	c := thirdparty.CollectionData{
		Traits: make(map[string]thirdparty.CollectionTrait),
	}
	err := row.Scan(
		&c.ID.ChainID,
		&c.ID.Address,
		&c.Provider,
		&c.Name,
		&c.Slug,
		&c.ImageURL,
		&c.ImagePayload,
		&c.CommunityID,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (o *CollectionDataDB) GetIDsNotInDB(ids []thirdparty.ContractID) ([]thirdparty.ContractID, error) {
	ret := make([]thirdparty.ContractID, 0, len(ids))
	idMap := make(map[string]thirdparty.ContractID, len(ids))

	// Ensure we don't have duplicates
	for _, id := range ids {
		idMap[id.HashKey()] = id
	}

	exists, err := o.db.Prepare(`SELECT EXISTS (
			SELECT 1 FROM collection_data_cache
			WHERE chain_id=? AND contract_address=?
		)`)
	if err != nil {
		return nil, err
	}

	for _, id := range idMap {
		row := exists.QueryRow(
			id.ChainID,
			id.Address,
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

func (o *CollectionDataDB) GetData(ids []thirdparty.ContractID) (map[string]thirdparty.CollectionData, error) {
	ret := make(map[string]thirdparty.CollectionData)

	getData, err := o.db.Prepare(fmt.Sprintf(`SELECT %s
		FROM collection_data_cache
		WHERE chain_id=? AND contract_address=?`, collectionDataColumns))
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		row := getData.QueryRow(
			id.ChainID,
			id.Address,
		)
		c, err := scanCollectionsDataRow(row)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return nil, err
		} else {
			// Get traits from different table
			c.Traits, err = getCollectionTraits(o.db, c.ID)
			if err != nil {
				return nil, err
			}

			// Get contract type from different table
			c.ContractType, err = readContractType(o.db, c.ID)
			if err != nil {
				return nil, err
			}

			// Get socials from different table
			c.Socials, err = getCollectionSocials(o.db, c.ID)
			if err != nil {
				return nil, err
			}

			ret[c.ID.HashKey()] = *c
		}
	}
	return ret, nil
}

func (o *CollectionDataDB) GetSocialsValidForID(id thirdparty.CollectibleUniqueID) bool {
	return idExists(o.db, id.ContractID)
}

func (o *CollectionDataDB) SetCollectionSocialsData(id thirdparty.ContractID, collectionSocials thirdparty.CollectionSocials) (err error) {
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

	// Insert new collections socials
	if idExists(tx, id) {
		err = upsertCollectionSocials(tx, id, collectionSocials)
		if err != nil {
			return err
		}
	}

	return
}

func rowsToCollectionSocials(rows *sql.Rows) (thirdparty.CollectionSocials, error) {
	socials := thirdparty.CollectionSocials{}
	for rows.Next() {
		var website string
		var twitterHandle string
		err := rows.Scan(
			&website,
			&twitterHandle,
		)
		if err != nil {
			return socials, err
		}
		if len(website) > 0 {
			socials.Website = website
		}
		if len(twitterHandle) > 0 {
			socials.TwitterHandle = twitterHandle
		}
	}
	return socials, nil
}

func getCollectionSocials(creator sqlite.StatementCreator, id thirdparty.ContractID) (thirdparty.CollectionSocials, error) {
	// Get socials
	selectSocials, err := creator.Prepare(fmt.Sprintf(`SELECT %s
                FROM collection_socials_cache
                WHERE chain_id = ? AND contract_address = ?`, selectCollectionSocialsColumns))
	if err != nil {
		return thirdparty.CollectionSocials{}, err
	}

	rows, err := selectSocials.Query(
		id.ChainID,
		id.Address,
	)
	if err != nil {
		return thirdparty.CollectionSocials{}, err
	}

	return rowsToCollectionSocials(rows)
}

func upsertCollectionSocials(creator sqlite.StatementCreator, id thirdparty.ContractID, socials thirdparty.CollectionSocials) error {
	// Insert socials
	insertSocial, err := creator.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO collection_socials_cache (%s)
        VALUES (?, ?, ?, ?)`, collectionSocialsColumns))
	if err != nil {
		return err
	}

	_, err = insertSocial.Exec(
		id.ChainID,
		id.Address,
		socials.Website,
		socials.TwitterHandle,
	)
	if err != nil {
		return err
	}

	return nil
}
