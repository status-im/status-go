package connector

import (
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
	c := commands.NewClientSideHandler()

	r.Register("eth_sendTransaction", &commands.SendTransactionCommand{
		Db:            s.db,
		ClientHandler: c,
	})

	// Accounts query and dapp permissions
	r.Register("eth_accounts", &commands.AccountsCommand{Db: s.db})
	r.Register("eth_requestAccounts", &commands.RequestAccountsCommand{
		ClientHandler:   c,
		AccountsCommand: commands.AccountsCommand{Db: s.db},
	})

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

func (api *API) forwardRPC(URL string, inputJSON string) (interface{}, error) {
	dApp, err := persistence.SelectDAppByUrl(api.s.db, URL)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", commands.ErrDAppIsNotPermittedByUser
	}

	var response map[string]interface{}
	rawResponse := api.s.rpc.CallRaw(inputJSON)
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

func (api *API) CallRPC(inputJSON string) (interface{}, error) {
	request, err := commands.RPCRequestFromJSON(inputJSON)
	if err != nil {
		return "", err
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(request)
	}

	return api.forwardRPC(request.URL, inputJSON)
}

func (api *API) RecallDAppPermission(origin string) error {
	return persistence.DeleteDApp(api.s.db, origin)
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
