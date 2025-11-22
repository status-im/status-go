package backend

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
	"github.com/status-im/status-go/services/media"
)

type API struct {
	backend *StatusBackend
}

func (a *API) ListAccounts(ctx context.Context) ([]multiaccounts.Account, error) {
	return a.backend.ListAccounts(ctx)
}

func (a *API) CreateAccount(ctx context.Context, request *requests2.CreateAccount, keycardData *requests2.KeycardData) (*multiaccounts.Account, error) {
	//return a.backend.createAccount(ctx, request, keycardData)

	//creator := NewAccountCreator(b.rootDataDir, b.logger.Named("account-creator"), b.multiaccountsDB, b.mediaService)
	//return creator.StartNodeWithChatKeyOrMnemonic(ctx, request, mnemonic, keycardData, true)
	b := a.backend

	activeAcc, err := createAccount(b.rootDataDir, b.logger, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create account")
	}

	// We don't need the account, so we can immediately close it
	err = activeAcc.Close()
	if err != nil {
		b.logger.Error("failed to close account", zap.Error(err))
	}

	// Save account to multiaccounts database
	err = b.multiaccountsDB.SaveAccount(*activeAcc.account)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save account")
	}

	return activeAcc.account, nil
}

//func (a *API) DeleteAccount() error {
//
//}
//

func (a *API) Login(request requests2.Login) error {
	b := a.backend

	// TODO: Validate request
	//if err := request.Validate(); err != nil {
	//	return err

	// Get account from database
	acc, err := b.multiaccountsDB.GetAccount(request.KeyUID)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
	}
	if acc == nil {
		return errors.New("account not found")
	}

	// Set runtime parameters
	if request.RuntimeLogLevel != "" {
		// TODO
	}

	activeAccount, err := login(b.rootDataDir, b.logger, acc, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to login")
	}

	// Update account last login timestamp
	acc.Timestamp = time.Now().Unix()
	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, acc.Timestamp)
	if err != nil {
		// Do not return, just log the error. This counts as a successful login.
		b.logger.Error("failed to update account timestamp", zap.Error(err))
	}

	// Save active account
	b.activeAccount = activeAccount

	// Start services
	err = b.startServices()
	if err != nil {
		// WARNING: stop services and logout if failed?
		return errors.Wrap(err, "failed to start services")
	}

	// NOTE: In new API we don't send LoggedIn signal.
	//       Instead returned error should be used. When err == nil, login was successful.
	//       Settings, account and other info should be fetched independently after the call.

	return nil
}

//func (a *API) Logout() error {
//
//}

func (a *API) MediaService() *media.Service {
	return a.backend.mediaService
}
