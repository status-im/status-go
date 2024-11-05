package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
)

var (
	ErrInvalidResponseFromForwardedRpc = errors.New("invalid response from forwarded RPC")
)

type API struct {
	s *Service
	r *CommandRegistry
	c *commands.ClientSideHandler
}

func NewAPI(s *Service) *API {
	r := NewCommandRegistry()
	c := commands.NewClientSideHandler(s.db)

	// Transactions and signing
	r.Register("eth_sendTransaction", &commands.SendTransactionCommand{
		RpcClient:     s.rpc,
		Db:            s.db,
		ClientHandler: c,
	})
	r.Register("personal_sign", &commands.PersonalSignCommand{
		Db:            s.db,
		ClientHandler: c,
	})

	// Accounts query and dapp permissions
	// NOTE: Some dApps expect same behavior for both eth_accounts and eth_requestAccounts
	accountsCommand := &commands.RequestAccountsCommand{
		ClientHandler: c,
		Db:            s.db,
	}
	r.Register("eth_accounts", accountsCommand)
	r.Register("eth_requestAccounts", accountsCommand)

	// Active chain per dapp management
	r.Register("eth_chainId", &commands.ChainIDCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})
	r.Register("wallet_switchEthereumChain", &commands.SwitchEthereumChainCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})

	// Permissions
	r.Register("wallet_requestPermissions", &commands.RequestPermissionsCommand{})
	r.Register("wallet_revokePermissions", &commands.RevokePermissionsCommand{
		Db: s.db,
	})

	return &API{
		s: s,
		r: r,
		c: c,
	}
}

func (api *API) forwardRPC(URL string, request commands.RPCRequest) (interface{}, error) {
	dApp, err := persistence.SelectDAppByUrl(api.s.db, URL)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", commands.ErrDAppIsNotPermittedByUser
	}

	if request.ChainID != dApp.ChainID {
		request.ChainID = dApp.ChainID
	}

	var response map[string]interface{}
	byteRequest, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	rawResponse := api.s.rpc.CallRaw(string(byteRequest))
	if err := json.Unmarshal([]byte(rawResponse), &response); err != nil {
		return "", err
	}

	if errorField, ok := response["error"]; ok {
		errorMap, _ := errorField.(map[string]interface{})
		errorCode, _ := errorMap["code"].(float64)
		errorMessage, _ := errorMap["message"].(string)
		return nil, fmt.Errorf("error code %v: %s", errorCode, errorMessage)
	}

	if result, ok := response["result"]; ok {
		return result, nil
	}

	return nil, ErrInvalidResponseFromForwardedRpc
}

func (api *API) CallRPC(ctx context.Context, inputJSON string) (interface{}, error) {
	request, err := commands.RPCRequestFromJSON(inputJSON)
	if err != nil {
		return "", err
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(ctx, request)
	}

	return api.forwardRPC(request.URL, request)
}

func (api *API) RecallDAppPermission(origin string) error {
	// TODO: close the websocket connection
	return api.c.RecallDAppPermissions(commands.RecallDAppPermissionsArgs{URL: origin})
}

func (api *API) GetPermittedDAppsList() ([]persistence.DApp, error) {
	return persistence.SelectAllDApps(api.s.db)
}

func (api *API) RequestAccountsAccepted(args commands.RequestAccountsAcceptedArgs) error {
	return api.c.RequestAccountsAccepted(args)
}

func (api *API) RequestAccountsRejected(args commands.RejectedArgs) error {
	return api.c.RequestAccountsRejected(args)
}

func (api *API) SendTransactionAccepted(args commands.SendTransactionAcceptedArgs) error {
	return api.c.SendTransactionAccepted(args)
}

func (api *API) SendTransactionRejected(args commands.RejectedArgs) error {
	return api.c.SendTransactionRejected(args)
}

func (api *API) PersonalSignAccepted(args commands.PersonalSignAcceptedArgs) error {
	return api.c.PersonalSignAccepted(args)
}

func (api *API) PersonalSignRejected(args commands.RejectedArgs) error {
	return api.c.PersonalSignRejected(args)
}
