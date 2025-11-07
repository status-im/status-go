package commands

import (
	"context"
	"database/sql"

	persistence "github.com/status-im/status-go/services/connector/database"
)

type GetPermissionsCommand struct {
	db *sql.DB
}

func NewGetPermissionsCommand(db *sql.DB) *GetPermissionsCommand {
	return &GetPermissionsCommand{
		db: db,
	}
}

func (c *GetPermissionsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	permissions, err := persistence.SelectPermissions(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	// EIP-2255 specifies that wallet_getPermissions should return an array of permissions
	if permissions == nil {
		return []persistence.Permission{}, nil
	}

	return permissions, nil
}
