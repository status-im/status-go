package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/services/connector/chainutils"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
	"github.com/status-im/status-go/signal"
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
	wcClient                   *walletconnect.Client
	changeAccountCommand       *commands.ChangeAccountCommand
	pairWCCommand              *commands.PairWCCommand
	wcSessionDisconnector      commands.WCSessionDisconnector
	getWCActiveSessionsCommand *commands.GetWCActiveSessionsCommand
	approveWCSessionCommand    *commands.ApproveWCSessionCommand
	rejectWCSessionCommand     *commands.RejectWCSessionCommand
	approveWCSessionRequestCmd *commands.ApproveWCSessionRequestCommand
	rejectWCSessionRequestCmd  *commands.RejectWCSessionRequestCommand
}

func NewAPI(s *Service) *API {
	r := NewCommandRegistry()

	wcClient, err := walletconnect.NewClient(s.config.ProjectID)
	if err != nil {
		s.logger.Error("failed to create WalletConnect client", zap.Error(err))
	}

	if wcClient != nil {
		activeSessions, err := persistence.SelectActiveWCSessions(s.db, time.Now().Unix())
		if err != nil {
			s.logger.Error("failed to load active WC sessions", zap.Error(err))
		} else if len(activeSessions) > 0 {
			restoredSessions := make([]walletconnect.RestoredSession, 0, len(activeSessions))
			for _, session := range activeSessions {
				restoredSessions = append(restoredSessions, walletconnect.RestoredSession{
					Topic:  session.Topic,
					SymKey: session.SymKey,
				})
			}
			wcClient.RestoreSessions(restoredSessions)
			s.logger.Info("restored WalletConnect sessions", zap.Int("count", len(restoredSessions)))
			if err := wcClient.ConnectAndResubscribe(); err != nil {
				s.logger.Warn("failed to connect relay for restored WC sessions", zap.Error(err))
			}
		}
	}

	wcSessionDisconnector := commands.NewWCSessionDisconnector(s.db, wcClient)

	if wcClient != nil {
		wcClient.SetSessionDeleteHandler(func(topic string) {
			s.logger.Info("received wc_sessionDelete", zap.String("topic", topic))
			session, err := persistence.SelectWCSession(s.db, topic)
			dappURL := ""
			if err == nil && session != nil {
				dappURL = session.DAppURL
			}
			if err := wcSessionDisconnector.DisconnectSession(context.Background(), topic); err != nil {
				s.logger.Error("failed to disconnect WC session", zap.String("topic", topic), zap.Error(err))
			} else {
				s.logger.Info("WC session disconnected", zap.String("topic", topic), zap.String("dappURL", dappURL))
			}
			signal.SendWCSessionDelete(topic, dappURL)
		})
	}
	c := commands.NewClientSideHandler(s.db, wcClient)

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
	r.Register("wallet_requestPermissions", commands.NewRequestPermissionsCommand(s.db, c))
	r.Register("wallet_getPermissions", commands.NewGetPermissionsCommand(s.db))
	r.Register("wallet_revokePermissions", commands.NewRevokePermissionsCommand(s.db, wcSessionDisconnector))
	r.Register("wallet_getCapabilities", commands.NewGetCapabilitiesCommand())

	changeAccountCommand := commands.NewChangeAccountCommand(s.db)

	return &API{
		s:                          s,
		r:                          r,
		c:                          c,
		wcClient:                   wcClient,
		changeAccountCommand:       changeAccountCommand,
		pairWCCommand:              commands.NewPairWCCommand(wcClient),
		wcSessionDisconnector:      wcSessionDisconnector,
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

// DeleteEphemeralDApps removes persisted connector rows for ephemeral (incognito) sessions.
func (api *API) DeleteEphemeralDApps() error {
	return persistence.DeleteEphemeralDApps(api.s.db)
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
	return api.wcSessionDisconnector.DisconnectSession(ctx, topic)
}

func (api *API) GetWCActiveSessions(ctx context.Context, validAtTimestamp int64) ([]persistence.WCSession, error) {
	return api.getWCActiveSessionsCommand.Execute(ctx, validAtTimestamp)
}

func (api *API) ApproveWCSession(ctx context.Context, proposalID, account string, dappURL, dappName, dappIcon string, supportedChains []uint64) (string, error) {
	return api.approveWCSessionCommand.Execute(ctx, proposalID, account, dappURL, dappName, dappIcon, supportedChains)
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

func (api *API) UpdateWCSessionChains(ctx context.Context, topic string, account string, chains []uint64) error {
	if api.wcClient == nil {
		return fmt.Errorf("WalletConnect client not initialized")
	}

	if len(chains) == 0 {
		return fmt.Errorf("chains must not be empty")
	}
	primaryChainID := chains[0]

	// Get existing session from DB to extract metadata
	session, err := persistence.SelectWCSession(api.s.db, topic)
	if err != nil || session == nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Parse existing session JSON to get proposal params
	var existingSession walletconnect.Session
	if err := json.Unmarshal([]byte(session.SessionJSON), &existingSession); err != nil {
		return fmt.Errorf("failed to parse session: %w", err)
	}

	// Convert chains to int64
	chainIDs := make([]int64, len(chains))
	for i, c := range chains {
		chainIDs[i] = int64(c)
	}

	// Carry forward the dApp icon from the persisted session peer metadata.
	dappIcon := ""
	if len(existingSession.Peer.Metadata.Icons) > 0 {
		dappIcon = existingSession.Peer.Metadata.Icons[0]
	}

	// Build updated namespaces using existing proposal structure
	meta := walletconnect.SessionMetadata{
		Account:   account,
		ChainID:   primaryChainID,
		Chains:    chainIDs,
		DAppURL:   session.DAppURL,
		DAppName:  existingSession.Peer.Metadata.Name,
		DAppIcon:  dappIcon,
		ExpirySec: 0,
	}

	// Create a minimal proposal params from existing session
	proposal := &walletconnect.ProposalParams{
		RequiredNamespaces: existingSession.RequiredNamespaces,
	}

	namespaces, _, _ := walletconnect.BuildNamespaces(meta, proposal)

	// Send session update to dApp
	if err := api.wcClient.SendSessionUpdate(topic, namespaces); err != nil {
		return fmt.Errorf("failed to send session update: %w", err)
	}

	// Update session in database
	existingSession.Namespaces = namespaces
	updatedSessionJSON, err := json.Marshal(existingSession)
	if err != nil {
		return fmt.Errorf("failed to marshal updated session: %w", err)
	}

	if err := persistence.UpsertWCSession(api.s.db, topic, string(updatedSessionJSON), session.Expiry, session.PairingTopic, session.DAppURL, session.SymKey, time.Now().Unix()); err != nil {
		return fmt.Errorf("failed to update session in database: %w", err)
	}

	return nil
}

// EmitWCSessionEvent sends a wc_sessionEvent to the dapp (e.g., chainChanged, accountsChanged, connect, disconnect).
func (api *API) EmitWCSessionEvent(ctx context.Context, topic, name, dataJSON, chainID string) error {
	if api.wcClient == nil {
		return fmt.Errorf("WalletConnect client not initialized")
	}

	var data interface{}
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return fmt.Errorf("invalid dataJSON: %w", err)
		}
	}

	event := walletconnect.SessionEvent{
		Name: name,
		Data: data,
	}

	return api.wcClient.SendSessionEvent(topic, event, chainID)
}
