package walletconnect

import "context"

func NewAPI(db *Database) *API {
	return &API{db: db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (api *API) StoreWalletConnectSession(ctx context.Context, session Session) (Session, error) {
	log.Debug("call to create a bookmark")
	walletConnectSessionStoreResult, err := api.db.InsertWalletConnectSession(session)
	log.Debug("result from database for creating a bookmark", "err", err)
	return walletConnectSessionStoreResult, err
}