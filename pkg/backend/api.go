package backend

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
)

type API struct {
	backend *StatusBackend
}

func (a *API) ListAccounts(ctx context.Context) ([]multiaccounts.Account, error) {
	return a.backend.ListAccounts(ctx)
}

func (a *API) CreateAccount(ctx context.Context, request *requests2.CreateAccount, keycardData *requests2.KeycardData) (*multiaccounts.Account, error) {
	mnemonic, err := accscommon.CreateRandomMnemonicWithDefaultLength()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create random mnemonic")
	}

	//creator := NewAccountCreator(b.rootDataDir, b.logger.Named("account-creator"), b.multiaccountsDB, b.mediaServer)
	return a.backend.StartNodeWithChatKeyOrMnemonic(ctx, request, mnemonic, keycardData, false)

}

//func (a *API) DeleteAccount() error {
//
//}
//

func (a *API) Login(request requests2.Login) (finalErr error) {
	defer func() {
		if finalErr != nil {
			a.backend.logger.Error("login failed", zap.Error(finalErr))
			err := a.backend.services.Stop()
			if err != nil {
				a.backend.logger.Error("login failed - failed to stop services", zap.Error(err))
			}
		}
	}()

	//if err := request.Validate(); err != nil {
	//	return err
	//}

	// Get account from database
	acc, err := a.backend.multiaccountsDB.GetAccount(request.KeyUID)
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

	// Create active account
	activeAcc := &activeAccount{
		account: acc,
	}

	// Open databases and run migrations
	err = activeAcc.ensureAppDBOpened(a.backend.rootDataDir, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to open app database")
	}
	err = activeAcc.ensureAppDBOpened(a.backend.rootDataDir, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to open wallet database")
	}

	// Create accountManager
	accdb, err := accounts.NewDB(activeAcc.appDB)
	if err != nil {
		return errors.Wrap(err, "failed to create accounts db")
	}
	activeAcc.accountsManager, err = accsmanagement.NewAccountsManager(a.backend.logger.Named("accounts-manager"))
	if err != nil {
		return errors.Wrap(err, "failed to create AccountsManager")
	}

	activeAcc.accountsManager.SetRootDataDir(a.backend.rootDataDir)
	activeAcc.accountsManager.SetPersistence(accdb)

	// TEMP: Load old-approach NodeConfig. Check getNodeConfig doc for more info.
	nodeConfig, err := activeAcc.getNodeConfig(request, a.backend.rootDataDir)
	if err != nil {
		return errors.Wrap(err, "failed to get node config")
	}

	// Register and start services, according to persisted settings
	// Root service should only load the settings required to define the list of services to start
	// Other services should load their settings through its persistence interfaces.
	// NOTE: For now we always register and start all services, as we did before.
	//   (messenger and wallet started by client)

	a.backend.activeAccount = activeAcc
	a.backend.services.SetAppDB(activeAcc.appDB)
	a.backend.services.SetWalletDB(activeAcc.walletDB)

	// Post-login required procedures
	chatAddr, err := activeAcc.GetChatAddress()
	if err != nil {
		return err
	}

	err = activeAcc.accountsManager.SetChatAccount(chatAddr, request.Password, request.ChatPrivateKey())
	if err != nil {
		return err
	}

	err = a.backend.services.Start(nodeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to start services")
	}

	//err = b.SelectAccount(login, chatKey)
	//if err != nil {
	//	return err
	//}

	err = a.backend.services.initProtocol(
		activeAcc,
		a.backend.mediaServer,
		a.backend.multiaccountsDB,
		a.backend.prometheusMetrics != nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to init protocol")
	}

	if err = a.backend.services.StartLocalBackup(); err != nil {
		return errors.Wrap(err, "failed to start local backup")
	}

	err = a.backend.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		return errors.Wrap(err, "failed to update account")
	}

	// TODO: local pairing should be a separate service
	if a.backend.LocalPairingStateManager.IsPairing() {
		return nil
	}

	// Send LoggedIn signal to the client
	// TODO: Perhaps this signal makes no sense when Login is a sync call
	err = a.backend.LoggedIn(request.KeyUID, err)
	if err != nil {
		return errors.Wrap(err, "failed to send LoggedIn signal")
	}

	return nil
}

//func (a *API) Logout() error {
//
//}
