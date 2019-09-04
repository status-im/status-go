package mailservers

import "database/sql"

type Mailserver struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Password string `json:"password,omitempty"`
	Fleet    string `json:"fleet"`
}

func (m Mailserver) nullablePassword() (val sql.NullString) {
	if m.Password != "" {
		val.String = m.Password
		val.Valid = true
	}
	return
}

// Database sql wrapper for operations with mailserver objects.
type Database struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

func (d *Database) Add(mailserver Mailserver) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailservers(
			id,
			name,
			address,
			password,
			fleet
		) VALUES (?, ?, ?, ?, ?)`,
		mailserver.ID,
		mailserver.Name,
		mailserver.Address,
		mailserver.nullablePassword(),
		mailserver.Fleet,
	)
	return err
}

func (d *Database) Mailservers() ([]Mailserver, error) {
	var result []Mailserver

	rows, err := d.db.Query(`SELECT id, name, address, password, fleet FROM mailservers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			m        Mailserver
			password sql.NullString
		)
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.Address,
			&password,
			&m.Fleet,
		); err != nil {
			return nil, err
		}
		if password.Valid {
			m.Password = password.String
		}
		result = append(result, m)
	}

	return result, nil
}

func (d *Database) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM mailservers WHERE id = ?`, id)
	return err
}
