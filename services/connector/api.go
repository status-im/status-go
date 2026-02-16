package connector

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/services/connector/chainutils"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
)

var (
	ErrInvalidResponseFromForwardedRpc              = errors.New("invalid response from forwarded RPC")
	ErrCannotOverrideClientIDForUntrustedConnection = errors.New("cannot override clientId for untrusted connection")
	ErrNotAllowedForUntrustedConnection             = errors.New("cannot call from untrusted connection")
	ErrEmptyClientIDFromTrustedConnection           = errors.New("trusted connection must provide a clientId")
)

type API struct {
	s                          *Service
	r                          *CommandRegistry
	c                          *commands.ClientSideHandler
	changeAccountCommand       *commands.ChangeAccountCommand
	pairWCCommand              *commands.PairWCCommand
	disconnectWCSessionCommand *commands.DisconnectWCSessionCommand
	getWCActiveSessionsCommand *commands.GetWCActiveSessionsCommand
	approveWCSessionCommand    *commands.ApproveWCSessionCommand
	rejectWCSessionCommand     *commands.RejectWCSessionCommand
	approveWCSessionRequestCmd *commands.ApproveWCSessionRequestCommand
	rejectWCSessionRequestCmd  *commands.RejectWCSessionRequestCommand
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

	wcClient, err := walletconnect.NewClient(s.config.ProjectID)
	if err != nil {
		s.logger.Error("failed to create WalletConnect client", zap.Error(err))
	}

	return &API{
		s:                          s,
		r:                          r,
		c:                          c,
		changeAccountCommand:       changeAccountCommand,
		pairWCCommand:              commands.NewPairWCCommand(wcClient),
		disconnectWCSessionCommand: commands.NewDisconnectWCSessionCommand(s.db, wcClient),
		getWCActiveSessionsCommand: commands.NewGetWCActiveSessionsCommand(s.db),
		approveWCSessionCommand:    commands.NewApproveWCSessionCommand(s.db, wcClient),
		rejectWCSessionCommand:     commands.NewRejectWCSessionCommand(wcClient),
		approveWCSessionRequestCmd: commands.NewApproveWCSessionRequestCommand(wcClient),
		rejectWCSessionRequestCmd:  commands.NewRejectWCSessionRequestCommand(wcClient),
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
			return "", ErrCannotOverrideClientIDForUntrustedConnection
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

func (api *API) PairWalletConnect(ctx context.Context, uri string) error {
	return api.pairWCCommand.Execute(ctx, uri)
}

func (api *API) DisconnectWCSession(ctx context.Context, topic string) error {
	return api.disconnectWCSessionCommand.Execute(ctx, topic)
}

func (api *API) GetWCActiveSessions(ctx context.Context, validAtTimestamp int64) ([]persistence.WCSession, error) {
	return api.getWCActiveSessionsCommand.Execute(ctx, validAtTimestamp)
}

func (api *API) ApproveWCSession(ctx context.Context, proposalID, account string, chainID uint64, dappURL, dappName, dappIcon string) (string, error) {
	return api.approveWCSessionCommand.Execute(ctx, proposalID, account, chainID, dappURL, dappName, dappIcon)
}

func (api *API) RejectWCSession(ctx context.Context, proposalID string) error {
	return api.rejectWCSessionCommand.Execute(ctx, proposalID)
}

func (api *API) ApproveWCSessionRequest(ctx context.Context, topic, requestIDStr, signature string) error {
	return api.approveWCSessionRequestCmd.Execute(ctx, topic, requestIDStr, signature)
}

func (api *API) RejectWCSessionRequest(ctx context.Context, topic, requestIDStr string, code int, message string) error {
	return api.rejectWCSessionRequestCmd.Execute(ctx, topic, requestIDStr, code, message)
}
