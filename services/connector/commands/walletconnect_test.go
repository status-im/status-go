package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/connector/walletconnect"
)

// --- GetWCActiveSessionsCommand tests ---

func TestGetWCActiveSessionsCommand_Empty(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewGetWCActiveSessionsCommand(db)
	sessions, err := cmd.Execute(context.Background(), 0)
	require.NoError(t, err)
	require.Empty(t, sessions)
}

func TestGetWCActiveSessionsCommand_WithActiveSessions(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	// WC sessions require an associated DApp (foreign key)
	dappURL := "https://dapp.com"
	require.NoError(t, persistence.UpsertDApp(db, &persistence.DApp{
		URL: dappURL, Name: "Test DApp", IconURL: "", ClientID: persistence.WCClientID,
	}))

	err := persistence.UpsertWCSession(db, "active-topic", `{}`, 9999999999, "pairing1", dappURL, "symkey1", 100)
	require.NoError(t, err)
	err = persistence.UpsertWCSession(db, "expired-topic", `{}`, 1000, "pairing2", dappURL, "symkey2", 100)
	require.NoError(t, err)

	cmd := NewGetWCActiveSessionsCommand(db)

	// With timestamp 0 — both sessions returned
	sessions, err := cmd.Execute(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	// With a timestamp that makes the second session expired
	sessions, err = cmd.Execute(context.Background(), 9999999998)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "active-topic", sessions[0].Topic)
}

// --- ApproveWCSessionCommand tests ---

func TestApproveWCSessionCommand_InvalidAccount(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewApproveWCSessionCommand(db, nil)
	_, err := cmd.Execute(context.Background(), "proposal1", "not-an-address", "https://dapp.com", "DApp", "", []uint64{1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid account address")
}

func TestApproveWCSessionCommand_EmptyChains(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	// Need a non-nil client to reach the chains validation (nil check comes before chains check)
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewApproveWCSessionCommand(db, func() *walletconnect.Client { return client })
	_, err = cmd.Execute(context.Background(), "proposal1", "0x1234567890abcdef1234567890abcdef12345678", "https://dapp.com", "DApp", "", []uint64{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "supportedChains must not be empty")
}

func TestApproveWCSessionCommand_NilClient(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewApproveWCSessionCommand(db, nil)
	_, err := cmd.Execute(context.Background(), "proposal1", "0x1234567890abcdef1234567890abcdef12345678", "https://dapp.com", "DApp", "", []uint64{1})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrWCClientNotInitialized)
}

func TestApproveWCSessionCommand_ProposalNotFound(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewApproveWCSessionCommand(db, func() *walletconnect.Client { return client })
	_, err = cmd.Execute(context.Background(), "non-existent-proposal", "0x1234567890abcdef1234567890abcdef12345678", "https://dapp.com", "DApp", "", []uint64{1})
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrProposalNotFound)
}

// --- RejectWCSessionCommand tests ---

func TestRejectWCSessionCommand_NilClient(t *testing.T) {
	cmd := NewRejectWCSessionCommand(nil)
	err := cmd.Execute(context.Background(), "proposal1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrWCClientNotInitialized)
}

func TestRejectWCSessionCommand_ProposalNotFound(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewRejectWCSessionCommand(func() *walletconnect.Client { return client })
	err = cmd.Execute(context.Background(), "non-existent-proposal")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrProposalNotFound)
}

// --- ApproveWCSessionRequestCommand tests ---

func TestApproveWCSessionRequestCommand_NilClient(t *testing.T) {
	cmd := NewApproveWCSessionRequestCommand(nil)
	err := cmd.Execute(context.Background(), "topic", "12345", "0xsignature")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrWCClientNotInitialized)
}

func TestApproveWCSessionRequestCommand_InvalidRequestID(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewApproveWCSessionRequestCommand(func() *walletconnect.Client { return client })
	err = cmd.Execute(context.Background(), "topic", "not-a-number", "0xsignature")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request ID")
}

func TestApproveWCSessionRequestCommand_SessionNotFound(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewApproveWCSessionRequestCommand(func() *walletconnect.Client { return client })
	err = cmd.Execute(context.Background(), "non-existent-topic", "12345", "0xsignature")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

// --- RejectWCSessionRequestCommand tests ---

func TestRejectWCSessionRequestCommand_NilClient(t *testing.T) {
	cmd := NewRejectWCSessionRequestCommand(nil)
	err := cmd.Execute(context.Background(), "topic", "12345", 4001, "User rejected")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrWCClientNotInitialized)
}

func TestRejectWCSessionRequestCommand_InvalidRequestID(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewRejectWCSessionRequestCommand(func() *walletconnect.Client { return client })
	err = cmd.Execute(context.Background(), "topic", "not-a-number", 4001, "User rejected")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request ID")
}

func TestRejectWCSessionRequestCommand_SessionNotFound(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewRejectWCSessionRequestCommand(func() *walletconnect.Client { return client })
	err = cmd.Execute(context.Background(), "non-existent-topic", "12345", 4001, "User rejected")
	require.Error(t, err)
	require.ErrorIs(t, err, walletconnect.ErrSessionNotFound)
}

// --- PairWCCommand tests ---

func TestPairWCCommand_InvalidURI(t *testing.T) {
	client, err := walletconnect.NewClient("test")
	require.NoError(t, err)

	cmd := NewPairWCCommand(func() *walletconnect.Client { return client })
	// An invalid URI that fails ParseURI before any network access
	err = cmd.Execute(context.Background(), "not-a-valid-wc-uri")
	require.Error(t, err)
}
