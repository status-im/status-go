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
)

type PersonalSignCommand struct {
	Db            *sql.DB
	ClientHandler ClientSideHandlerInterface
}

type PersonalSignParams struct {
	Challenge string `json:"challenge"`
	Address   string `json:"address"`
}

func (r *RPCRequest) getPersonalSignParams() (*PersonalSignParams, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return nil, ErrEmptyRPCParams
	}

	if len(r.Params) < 2 {
		return nil, ErrInvalidParamsStructure
	}

	// Extract the Challenge and Address fields from paramsArray
	challenge, ok := r.Params[0].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'challenge' field")
	}

	address, ok := r.Params[1].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'address' field")
	}

	// Create and return the PersonalSignParams
	return &PersonalSignParams{
		Challenge: challenge,
		Address:   address,
	}, nil
}

func (c *PersonalSignCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	params, err := request.getPersonalSignParams()
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

	return c.ClientHandler.RequestPersonalSign(signal.ConnectorDApp{
		URL:     request.URL,
		Name:    request.Name,
		IconURL: request.IconURL,
	}, params.Challenge, params.Address)
}
