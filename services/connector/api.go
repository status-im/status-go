package connector

import (
	"context"
	"errors"
	"fmt"

	"github.com/status-im/status-go/services/connector/chainutils"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
)

var (
	ErrInvalidResponseFromForwardedRpc         = errors.New("invalid response from forwarded RPC")
	ErrCannotOverrideClientIDForHttpConnection = errors.New("cannot override clientId for HTTP connection")
	ErrNotAllowedForUntrustedConnection        = errors.New("cannot call from untrusted connection")
	ErrEmptyClientIDFromTrustedConnection      = errors.New("trusted connection must provide a clientId")
)

type API struct {
	s                    *Service
	r                    *CommandRegistry
	c                    *commands.ClientSideHandler
	changeAccountCommand *commands.ChangeAccountCommand
}

func NewAPI(s *Service) *API {
	r := NewCommandRegistry()
	c := commands.NewClientSideHandler(s.db)

	// Transactions and signing
	r.Register("eth_sendTransaction", commands.NewSendTransactionCommand(s.db, s.ethClientGetter, s.feeManager, c))
	r.Register("personal_sign", commands.NewSignCommand(s.db, c))
	r.Register("eth_signTypedData_v4", commands.NewSignCommand(s.db, c))

	// Accounts query and dapp permissions
	// NOTE: eth_accounts returns accounts only if already permitted, without user prompt
	// eth_requestAccounts always prompts the user for permission (EIP-1102)
	accountsCommand := commands.NewAccountsCommand(s.db)
	requestAccountsCommand := commands.NewRequestAccountsCommand(s.db, c)
	r.Register("eth_accounts", accountsCommand)
	r.Register("eth_requestAccounts", requestAccountsCommand)

	// Active chain per dapp management
	defaultChainIDGetter := chainutils.NewNetworkManagerAdapter(s.nm)
	r.Register("eth_chainId", commands.NewChainIDCommand(s.db, defaultChainIDGetter))
	r.Register("net_version", commands.NewNetVersionCommand(s.db, defaultChainIDGetter))
	r.Register("wallet_switchEthereumChain", commands.NewSwitchEthereumChainCommand(s.db, s.nm))

	// Permissions
	r.Register("wallet_requestPermissions", commands.NewRequestPermissionsCommand(s.db))
	r.Register("wallet_getPermissions", commands.NewGetPermissionsCommand(s.db))
	r.Register("wallet_revokePermissions", commands.NewRevokePermissionsCommand(s.db))

	changeAccountCommand := commands.NewChangeAccountCommand(s.db)

	return &API{
		s:                    s,
		r:                    r,
		c:                    c,
		changeAccountCommand: changeAccountCommand,
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

func (api *API) ChangeAccount(ctx context.Context, args commands.ChangeAccountArgs) error {
	if IsUntrustedConnection(ctx) {
		return ErrNotAllowedForUntrustedConnection
	}
	return api.changeAccountCommand.Execute(args)
}
