package walletconnect

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

func NewAPI(db *Database) *API {
	return &API{db: db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (api *API) StoreWalletConnectSession(ctx context.Context, session Session) (Session, error) {
	log.Debug("call to store walletconnect session")
	walletConnectSessionStoreResult, err := api.db.InsertWalletConnectSession(session)
	log.Debug("result from database for storing a walletconnect session object", "err", err)
	return walletConnectSessionStoreResult, err
}

func (api *API) fetchWalletConnectSession(ctx context.Context) (Session, error) {
	log.Debug("call to fetch walletconnect session")
	walletConnectSessionStoreResult, err := api.db.GetWalletConnectSession()
	log.Debug("result from database for fetching existing walletconnect session object", "err", err)
	return walletConnectSessionStoreResult, err
}