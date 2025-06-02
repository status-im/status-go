package localnotifications

import (
	"database/sql"
)

type Database struct {
	db *sql.DB
}

type NotificationPreference struct {
	Enabled    bool   `json:"enabled"`
	Service    string `json:"service"`
	Event      string `json:"event,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

func (db *Database) GetPreferences() (rst []NotificationPreference, err error) {
	rows, err := db.db.Query("SELECT service, event, identifier, enabled FROM local_notifications_preferences")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pref := NotificationPreference{}
		err = rows.Scan(&pref.Service, &pref.Event, &pref.Identifier, &pref.Enabled)
		if err != nil {
			return nil, err
		}
		rst = append(rst, pref)
	}
	return rst, nil
}

func (db *Database) ChangePreference(p NotificationPreference) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO local_notifications_preferences (enabled, service, event, identifier) VALUES (?, ?, ?, ?)", p.Enabled, p.Service, p.Event, p.Identifier)
	return err
}
