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

type DisconnectWCSessionCommand struct {
	db       *sql.DB
	wcClient *walletconnect.Client
}

func NewDisconnectWCSessionCommand(db *sql.DB, wcClient *walletconnect.Client) *DisconnectWCSessionCommand {
	return &DisconnectWCSessionCommand{
		db:       db,
		wcClient: wcClient,
	}
}

func (c *DisconnectWCSessionCommand) Execute(ctx context.Context, topic string) error {
	if c.wcClient != nil {
		c.wcClient.RemoveSession(topic)
	}
	session, err := persistence.SelectWCSession(c.db, topic)
	if err == nil && session != nil {
		_ = persistence.DeleteDApp(c.db, session.DAppURL, session.ClientID)
	}
	return persistence.DeleteWCSession(c.db, topic)
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

func (c *ApproveWCSessionCommand) Execute(ctx context.Context, proposalID, account string, chainID uint64, dappURL, dappName, dappIcon string) (string, error) {
	if !types.IsHexAddress(account) {
		return "", fmt.Errorf("invalid account address: %s", account)
	}

	if c.wcClient == nil {
		return "", fmt.Errorf("WalletConnect client not initialized")
	}

	meta := walletconnect.SessionMetadata{
		Account:   account,
		ChainID:   chainID,
		Chains:    []int64{int64(chainID)},
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
	if err := persistence.UpsertWCSession(c.db, result.Topic, result.SessionJSON, result.Expiry, result.PairingTopic, dappURL, createdAt); err != nil {
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
