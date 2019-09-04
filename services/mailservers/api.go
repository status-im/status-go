package mailservers

import "context"

func NewAPI(db *Database) *API {
	return &API{db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (a *API) AddMailserver(ctx context.Context, m Mailserver) error {
	return a.db.Add(m)
}

func (a *API) GetMailservers(ctx context.Context) ([]Mailserver, error) {
	return a.db.Mailservers()
}

func (a *API) DeleteMailserver(ctx context.Context, id string) error {
	return a.db.Delete(id)
}
