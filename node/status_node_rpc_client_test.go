package node

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/pkg/testutils"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

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
