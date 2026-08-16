package collectibles

//go:generate go tool mockgen -package=mock_collectibles -source=collectible_data_db.go -destination=mock/collectible_data_db.go

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type CollectibleDataStorage interface {
	SetData(collectibles []thirdparty.CollectibleData, allowUpdate bool) error
	GetIDsNeedingFetch(ids []thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleUniqueID, error)
	GetData(ids []thirdparty.CollectibleUniqueID) (map[string]thirdparty.CollectibleData, error)
	SetCommunityInfo(id thirdparty.CollectibleUniqueID, communityInfo thirdparty.CollectibleCommunityInfo) error
	GetCommunityInfo(id thirdparty.CollectibleUniqueID) (*thirdparty.CollectibleCommunityInfo, error)
}

type CollectibleDataDB struct {
	db *sql.DB
}

func NewCollectibleDataDB(sqlDb *sql.DB) *CollectibleDataDB {
	return &CollectibleDataDB{
		db: sqlDb,
	}
}

// collectibleMetadataVersion is the version of the mapping from a provider's
// response to CollectibleData. Bump it whenever the same response would now map
// to a different row, so that rows written by an older mapping are refetched
// instead of being served from cache forever. Changes that leave the mapping
// alone - a new consumer of an existing field, media server or UI work - do not
// qualify.
//
// 1: collectibles carry the provider's own thumbnail, and an animation URL is
// only reported for media that can actually move.
// 2: the animation media type comes from the provider rather than from a HEAD
// request against the animation URL, and the two can disagree.
// 3: rows carry the size the provider reported for each media URL. Older rows
// have no size, which reads as "the provider did not say" and lets the asset
// past the size cap, so they have to be refetched rather than trusted.
const collectibleMetadataVersion = 3

const collectibleDataColumns = "chain_id, contract_address, token_id, provider, name, description, permalink, image_url, thumbnail_url, image_size, thumbnail_size, animation_size, image_payload, animation_url, animation_media_type, background_color, token_uri, community_id, soulbound"

// metadata_version is written on every insert and read only when deciding what
// needs refetching. It describes the cache rather than the collectible, so it
// stays out of collectibleDataColumns and never reaches CollectibleData.
const collectibleDataInsertColumns = collectibleDataColumns + ", metadata_version"
const collectibleCommunityDataColumns = "community_privileges_level"
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
	defer selectTraits.Close()

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
	defer deleteTraits.Close()

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
	defer insertTrait.Close()

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

func setCollectiblesData(creator sqlite.StatementCreator, collectibles []thirdparty.CollectibleData, allowUpdate bool) error {
	insertCollectible, err := creator.Prepare(fmt.Sprintf(`%s INTO collectible_data_cache (%s)
																				VALUES (%s)`, insertStatement(allowUpdate), collectibleDataInsertColumns, valuePlaceholders(collectibleDataInsertColumns)))
	if err != nil {
		return err
	}
	defer insertCollectible.Close()

	// INSERT OR IGNORE leaves an already cached row untouched, so relying on the
	// insert alone would leave a row written by an older mapping at its old
	// version and refetch it on every read. Stamp the version separately: it
	// records that the current mapping has been applied to this ID as far as it
	// can be, which holds whether or not the row itself was replaced.
	markVersion, err := creator.Prepare(`UPDATE collectible_data_cache SET metadata_version = ?
																				WHERE chain_id = ? AND contract_address = ? AND token_id = ?`)
	if err != nil {
		return err
	}
	defer markVersion.Close()

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
			c.ThumbnailURL,
			c.ImageSize,
			c.ThumbnailSize,
			c.AnimationSize,
			c.ImagePayload,
			c.AnimationURL,
			c.AnimationMediaType,
			c.BackgroundColor,
			c.TokenURI,
			c.CommunityID,
			c.Soulbound,
			collectibleMetadataVersion,
		)
		if err != nil {
			return err
		}

		_, err = markVersion.Exec(
			collectibleMetadataVersion,
			c.ID.ContractID.ChainID,
			c.ID.ContractID.Address,
			(*bigint.SQLBigIntBytes)(c.ID.TokenID.Int),
		)
		if err != nil {
			return err
		}

		err = upsertContractType(creator, c.ID.ContractID, c.ContractType)
		if err != nil {
			return err
		}

		if allowUpdate {
			err = upsertCollectibleTraits(creator, c.ID, c.Traits)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *CollectibleDataDB) SetData(collectibles []thirdparty.CollectibleData, allowUpdate bool) (err error) {
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
	err = setCollectiblesData(tx, collectibles, allowUpdate)
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
		&c.ThumbnailURL,
		&c.ImageSize,
		&c.ThumbnailSize,
		&c.AnimationSize,
		&c.ImagePayload,
		&c.AnimationURL,
		&c.AnimationMediaType,
		&c.BackgroundColor,
		&c.TokenURI,
		&c.CommunityID,
		&c.Soulbound,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetIDsNeedingFetch returns the IDs that are not cached at all, plus those cached
// by an older version of the metadata mapping. Presence alone is not enough: a
// mapping change adds fields to rows that already exist, and those rows would
// otherwise keep serving what the previous mapping produced for the rest of the
// account's life.
//
// Staleness is keyed on the mapping version rather than on a field being empty,
// because empty is a legitimate answer - providers return no preview for plenty
// of assets - and refetching on empty would retry those forever.
func (o *CollectibleDataDB) GetIDsNeedingFetch(ids []thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleUniqueID, error) {
	ret := make([]thirdparty.CollectibleUniqueID, 0, len(ids))
	idMap := make(map[string]thirdparty.CollectibleUniqueID, len(ids))

	// Ensure we don't have duplicates
	for _, id := range ids {
		idMap[id.HashKey()] = id
	}

	isUpToDate, err := o.db.Prepare(`SELECT EXISTS (
			SELECT 1 FROM collectible_data_cache
			WHERE chain_id=? AND contract_address=? AND token_id=? AND metadata_version>=?
		)`)
	if err != nil {
		return nil, err
	}
	defer isUpToDate.Close()

	for _, id := range idMap {
		row := isUpToDate.QueryRow(
			id.ContractID.ChainID,
			id.ContractID.Address,
			(*bigint.SQLBigIntBytes)(id.TokenID.Int),
			collectibleMetadataVersion,
		)
		var upToDate bool
		err = row.Scan(&upToDate)
		if err != nil {
			return nil, err
		}
		if !upToDate {
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
	defer getData.Close()

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

			// Get contract type from different table
			c.ContractType, err = readContractType(o.db, c.ID.ContractID)
			if err != nil {
				return nil, err
			}

			ret[c.ID.HashKey()] = *c
		}
	}
	return ret, nil
}

func (o *CollectibleDataDB) SetCommunityInfo(id thirdparty.CollectibleUniqueID, communityInfo thirdparty.CollectibleCommunityInfo) (err error) {
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

	update, err := tx.Prepare(`UPDATE collectible_data_cache 
		SET community_privileges_level=?
		WHERE chain_id=? AND contract_address=? AND token_id=?`)
	if err != nil {
		return err
	}
	defer update.Close()

	_, err = update.Exec(
		communityInfo.PrivilegesLevel,
		id.ContractID.ChainID,
		id.ContractID.Address,
		(*bigint.SQLBigIntBytes)(id.TokenID.Int),
	)

	return err
}

func (o *CollectibleDataDB) GetCommunityInfo(id thirdparty.CollectibleUniqueID) (*thirdparty.CollectibleCommunityInfo, error) {
	ret := thirdparty.CollectibleCommunityInfo{
		PrivilegesLevel: token.CommunityLevel,
	}

	getData, err := o.db.Prepare(fmt.Sprintf(`SELECT %s
		FROM collectible_data_cache
		WHERE chain_id=? AND contract_address=? AND token_id=?`, collectibleCommunityDataColumns))
	if err != nil {
		return nil, err
	}
	defer getData.Close()

	row := getData.QueryRow(
		id.ContractID.ChainID,
		id.ContractID.Address,
		(*bigint.SQLBigIntBytes)(id.TokenID.Int),
	)

	var dbPrivilegesLevel sql.NullByte

	err = row.Scan(
		&dbPrivilegesLevel,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if dbPrivilegesLevel.Valid {
		ret.PrivilegesLevel = token.PrivilegesLevel(dbPrivilegesLevel.Byte)
	}

	return &ret, nil
}
