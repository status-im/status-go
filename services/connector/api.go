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
		NetworkManager:  s.nm,
	})

	// Active chain per dapp management
	r.Register("eth_chainId", &commands.ChainIDCommand{Db: s.db})
	r.Register("wallet_switchEthereumChain", &commands.SwitchEthereumChainCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})

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

func (api *API) SendTransactionFinished(args commands.SendTransactionFinishedArgs) error {
	return api.c.SendTransactionFinished(args)
}

func (api *API) RequestAccountsFinished(args commands.RequestAccountsFinishedArgs) error {
	return api.c.RequestAccountsFinished(args)
}
