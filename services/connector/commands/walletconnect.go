package commands

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
)

var errWCClientNotInitialized = fmt.Errorf("WalletConnect client not initialized")

func (g WCClientGetter) resolve() (*walletconnect.Client, error) {
	if g == nil {
		return nil, errWCClientNotInitialized
	}
	c := g()
	if c == nil {
		return nil, errWCClientNotInitialized
	}
	return c, nil
}

type wcSessionDisconnector struct {
	db        *sql.DB
	getClient WCClientGetter
}

// NewWCSessionDisconnector creates a new WCSessionDisconnector
func NewWCSessionDisconnector(db *sql.DB, getClient WCClientGetter) WCSessionDisconnector {
	return &wcSessionDisconnector{
		db:        db,
		getClient: getClient,
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

	if client, err := d.getClient.resolve(); err == nil {
		go func(ctx context.Context, topic string, client *walletconnect.Client) {
			defer common.LogOnPanic()
			_ = client.SendSessionDelete(ctx, topic)
		}(ctx, topic, client)
	}
	return nil
}

type PairWCCommand struct {
	getClient WCClientGetter
}

func NewPairWCCommand(getClient WCClientGetter) *PairWCCommand {
	return &PairWCCommand{
		getClient: getClient,
	}
}

func (c *PairWCCommand) Execute(ctx context.Context, uri string) error {
	client, err := c.getClient.resolve()
	if err != nil {
		return err
	}
	return client.Pair(ctx, uri)
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
	db        *sql.DB
	getClient WCClientGetter
}

func NewApproveWCSessionCommand(db *sql.DB, getClient WCClientGetter) *ApproveWCSessionCommand {
	return &ApproveWCSessionCommand{
		db:        db,
		getClient: getClient,
	}
}

func (c *ApproveWCSessionCommand) Execute(ctx context.Context, proposalID, account string, dappURL, dappName, dappIcon string, supportedChains []uint64) (string, error) {
	if !types.IsHexAddress(account) {
		return "", fmt.Errorf("invalid account address: %s", account)
	}

	client, err := c.getClient.resolve()
	if err != nil {
		return "", err
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

	result, err := client.ApproveSession(ctx, proposalID, meta)
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
	getClient WCClientGetter
}

func NewRejectWCSessionCommand(getClient WCClientGetter) *RejectWCSessionCommand {
	return &RejectWCSessionCommand{
		getClient: getClient,
	}
}

func (c *RejectWCSessionCommand) Execute(ctx context.Context, proposalID string) error {
	client, err := c.getClient.resolve()
	if err != nil {
		return err
	}
	return client.RejectSession(proposalID)
}

type ApproveWCSessionRequestCommand struct {
	getClient WCClientGetter
}

func NewApproveWCSessionRequestCommand(getClient WCClientGetter) *ApproveWCSessionRequestCommand {
	return &ApproveWCSessionRequestCommand{
		getClient: getClient,
	}
}

func (c *ApproveWCSessionRequestCommand) Execute(ctx context.Context, topic, requestIDStr, signature string) error {
	client, err := c.getClient.resolve()
	if err != nil {
		return err
	}
	var requestID int64
	if _, err := fmt.Sscanf(requestIDStr, "%d", &requestID); err != nil {
		return fmt.Errorf("invalid request ID: %w", err)
	}
	return client.RespondToWCSessionRequest(topic, requestID, signature)
}

type RejectWCSessionRequestCommand struct {
	getClient WCClientGetter
}

func NewRejectWCSessionRequestCommand(getClient WCClientGetter) *RejectWCSessionRequestCommand {
	return &RejectWCSessionRequestCommand{
		getClient: getClient,
	}
}

func (c *RejectWCSessionRequestCommand) Execute(ctx context.Context, topic, requestIDStr string, code int, message string) error {
	client, err := c.getClient.resolve()
	if err != nil {
		return err
	}
	var requestID int64
	if _, err := fmt.Sscanf(requestIDStr, "%d", &requestID); err != nil {
		return fmt.Errorf("invalid request ID: %w", err)
	}
	return client.RejectWCSessionRequest(topic, requestID, code, message)
}
