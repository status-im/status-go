package protocol

import "database/sql"

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
	ID         string                    `json:"id"`
	Type       ProfileShowcaseEntryType  `json:"type"`
	Visibility ProfileShowcaseVisibility `json:"visiblity"`
	Order      int                       `json:"order"`
}

func (db sqlitePersistence) InsertOrUpdateProfileShowcasePreference(entry *ProfileShowcaseEntry) error {
	_, err := db.db.Exec(insertOrUpdateProfilePreferencesQuery,
		entry.ID,
		entry.Type,
		entry.Visibility,
		entry.Order,
	)

	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) parseProfileShowcasePreferencesRows(rows *sql.Rows) ([]*ProfileShowcaseEntry, error) {
	entries := []*ProfileShowcaseEntry{}

	for rows.Next() {
		entry := &ProfileShowcaseEntry{}

		err := rows.Scan(
			&entry.ID,
			&entry.Type,
			&entry.Visibility,
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
