package protocol

import (
	"context"
	"database/sql"
)

type ProfileShowcaseEntryType int

const (
	ProfileShowcaseEntryTypeCommunity ProfileShowcaseEntryType = iota
	ProfileShowcaseEntryTypeAccount
	ProfileShowcaseEntryTypeCollectible
	ProfileShowcaseEntryTypeAsset
)

type ProfileShowcaseVisibility int

const (
	ProfileShowcaseVisibilityNoOne ProfileShowcaseVisibility = iota
	ProfileShowcaseVisibilityIDVerifiedContacts
	ProfileShowcaseVisibilityContacts
	ProfileShowcaseVisibilityEveryone
)

const insertOrUpdateProfilePreferencesQuery = "INSERT OR REPLACE INTO profile_showcase_preferences(id, entry_type, visibility, sort_order) VALUES (?, ?, ?, ?)"
const selectAllProfilePreferencesQuery = "SELECT id, entry_type, visibility, sort_order FROM profile_showcase_preferences"
const selectProfilePreferencesByTypeQuery = "SELECT id, entry_type, visibility, sort_order FROM profile_showcase_preferences WHERE entry_type = ?"

type ProfileShowcaseEntry struct {
	ID                 string                    `json:"id"`
	EntryType          ProfileShowcaseEntryType  `json:"entryType"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

func (db sqlitePersistence) InsertOrUpdateProfileShowcasePreference(entry *ProfileShowcaseEntry) error {
	_, err := db.db.Exec(insertOrUpdateProfilePreferencesQuery,
		entry.ID,
		entry.EntryType,
		entry.ShowcaseVisibility,
		entry.Order,
	)

	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) SaveProfileShowcasePreferences(entries []*ProfileShowcaseEntry) error {
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

	for _, entry := range entries {
		_, err = tx.Exec(insertOrUpdateProfilePreferencesQuery,
			entry.ID,
			entry.EntryType,
			entry.ShowcaseVisibility,
			entry.Order,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (db sqlitePersistence) parseProfileShowcasePreferencesRows(rows *sql.Rows) ([]*ProfileShowcaseEntry, error) {
	entries := []*ProfileShowcaseEntry{}

	for rows.Next() {
		entry := &ProfileShowcaseEntry{}

		err := rows.Scan(
			&entry.ID,
			&entry.EntryType,
			&entry.ShowcaseVisibility,
			&entry.Order,
		)

		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

func (db sqlitePersistence) GetAllProfileShowcasePreferences() ([]*ProfileShowcaseEntry, error) {
	rows, err := db.db.Query(selectAllProfilePreferencesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return db.parseProfileShowcasePreferencesRows(rows)
}

func (db sqlitePersistence) GetProfileShowcasePreferencesByType(entryType ProfileShowcaseEntryType) ([]*ProfileShowcaseEntry, error) {
	rows, err := db.db.Query(selectProfilePreferencesByTypeQuery, entryType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return db.parseProfileShowcasePreferencesRows(rows)
}
