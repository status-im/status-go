package commands

import (
	"context"
	"database/sql"
	"errors"
	"time"

	persistence "github.com/status-im/status-go/services/connector/database"
)

type RequestPermissionsCommand struct {
	Db *sql.DB
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

	dApp, err := persistence.SelectDApp(c.Db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppNotFound
	}

	createdAt := time.Now().Unix()
	err = persistence.InsertPermission(c.Db, request.URL, request.ClientID, methodName, caveats, createdAt)
	if err != nil {
		return "", err
	}

	// EIP-2255 - wallet_requestPermissions returns an array
	return []persistence.Permission{c.getPermissionResponse(request.URL, methodName, caveats)}, nil
}
