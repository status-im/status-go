package connector

import (
	"context"
	"errors"
	"fmt"

	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
)

var (
	ErrInvalidResponseFromForwardedRpc         = errors.New("invalid response from forwarded RPC")
	ErrCannotOverrideClientIDForHttpConnection = errors.New("cannot override clientId for HTTP connection")
	ErrEmptyClientIDFromTrustedConnection      = errors.New("trusted connection must provide a clientId")
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
		EthClientGetter: s.ethClientGetter,
		FeeManager:      s.feeManager,
		Db:              s.db,
		ClientHandler:   c,
	})
	r.Register("personal_sign", &commands.SignCommand{
		Db:            s.db,
		ClientHandler: c,
	})
	r.Register("eth_signTypedData_v4", &commands.SignCommand{
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
	r.Register("net_version", &commands.NetVersionCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})
	r.Register("wallet_switchEthereumChain", &commands.SwitchEthereumChainCommand{
		Db:             s.db,
		NetworkManager: s.nm,
	})

	// Permissions
	r.Register("wallet_requestPermissions", &commands.RequestPermissionsCommand{
		Db: s.db,
	})
	r.Register("wallet_getPermissions", &commands.GetPermissionsCommand{
		Db: s.db,
	})
	r.Register("wallet_revokePermissions", &commands.RevokePermissionsCommand{
		Db: s.db,
	})

	return &API{
		s: s,
		r: r,
		c: c,
	}
}

func (api *API) forwardRPC(ctx context.Context, request commands.RPCRequest) (interface{}, error) {
	dApp, err := persistence.SelectDApp(api.s.db, request.URL, request.ClientID)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", commands.ErrDAppIsNotPermittedByUser
	}

	rpcClient, err := api.s.ethClientGetter.EthClient(dApp.ChainID)
	if err != nil {
		return "", err
	}

	var result interface{}
	err = rpcClient.CallContext(ctx, &result, request.Method, request.Params...)
	return result, err
}

func (api *API) CallRPC(ctx context.Context, inputJSON string) (interface{}, error) {
	request, err := commands.RPCRequestFromJSON(inputJSON)
	if err != nil {
		return "", err
	}

	// This prevents external clients from spoofing ClientID to impersonate trusted clients
	if IsUntrustedConnection(ctx) {
		if request.ClientID != "" {
			return "", ErrCannotOverrideClientIDForHttpConnection
		}
	} else {
		// Trusted connections MUST provide a ClientID
		if request.ClientID == "" {
			return "", ErrEmptyClientIDFromTrustedConnection
		}
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(ctx, request)
	}

	if !dappRemoteMethodIsAllowed(request.Method) {
		return nil, fmt.Errorf("method %s is not allowed", request.Method)
	}

	return api.forwardRPC(ctx, request)
}

// Deprecated: Use RecallDAppPermissionV2 instead
func (api *API) RecallDAppPermission(origin string) error {
	return api.RecallDAppPermissionV2(origin, "")
}

func (api *API) RecallDAppPermissionV2(origin string, clientID string) error {
	// TODO: close the websocket connection
	return api.c.RecallDAppPermissions(commands.RecallDAppPermissionsArgs{URL: origin, ClientID: clientID})
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

func (api *API) SignAccepted(args commands.SignAcceptedArgs) error {
	return api.c.SignAccepted(args)
}

func (api *API) SignRejected(args commands.RejectedArgs) error {
	return api.c.SignRejected(args)
}
