package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

var (
	ErrInvalidParamsStructure = errors.New("invalid params structure")
	ErrInvalidMethod          = errors.New("invalid method")
)

type SignCommand struct {
	Db            *sql.DB
	ClientHandler ClientSideHandlerInterface
}

type SignParams struct {
	Challenge string `json:"challenge"`
	Address   string `json:"address"`
	Method    string `json:"method"`
}

func (r *RPCRequest) getSignParams() (*SignParams, error) {
	if r.Method != Method_PersonalSign && r.Method != Method_SignTypedDataV4 {
		return nil, ErrInvalidMethod
	}

	if r.Params == nil || len(r.Params) == 0 {
		return nil, ErrEmptyRPCParams
	}

	if len(r.Params) < 2 {
		return nil, ErrInvalidParamsStructure
	}

	challengeIndex := 0
	addressIndex := 1

	if r.Method == Method_SignTypedDataV4 {
		challengeIndex = 1
		addressIndex = 0
	}

	// Extract the Challenge and Address fields from paramsArray
	challenge, ok := r.Params[challengeIndex].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'challenge' field")
	}

	address, ok := r.Params[addressIndex].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'address' field")
	}

	// Create and return the PersonalSignParams
	return &SignParams{
		Challenge: challenge,
		Address:   address,
		Method:    r.Method,
	}, nil
}

func (c *SignCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	params, err := request.getSignParams()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	return c.ClientHandler.RequestSign(signal.ConnectorDApp{
		URL:     request.URL,
		Name:    request.Name,
		IconURL: request.IconURL,
	}, params.Challenge, params.Address, params.Method)
}
