package node

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/testutils"
	"github.com/status-im/status-go/t/helpers"
)

type TestServiceAPI struct{}

func (api *TestServiceAPI) SomeMethod(_ context.Context) (string, error) {
	return "some method result", nil
}

func setupTestDBs() (appDB *sql.DB, walletDB *sql.DB, closeFn func() error, err error) {
	appDB, err = helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup app db: %w", err)
	}

	walletDB, err = helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup wallet db: %w", err)
	}
	return appDB, walletDB, func() error {
		appErr := appDB.Close()
		walletErr := walletDB.Close()
		if appErr != nil {
			return fmt.Errorf("failed to close app db: %w", appErr)
		}
		if walletErr != nil {
			return fmt.Errorf("failed to close wallet db: %w", walletErr)
		}
		return nil
	}, err
}

func setupTestMultiDB() (*multiaccounts.Database, func() error, error) {
	tmpfile, err := ioutil.TempFile("", "tests")
	if err != nil {
		return nil, nil, err
	}
	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, nil, err
	}
	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}, nil
}

func createAndStartStatusNode(config *params.NodeConfig) (*StatusNode, error) {
	appDB, walletDB, stop, err := setupTestDBs()
	defer func() {
		err = stop()
	}()
	if err != nil {
		return nil, err
	}

	accsDB, err := accounts.NewDB(appDB)
	if err != nil {
		return nil, err
	}

	accountsManager, err := accsmanagement.NewAccountsManager(testutils.MustCreateTestLogger())
	if err != nil {
		return nil, err
	}
	accountsManager.SetPersistence(accsDB)

	statusNode := New(nil, accountsManager, testutils.MustCreateTestLogger())

	statusNode.appDB = appDB
	statusNode.walletDB = walletDB

	ma, stop2, err := setupTestMultiDB()
	defer func() {
		err := stop2()
		if err != nil {
			statusNode.logger.Error("stopping multiaccount db", zap.Error(err))
		}
	}()
	if err != nil {
		return nil, err
	}
	statusNode.multiaccountsDB = ma

	err = statusNode.Start(config)
	if err != nil {
		return nil, err
	}

	return statusNode, nil
}

func createStatusNode() (*StatusNode, func() error, func() error, error) {
	appDB, walletDB, stop1, err := setupTestDBs()
	if err != nil {
		return nil, nil, nil, err
	}

	accsDB, err := accounts.NewDB(appDB)
	if err != nil {
		return nil, nil, nil, err
	}

	accountsManager, err := accsmanagement.NewAccountsManager(testutils.MustCreateTestLogger())
	if err != nil {
		return nil, nil, nil, err
	}
	accountsManager.SetPersistence(accsDB)

	statusNode := New(nil, accountsManager, testutils.MustCreateTestLogger())
	statusNode.SetAppDB(appDB)
	statusNode.SetWalletDB(walletDB)

	ma, stop2, err := setupTestMultiDB()
	statusNode.SetMultiaccountsDB(ma)

	return statusNode, stop1, stop2, err
}
