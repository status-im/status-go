package connector

import (
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
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
	r.Register("eth_chainId", &commands.ChainIDCommand{Db: s.db})
	r.Register("wallet_switchEthereumChain", &commands.SwitchEthereumChainCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})

	// Request permissions
	r.Register("wallet_requestPermissions", &commands.RequestPermissionsCommand{})

	return &API{
		s: s,
		r: r,
		c: c,
	}
}

func (api *API) CallRPC(inputJSON string) (string, error) {
	request, err := commands.RPCRequestFromJSON(inputJSON)
	if err != nil {
		return "", err
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(request)
	}

	return api.s.rpc.CallRaw(inputJSON), nil
}

func (api *API) RecallDAppPermission(origin string) error {
	return persistence.DeleteDApp(api.s.db, origin)
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
