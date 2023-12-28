package protocol

import (
	"context"
	"database/sql"
)

type ProfileShowcaseVisibility int

const (
	ProfileShowcaseVisibilityNoOne ProfileShowcaseVisibility = iota
	ProfileShowcaseVisibilityIDVerifiedContacts
	ProfileShowcaseVisibilityContacts
	ProfileShowcaseVisibilityEveryone
)

const upsertProfileShowcaseCommunityPreferenceQuery = "INSERT OR REPLACE INTO profile_showcase_communities_preferences(community_id, visibility, sort_order) VALUES (?, ?, ?)"
const selectProfileShowcaseCommunityPreferenceQuery = "SELECT community_id, visibility, sort_order FROM profile_showcase_communities_preferences"

const upsertProfileShowcaseAccountPreferenceQuery = "INSERT OR REPLACE INTO profile_showcase_accounts_preferences(address, name, color_id, emoji, visibility, sort_order) VALUES (?, ?, ?, ?, ?, ?)"
const selectProfileShowcaseAccountPreferenceQuery = "SELECT address, name, color_id, emoji, visibility, sort_order FROM profile_showcase_accounts_preferences"

const upsertProfileShowcaseCollectiblePreferenceQuery = "INSERT OR REPLACE INTO profile_showcase_collectibles_preferences(contract_address, chain_id, token_id, community_id, account_address, visibility, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)"
const selectProfileShowcaseCollectiblePreferenceQuery = "SELECT contract_address, chain_id, token_id, community_id, account_address, visibility, sort_order FROM profile_showcase_collectibles_preferences"

const upsertProfileShowcaseAssetPreferenceQuery = "INSERT OR REPLACE INTO profile_showcase_assets_preferences(contract_address, community_id, symbol, visibility, sort_order) VALUES (?, ?, ?, ?, ?)"
const selectProfileShowcaseAssetPreferenceQuery = "SELECT contract_address, community_id, symbol, visibility, sort_order FROM profile_showcase_assets_preferences"

const upsertContactProfileShowcaseCommunityQuery = "INSERT OR REPLACE INTO profile_showcase_communities_contacts(contact_id, community_id, sort_order) VALUES (?, ?, ?)"
const selectContactProfileShowcaseCommunityQuery = "SELECT community_id, sort_order FROM profile_showcase_communities_contacts WHERE contact_id = ?"
const removeContactProfileShowcaseCommunityQuery = "DELETE FROM profile_showcase_communities_contacts WHERE contact_id = ?"

const upsertContactProfileShowcaseAccountQuery = "INSERT OR REPLACE INTO profile_showcase_accounts_contacts(contact_id, address, name, color_id, emoji, sort_order) VALUES (?, ?, ?, ?, ?, ?)"
const selectContactProfileShowcaseAccountQuery = "SELECT address, name, color_id, emoji, sort_order FROM profile_showcase_accounts_contacts WHERE contact_id = ?"
const removeContactProfileShowcaseAccountQuery = "DELETE FROM profile_showcase_accounts_contacts WHERE contact_id = ?"

const upsertContactProfileShowcaseCollectibleQuery = "INSERT OR REPLACE INTO profile_showcase_collectibles_contacts(contact_id, contract_address, chain_id, token_id, community_id, account_address, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)"
const selectContactProfileShowcaseCollectibleQuery = "SELECT contract_address, chain_id, token_id, community_id, account_address, sort_order FROM profile_showcase_collectibles_contacts WHERE contact_id = ?"
const removeContactProfileShowcaseCollectibleQuery = "DELETE FROM profile_showcase_collectibles_contacts WHERE contact_id = ?"

const upsertContactProfileShowcaseAssetQuery = "INSERT OR REPLACE INTO profile_showcase_assets_contacts(contact_id, contract_address, community_id, symbol, sort_order) VALUES (?, ?, ?, ?, ?)"
const selectContactProfileShowcaseAssetQuery = "SELECT contract_address, community_id, symbol, sort_order FROM profile_showcase_assets_contacts WHERE contact_id = ?"
const removeContactProfileShowcaseAssetQuery = "DELETE FROM profile_showcase_assets_contacts WHERE contact_id = ?"

type ProfileShowcaseCommunityPreference struct {
	CommunityID        string                    `json:"communityId"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseAccountPreference struct {
	Address            string                    `json:"address"`
	Name               string                    `json:"name"`
	ColorID            string                    `json:"colorId"`
	Emoji              string                    `json:"emoji"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseCollectiblePreference struct {
	ChainID            string                    `json:"chainId"`
	TokenID            string                    `json:"tokenId"`
	ContractAddress    string                    `json:"contractAddress"`
	CommunityID        string                    `json:"communityId"`
	AccountAddress     string                    `json:"accountAddress"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseAssetPreference struct {
	ContractAddress    string                    `json:"contractAddress"`
	CommunityID        string                    `json:"communityId"`
	Symbol             string                    `json:"symbol"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcasePreferences struct {
	Communities  []*ProfileShowcaseCommunityPreference   `json:"communities"`
	Accounts     []*ProfileShowcaseAccountPreference     `json:"accounts"`
	Collectibles []*ProfileShowcaseCollectiblePreference `json:"collectibles"`
	Assets       []*ProfileShowcaseAssetPreference       `json:"assets"`
}

type ProfileShowcaseCommunity struct {
	CommunityID string `json:"communityId"`
	Order       int    `json:"order"`
}

type ProfileShowcaseAccount struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	ColorID string `json:"colorId"`
	Emoji   string `json:"emoji"`
	Order   int    `json:"order"`
}

type ProfileShowcaseCollectible struct {
	ChainID         string `json:"chainId"`
	TokenID         string `json:"tokenId"`
	ContractAddress string `json:"contractAddress"`
	CommunityID     string `json:"communityId"`
	AccountAddress  string `json:"accountAddress"`
	Order           int    `json:"order"`
}

type ProfileShowcaseAsset struct {
	ContractAddress string `json:"contractAddress"`
	CommunityID     string `json:"communityId"`
	Symbol          string `json:"symbol"`
	Order           int    `json:"order"`
}

type ProfileShowcase struct {
	ContactID    string                        `json:"contactId"`
	Communities  []*ProfileShowcaseCommunity   `json:"communities"`
	Accounts     []*ProfileShowcaseAccount     `json:"accounts"`
	Collectibles []*ProfileShowcaseCollectible `json:"collectibles"`
	Assets       []*ProfileShowcaseAsset       `json:"assets"`
}

// Queries for showcase preferences
func (db sqlitePersistence) saveProfileShowcaseCommunityPreference(tx *sql.Tx, community *ProfileShowcaseCommunityPreference) error {
	_, err := tx.Exec(upsertProfileShowcaseCommunityPreferenceQuery,
		community.CommunityID,
		community.ShowcaseVisibility,
		community.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseCommunitiesPreferences(tx *sql.Tx) ([]*ProfileShowcaseCommunityPreference, error) {
	rows, err := tx.Query(selectProfileShowcaseCommunityPreferenceQuery)
	if err != nil {
		return nil, err
	}

	communities := []*ProfileShowcaseCommunityPreference{}

	for rows.Next() {
		community := &ProfileShowcaseCommunityPreference{}

		err := rows.Scan(
			&community.CommunityID,
			&community.ShowcaseVisibility,
			&community.Order,
		)

		if err != nil {
			return nil, err
		}

		communities = append(communities, community)
	}
	return communities, nil
}

func (db sqlitePersistence) saveProfileShowcaseAccountPreference(tx *sql.Tx, account *ProfileShowcaseAccountPreference) error {
	_, err := tx.Exec(upsertProfileShowcaseAccountPreferenceQuery,
		account.Address,
		account.Name,
		account.ColorID,
		account.Emoji,
		account.ShowcaseVisibility,
		account.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseAccountsPreferences(tx *sql.Tx) ([]*ProfileShowcaseAccountPreference, error) {
	rows, err := tx.Query(selectProfileShowcaseAccountPreferenceQuery)
	if err != nil {
		return nil, err
	}

	accounts := []*ProfileShowcaseAccountPreference{}

	for rows.Next() {
		account := &ProfileShowcaseAccountPreference{}

		err := rows.Scan(
			&account.Address,
			&account.Name,
			&account.ColorID,
			&account.Emoji,
			&account.ShowcaseVisibility,
			&account.Order,
		)

		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (db sqlitePersistence) saveProfileShowcaseCollectiblePreference(tx *sql.Tx, collectible *ProfileShowcaseCollectiblePreference) error {
	_, err := tx.Exec(upsertProfileShowcaseCollectiblePreferenceQuery,
		collectible.ContractAddress,
		collectible.ChainID,
		collectible.TokenID,
		collectible.CommunityID,
		collectible.AccountAddress,
		collectible.ShowcaseVisibility,
		collectible.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseCollectiblesPreferences(tx *sql.Tx) ([]*ProfileShowcaseCollectiblePreference, error) {
	rows, err := tx.Query(selectProfileShowcaseCollectiblePreferenceQuery)
	if err != nil {
		return nil, err
	}

	collectibles := []*ProfileShowcaseCollectiblePreference{}

	for rows.Next() {
		collectible := &ProfileShowcaseCollectiblePreference{}

		err := rows.Scan(
			&collectible.ContractAddress,
			&collectible.ChainID,
			&collectible.TokenID,
			&collectible.CommunityID,
			&collectible.AccountAddress,
			&collectible.ShowcaseVisibility,
			&collectible.Order,
		)

		if err != nil {
			return nil, err
		}

		collectibles = append(collectibles, collectible)
	}
	return collectibles, nil
}

func (db sqlitePersistence) saveProfileShowcaseAssetPreference(tx *sql.Tx, asset *ProfileShowcaseAssetPreference) error {
	_, err := tx.Exec(upsertProfileShowcaseAssetPreferenceQuery,
		asset.ContractAddress,
		asset.CommunityID,
		asset.Symbol,
		asset.ShowcaseVisibility,
		asset.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseAssetsPreferences(tx *sql.Tx) ([]*ProfileShowcaseAssetPreference, error) {
	rows, err := tx.Query(selectProfileShowcaseAssetPreferenceQuery)
	if err != nil {
		return nil, err
	}

	assets := []*ProfileShowcaseAssetPreference{}

	for rows.Next() {
		asset := &ProfileShowcaseAssetPreference{}

		err := rows.Scan(
			&asset.ContractAddress,
			&asset.CommunityID,
			&asset.Symbol,
			&asset.ShowcaseVisibility,
			&asset.Order,
		)

		if err != nil {
			return nil, err
		}

		assets = append(assets, asset)
	}
	return assets, nil
}

// Queries for contacts showcase
func (db sqlitePersistence) saveProfileShowcaseCommunityContact(tx *sql.Tx, contactID string, community *ProfileShowcaseCommunity) error {
	_, err := tx.Exec(upsertContactProfileShowcaseCommunityQuery,
		contactID,
		community.CommunityID,
		community.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseCommunitiesContact(tx *sql.Tx, contactID string) ([]*ProfileShowcaseCommunity, error) {
	rows, err := tx.Query(selectContactProfileShowcaseCommunityQuery, contactID)
	if err != nil {
		return nil, err
	}

	communities := []*ProfileShowcaseCommunity{}

	for rows.Next() {
		community := &ProfileShowcaseCommunity{}

		err := rows.Scan(&community.CommunityID, &community.Order)
		if err != nil {
			return nil, err
		}

		communities = append(communities, community)
	}
	return communities, nil
}

func (db sqlitePersistence) clearProfileShowcaseCommunityContact(tx *sql.Tx, contactID string) error {
	_, err := tx.Exec(removeContactProfileShowcaseCommunityQuery, contactID)
	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) saveProfileShowcaseAccountContact(tx *sql.Tx, contactID string, account *ProfileShowcaseAccount) error {
	_, err := tx.Exec(upsertContactProfileShowcaseAccountQuery,
		contactID,
		account.Address,
		account.Name,
		account.ColorID,
		account.Emoji,
		account.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseAccountsContact(tx *sql.Tx, contactID string) ([]*ProfileShowcaseAccount, error) {
	rows, err := tx.Query(selectContactProfileShowcaseAccountQuery, contactID)
	if err != nil {
		return nil, err
	}

	accounts := []*ProfileShowcaseAccount{}

	for rows.Next() {
		account := &ProfileShowcaseAccount{}

		err := rows.Scan(&account.Address, &account.Name, &account.ColorID, &account.Emoji, &account.Order)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (db sqlitePersistence) clearProfileShowcaseAccountsContact(tx *sql.Tx, contactID string) error {
	_, err := tx.Exec(removeContactProfileShowcaseAccountQuery, contactID)
	return err
}

func (db sqlitePersistence) saveProfileShowcaseCollectibleContact(tx *sql.Tx, contactID string, collectible *ProfileShowcaseCollectible) error {
	_, err := tx.Exec(upsertContactProfileShowcaseCollectibleQuery,
		contactID,
		collectible.ContractAddress,
		collectible.ChainID,
		collectible.TokenID,
		collectible.CommunityID,
		collectible.AccountAddress,
		collectible.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseCollectiblesContact(tx *sql.Tx, contactID string) ([]*ProfileShowcaseCollectible, error) {
	rows, err := tx.Query(selectContactProfileShowcaseCollectibleQuery, contactID)
	if err != nil {
		return nil, err
	}

	collectibles := []*ProfileShowcaseCollectible{}

	for rows.Next() {
		collectible := &ProfileShowcaseCollectible{}

		err := rows.Scan(
			&collectible.ContractAddress,
			&collectible.ChainID,
			&collectible.TokenID,
			&collectible.CommunityID,
			&collectible.AccountAddress,
			&collectible.Order)
		if err != nil {
			return nil, err
		}

		collectibles = append(collectibles, collectible)
	}
	return collectibles, nil
}

func (db sqlitePersistence) clearProfileShowcaseCollectiblesContact(tx *sql.Tx, contactID string) error {
	_, err := tx.Exec(removeContactProfileShowcaseCollectibleQuery, contactID)
	return err
}

func (db sqlitePersistence) saveProfileShowcaseAssetContact(tx *sql.Tx, contactID string, asset *ProfileShowcaseAsset) error {
	_, err := tx.Exec(upsertContactProfileShowcaseAssetQuery,
		contactID,
		asset.ContractAddress,
		asset.CommunityID,
		asset.Symbol,
		asset.Order,
	)

	return err
}

func (db sqlitePersistence) getProfileShowcaseAssetsContact(tx *sql.Tx, contactID string) ([]*ProfileShowcaseAsset, error) {
	rows, err := tx.Query(selectContactProfileShowcaseAssetQuery, contactID)
	if err != nil {
		return nil, err
	}

	assets := []*ProfileShowcaseAsset{}

	for rows.Next() {
		asset := &ProfileShowcaseAsset{}

		err := rows.Scan(
			&asset.ContractAddress,
			&asset.CommunityID,
			&asset.Symbol,
			&asset.Order)
		if err != nil {
			return nil, err
		}

		assets = append(assets, asset)
	}
	return assets, nil
}

func (db sqlitePersistence) clearProfileShowcaseAssetsContact(tx *sql.Tx, contactID string) error {
	_, err := tx.Exec(removeContactProfileShowcaseAssetQuery, contactID)
	return err
}

// public functions
func (db sqlitePersistence) SaveProfileShowcasePreferences(preferences *ProfileShowcasePreferences) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for _, community := range preferences.Communities {
		err = db.saveProfileShowcaseCommunityPreference(tx, community)
		if err != nil {
			return err
		}
	}

	for _, account := range preferences.Accounts {
		err = db.saveProfileShowcaseAccountPreference(tx, account)
		if err != nil {
			return err
		}
	}

	for _, collectible := range preferences.Collectibles {
		err = db.saveProfileShowcaseCollectiblePreference(tx, collectible)
		if err != nil {
			return err
		}
	}

	for _, asset := range preferences.Assets {
		err = db.saveProfileShowcaseAssetPreference(tx, asset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db sqlitePersistence) GetProfileShowcasePreferences() (*ProfileShowcasePreferences, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	communities, err := db.getProfileShowcaseCommunitiesPreferences(tx)
	if err != nil {
		return nil, err
	}

	accounts, err := db.getProfileShowcaseAccountsPreferences(tx)
	if err != nil {
		return nil, err
	}

	collectibles, err := db.getProfileShowcaseCollectiblesPreferences(tx)
	if err != nil {
		return nil, err
	}

	assets, err := db.getProfileShowcaseAssetsPreferences(tx)
	if err != nil {
		return nil, err
	}

	return &ProfileShowcasePreferences{
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}

func (db sqlitePersistence) SaveProfileShowcaseForContact(showcase *ProfileShowcase) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for _, community := range showcase.Communities {
		err = db.saveProfileShowcaseCommunityContact(tx, showcase.ContactID, community)
		if err != nil {
			return err
		}
	}

	for _, account := range showcase.Accounts {
		err = db.saveProfileShowcaseAccountContact(tx, showcase.ContactID, account)
		if err != nil {
			return err
		}
	}

	for _, collectible := range showcase.Collectibles {
		err = db.saveProfileShowcaseCollectibleContact(tx, showcase.ContactID, collectible)
		if err != nil {
			return err
		}
	}

	for _, asset := range showcase.Assets {
		err = db.saveProfileShowcaseAssetContact(tx, showcase.ContactID, asset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db sqlitePersistence) GetProfileShowcaseForContact(contactID string) (*ProfileShowcase, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	communities, err := db.getProfileShowcaseCommunitiesContact(tx, contactID)
	if err != nil {
		return nil, err
	}

	accounts, err := db.getProfileShowcaseAccountsContact(tx, contactID)
	if err != nil {
		return nil, err
	}

	collectibles, err := db.getProfileShowcaseCollectiblesContact(tx, contactID)
	if err != nil {
		return nil, err
	}

	assets, err := db.getProfileShowcaseAssetsContact(tx, contactID)
	if err != nil {
		return nil, err
	}

	return &ProfileShowcase{
		ContactID:    contactID,
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}

func (db sqlitePersistence) ClearProfileShowcaseForContact(contactID string) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	err = db.clearProfileShowcaseCommunityContact(tx, contactID)
	if err != nil {
		return err
	}

	err = db.clearProfileShowcaseAccountsContact(tx, contactID)
	if err != nil {
		return err
	}

	err = db.clearProfileShowcaseCollectiblesContact(tx, contactID)
	if err != nil {
		return err
	}

	err = db.clearProfileShowcaseAssetsContact(tx, contactID)
	if err != nil {
		return err
	}

	return nil
}
