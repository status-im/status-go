package protocol

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/protocol/identity"
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

const removeProfileShowcaseContactQuery = "DELETE FROM profile_showcase_contacts WHERE contact_id = ?"
const insertOrUpdateProfileShowcaseContactQuery = "INSERT OR REPLACE INTO profile_showcase_contacts(contact_id, entry_id, entry_type, entry_order) VALUES (?, ?, ?, ?)"
const selectProfileShowcaseByContactQuery = "SELECT entry_id, entry_order, entry_type FROM profile_showcase_contacts WHERE contact_id = ?"

type ProfileShowcaseEntry struct {
	ID                 string                    `json:"id"`
	EntryType          ProfileShowcaseEntryType  `json:"entryType"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

func (db sqlitePersistence) SaveProfileShowcasePreference(entry *ProfileShowcaseEntry) error {
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

func (db sqlitePersistence) GetProfileShowcaseForContact(contactID string) (*identity.ProfileShowcase, error) {
	rows, err := db.db.Query(selectProfileShowcaseByContactQuery, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	showcase := &identity.ProfileShowcase{}

	for rows.Next() {
		var entryType ProfileShowcaseEntryType
		entry := &identity.VisibleProfileShowcaseEntry{}

		err := rows.Scan(
			&entry.EntryID,
			&entry.Order,
			&entryType,
		)
		if err != nil {
			return nil, err
		}

		switch entryType {
		case ProfileShowcaseEntryTypeCommunity:
			showcase.Communities = append(showcase.Communities, entry)
		case ProfileShowcaseEntryTypeAccount:
			showcase.Accounts = append(showcase.Accounts, entry)
		case ProfileShowcaseEntryTypeCollectible:
			showcase.Collectibles = append(showcase.Collectibles, entry)
		case ProfileShowcaseEntryTypeAsset:
			showcase.Assets = append(showcase.Assets, entry)
		}
	}

	return showcase, nil
}

func (db sqlitePersistence) ClearProfileShowcaseForContact(contactID string) error {
	_, err := db.db.Exec(removeProfileShowcaseContactQuery, contactID)
	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) saveProfileShowcaseContactEntries(tx *sql.Tx, contactID string, entryType ProfileShowcaseEntryType, entries []*identity.VisibleProfileShowcaseEntry) error {
	for _, entry := range entries {
		_, err := tx.Exec(insertOrUpdateProfileShowcaseContactQuery,
			contactID,
			entry.EntryID,
			entryType,
			entry.Order,
		)

		if err != nil {
			return err
		}
	}
	return nil
}

func (db sqlitePersistence) SaveProfileShowcaseForContact(contactID string, showcase *identity.ProfileShowcase) error {
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

	// Remove old entries first
	_, err = tx.Exec(removeProfileShowcaseContactQuery, contactID)
	if err != nil {
		return err
	}

	// Save all profile showcase entries
	err = db.saveProfileShowcaseContactEntries(tx, contactID, ProfileShowcaseEntryTypeCommunity, showcase.Communities)
	if err != nil {
		return err
	}

	err = db.saveProfileShowcaseContactEntries(tx, contactID, ProfileShowcaseEntryTypeAccount, showcase.Accounts)
	if err != nil {
		return err
	}

	err = db.saveProfileShowcaseContactEntries(tx, contactID, ProfileShowcaseEntryTypeCollectible, showcase.Collectibles)
	if err != nil {
		return err
	}

	err = db.saveProfileShowcaseContactEntries(tx, contactID, ProfileShowcaseEntryTypeAsset, showcase.Assets)
	if err != nil {
		return err
	}

	return nil
}
