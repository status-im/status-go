package commands

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/pkg/services/connector/database"
)

type RequestPermissionsCommand struct {
	db            *sql.DB
	clientHandler ClientSideHandlerInterface
}

func NewRequestPermissionsCommand(db *sql.DB, clientHandler ClientSideHandlerInterface) *RequestPermissionsCommand {
	return &RequestPermissionsCommand{
		db:            db,
		clientHandler: clientHandler,
	}
}

var (
	ErrNoRequestPermissionsParamsFound = errors.New("no request permission params found")
	ErrMultipleKeysFound               = errors.New("multiple methodNames found in request permissions params")
	ErrInvalidParamType                = errors.New("invalid parameter type")
	ErrDAppNotFound                    = errors.New("dApp not found; permission not persisted")
)

func (r *RPCRequest) getRequestPermissionsParam() (string, map[string]interface{}, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return "", nil, ErrEmptyRPCParams
	}

	paramMap, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return "", nil, ErrInvalidParamType
	}

	if len(paramMap) > 1 {
		return "", nil, ErrMultipleKeysFound
	}

	for methodName, caveatsValue := range paramMap {
		caveatsMap, ok := caveatsValue.(map[string]interface{})
		if !ok {
			return methodName, make(map[string]interface{}), nil
		}
		return methodName, caveatsMap, nil
	}

	return "", nil, ErrNoRequestPermissionsParamsFound
}

func (c *RequestPermissionsCommand) parseCaveats(caveatsMap map[string]interface{}) []persistence.Caveat {
	// TODO: Validate caveat types and values according to EIP-2255 specification
	// TODO: Implement caveat enforcement logic (e.g., requiredMethods, expiry, etc.)
	var caveats []persistence.Caveat

	for caveatType, caveatValue := range caveatsMap {
		caveats = append(caveats, persistence.Caveat{
			Type:  caveatType,
			Value: caveatValue,
		})
	}

	return caveats
}

func (c *RequestPermissionsCommand) getPermissionResponse(url string, methodName string, caveats []persistence.Caveat) persistence.Permission {
	return persistence.Permission{
		Invoker:          persistence.NormalizeURL(url),
		ParentCapability: methodName,
		Caveats:          caveats,
	}
}

func findCapability(perms []persistence.Permission, methodName string) ([]persistence.Caveat, bool) {
	for i := range perms {
		if perms[i].ParentCapability == methodName {
			return perms[i].Caveats, true
		}
	}
	return nil, false
}

func (c *RequestPermissionsCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	methodName, caveatsMap, err := request.getRequestPermissionsParam()
	if err != nil {
		return "", err
	}

	caveats := c.parseCaveats(caveatsMap)

	dApp, err := persistence.SelectDApp(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}
	waitedInFlightShare := false
	if dApp == nil && c.clientHandler != nil {
		waitedInFlightShare = c.clientHandler.WaitForPendingShareAccount(ctx, request.URL, request.ClientID)
		dApp, err = persistence.SelectDApp(c.db, request.URL, request.ClientID)
		if err != nil {
			return "", err
		}
	}

	// Auto-share only when the caller identifies with clientId (trusted in-process desktop). Untrusted
	// HTTP callers never have ClientID (API strips/forbids it); parallel tests may finish eth_requestAccounts
	// before we observe pending, so skipping auto-share avoids a second blocking share after reject.
	autoShared := false
	if dApp == nil && methodName == "eth_accounts" && !waitedInFlightShare && request.ClientID != "" {
		dApp, err = shareAndUpsertDApp(c.db, c.clientHandler, request)
		if err != nil {
			return "", err
		}
		autoShared = true
	}

	if dApp == nil {
		return "", ErrDAppNotFound
	}

	permsBefore, err := persistence.SelectPermissions(c.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}
	existingCaveats, hadCapability := findCapability(permsBefore, methodName)

	createdAt := time.Now().Unix()
	err = persistence.InsertPermission(c.db, request.URL, request.ClientID, methodName, caveats, createdAt)
	if err != nil {
		return "", err
	}

	responseCaveats := caveats
	if hadCapability {
		responseCaveats = existingCaveats
	}

	// Notify UI after trusted auto-share created the dApp row (no prior eth_accounts permission existed).
	if autoShared {
		signal.SendConnectorDAppPermissionGranted(connectorDAppFromRequest(request), dApp.SharedAccount, []uint64{dApp.ChainID})
	}

	// EIP-2255 — return persisted caveats (InsertPermission is a no-op when row already exists).
	return []persistence.Permission{c.getPermissionResponse(request.URL, methodName, responseCaveats)}, nil
}
