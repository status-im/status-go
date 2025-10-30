package commands

import (
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

type ChangeAccountCommand struct {
	Db *sql.DB
}

func (c *ChangeAccountCommand) Execute(args ChangeAccountArgs) error {
	err := args.Validate()
	if err != nil {
		return err
	}

	dApp, err := persistence.SelectDApp(c.Db, args.URL, args.ClientID)
	if err != nil {
		return err
	}

	if dApp == nil {
		return nil
	}

	dApp.SharedAccount = args.Account

	err = persistence.UpsertDApp(c.Db, dApp)
	if err != nil {
		return err
	}

	signal.SendConnectorAccountChanged(args.URL, args.ClientID, args.Account)
	return nil
}
