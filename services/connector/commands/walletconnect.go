package commands

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
)

type wcSessionDisconnector struct {
	db       *sql.DB
	wcClient *walletconnect.Client
}

// NewWCSessionDisconnector creates a new WCSessionDisconnector
func NewWCSessionDisconnector(db *sql.DB, wcClient *walletconnect.Client) WCSessionDisconnector {
	return &wcSessionDisconnector{
		db:       db,
		wcClient: wcClient,
	}
}

// DisconnectSession disconnects a WalletConnect session by topic.
// It sends a wc_sessionDelete message to the dApp and removes the session from the local DB.
// Relay notification failures are non-fatal: the session is always cleaned up locally.
func (d *wcSessionDisconnector) DisconnectSession(ctx context.Context, topic string) error {
	session, err := persistence.SelectWCSession(d.db, topic)
	if err == nil && session != nil {
		_ = persistence.DeleteWCSession(d.db, topic)
		remaining, _ := persistence.SelectWCSessionsByDAppURL(d.db, session.DAppURL)
		if len(remaining) == 0 {
			_ = persistence.DeleteDApp(d.db, session.DAppURL, session.ClientID)
		}
	} else {
		_ = persistence.DeleteWCSession(d.db, topic)
	}

	if d.wcClient != nil {
		go func() {
			if err := d.wcClient.SendSessionDelete(context.Background(), topic); err != nil {
				fmt.Println("[WC Connector] DisconnectSession: relay send failed (non-fatal):", err)
			}
		}()
	}
	return nil
}

type PairWCCommand struct {
	wcClient *walletconnect.Client
}

func NewPairWCCommand(wcClient *walletconnect.Client) *PairWCCommand {
	return &PairWCCommand{
		wcClient: wcClient,
	}
}

func (c *PairWCCommand) Execute(ctx context.Context, uri string) error {
	return c.wcClient.Pair(ctx, uri)
}

type GetWCActiveSessionsCommand struct {
	db *sql.DB
}

func NewGetWCActiveSessionsCommand(db *sql.DB) *GetWCActiveSessionsCommand {
	return &GetWCActiveSessionsCommand{
		db: db,
	}
}

func (c *GetWCActiveSessionsCommand) Execute(ctx context.Context, validAtTimestamp int64) ([]persistence.WCSession, error) {
	return persistence.SelectActiveWCSessions(c.db, validAtTimestamp)
}

type ApproveWCSessionCommand struct {
	db       *sql.DB
	wcClient *walletconnect.Client
}

func NewApproveWCSessionCommand(db *sql.DB, wcClient *walletconnect.Client) *ApproveWCSessionCommand {
	return &ApproveWCSessionCommand{
		db:       db,
		wcClient: wcClient,
	}
}

func (c *ApproveWCSessionCommand) Execute(ctx context.Context, proposalID, account string, dappURL, dappName, dappIcon string, supportedChains []uint64) (string, error) {
	if !types.IsHexAddress(account) {
		return "", fmt.Errorf("invalid account address: %s", account)
	}

	if c.wcClient == nil {
		return "", fmt.Errorf("WalletConnect client not initialized")
	}

	if len(supportedChains) == 0 {
		return "", fmt.Errorf("supportedChains must not be empty")
	}
	chainID := supportedChains[0]

	chains := make([]int64, 0, len(supportedChains))
	for _, ch := range supportedChains {
		chains = append(chains, int64(ch))
	}

	meta := walletconnect.SessionMetadata{
		Account:   account,
		ChainID:   chainID,
		Chains:    chains,
		DAppURL:   dappURL,
		DAppName:  dappName,
		DAppIcon:  dappIcon,
		ExpirySec: 0,
	}

	result, err := c.wcClient.ApproveSession(ctx, proposalID, meta)
	if err != nil {
		return "", fmt.Errorf("approve session: %w", err)
	}

	addr := types.HexToAddress(account)
	dApp := &persistence.DApp{
		URL:           dappURL,
		Name:          dappName,
		IconURL:       dappIcon,
		ClientID:      walletconnect.ClientIDValue,
		SharedAccount: addr,
		ChainID:       chainID,
	}
	if err := persistence.UpsertDApp(c.db, dApp); err != nil {
		return "", fmt.Errorf("upsert dapp: %w", err)
	}

	createdAt := time.Now().Unix()
	if err := persistence.UpsertWCSession(c.db, result.Topic, result.SessionJSON, result.Expiry, result.PairingTopic, dappURL, result.SymKey, createdAt); err != nil {
		return "", fmt.Errorf("upsert session: %w", err)
	}
	return result.SessionJSON, nil
}

type RejectWCSessionCommand struct {
	wcClient *walletconnect.Client
}

func NewRejectWCSessionCommand(wcClient *walletconnect.Client) *RejectWCSessionCommand {
	return &RejectWCSessionCommand{
		wcClient: wcClient,
	}
}

func (c *RejectWCSessionCommand) Execute(ctx context.Context, proposalID string) error {
	if c.wcClient == nil {
		return fmt.Errorf("WalletConnect client not initialized")
	}
	return c.wcClient.RejectSession(proposalID)
}

type ApproveWCSessionRequestCommand struct {
	wcClient *walletconnect.Client
}

func NewApproveWCSessionRequestCommand(wcClient *walletconnect.Client) *ApproveWCSessionRequestCommand {
	return &ApproveWCSessionRequestCommand{
		wcClient: wcClient,
	}
}

func (c *ApproveWCSessionRequestCommand) Execute(ctx context.Context, topic, requestIDStr, signature string) error {
	if c.wcClient == nil {
		return fmt.Errorf("WalletConnect client not initialized")
	}
	var requestID int64
	if _, err := fmt.Sscanf(requestIDStr, "%d", &requestID); err != nil {
		return fmt.Errorf("invalid request ID: %w", err)
	}
	return c.wcClient.RespondToWCSessionRequest(topic, requestID, signature)
}

type RejectWCSessionRequestCommand struct {
	wcClient *walletconnect.Client
}

func NewRejectWCSessionRequestCommand(wcClient *walletconnect.Client) *RejectWCSessionRequestCommand {
	return &RejectWCSessionRequestCommand{
		wcClient: wcClient,
	}
}

func (c *RejectWCSessionRequestCommand) Execute(ctx context.Context, topic, requestIDStr string, code int, message string) error {
	if c.wcClient == nil {
		return fmt.Errorf("WalletConnect client not initialized")
	}
	var requestID int64
	if _, err := fmt.Sscanf(requestIDStr, "%d", &requestID); err != nil {
		return fmt.Errorf("invalid request ID: %w", err)
	}
	return c.wcClient.RejectWCSessionRequest(topic, requestID, code, message)
}
