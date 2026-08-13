package connector

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
)

var concurrencyTestAccountAddress = types2.BytesToAddress(types2.FromHex("0x0000000000000000000000000000000000000001"))

func recvRequestIDOrFatal(t *testing.T, ch <-chan string) string {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case rid := <-ch:
		require.NotEmpty(t, rid)
		return rid
	case <-timer.C:
	}
	t.Fatal("timed out waiting for ConnectorSendRequestAccounts signal request ID")
	return ""
}

func setupRequestAccountsRequestIDChan(t *testing.T) <-chan string {
	t.Helper()
	requestIDCh := make(chan string, 1)
	signal.SetHandler(signal.Handler(func(s []byte) {
		var evt commands.EventType
		if err := json.Unmarshal(s, &evt); err != nil || evt.Type != signal.EventConnectorSendRequestAccounts {
			return
		}
		var ev signal.ConnectorSendRequestAccountsSignal
		if err := json.Unmarshal(evt.Event, &ev); err != nil {
			return
		}
		select {
		case requestIDCh <- ev.RequestID:
		default:
		}
	}))
	t.Cleanup(signal.ResetHandler)
	return requestIDCh
}

func requestPermissionsReq(url, clientID, capability string) string {
	clientIDJSON := ""
	if clientID != "" {
		clientIDJSON = `,"clientId":"` + clientID + `"`
	}
	return `{"jsonrpc":"2.0","id":2,"method":"wallet_requestPermissions","params":[{"` + capability + `":{}}],"url":"` + url + `","name":"n","iconUrl":"http://i"` + clientIDJSON + `}`
}

func requestAccountsReq(url, clientID string) string {
	clientIDJSON := ""
	if clientID != "" {
		clientIDJSON = `,"clientId":"` + clientID + `"`
	}
	return `{"jsonrpc":"2.0","id":1,"method":"eth_requestAccounts","params":[],"url":"` + url + `","name":"n","iconUrl":"http://i"` + clientIDJSON + `}`
}

func TestWalletRequestPermissions_ParallelWithEthRequestAccounts(t *testing.T) {
	cases := []struct {
		name            string
		url             string
		clientID        string
		acceptShare     bool
		expectPerms     bool
		expectedPermErr error
		expectedEthErr  error
	}{
		{
			name:            "approve path returns permissions",
			url:             "https://parallel-wait-perm.test",
			acceptShare:     true,
			expectPerms:     true,
			expectedPermErr: nil,
			expectedEthErr:  nil,
		},
		{
			name:            "rejected share returns dapp not found",
			url:             "https://parallel-reject-perm.test",
			acceptShare:     false,
			expectPerms:     false,
			expectedPermErr: commands.ErrDAppNotFound,
			expectedEthErr:  commands.ErrRequestAccountsRejectedByUser,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := setupTests(t)
			requestIDCh := setupRequestAccountsRequestIDChan(t)

			ethReq := requestAccountsReq(tc.url, tc.clientID)
			permReq := requestPermissionsReq(tc.url, tc.clientID, "eth_accounts")
			callCtx := state.ctx
			if tc.clientID != "" {
				callCtx = WithConnectionType(context.Background(), ConnectionTypeTrusted)
			}

			ethErrCh := make(chan error, 1)
			go func() {
				_, err := state.api.CallRPC(callCtx, ethReq)
				ethErrCh <- err
			}()

			rid := recvRequestIDOrFatal(t, requestIDCh)

			permErrCh := make(chan error, 1)
			permResCh := make(chan interface{}, 1)
			go func() {
				res, err := state.api.CallRPC(callCtx, permReq)
				permErrCh <- err
				permResCh <- res
			}()

			select {
			case err := <-permErrCh:
				t.Fatalf("wallet_requestPermissions completed before share resolution (likely race): %v", err)
			case <-time.After(300 * time.Millisecond):
			}

			if tc.acceptShare {
				require.NoError(t, state.api.RequestAccountsAccepted(commands.RequestAccountsAcceptedArgs{
					RequestID: rid,
					Account:   concurrencyTestAccountAddress,
					ChainID:   1,
				}))
			} else {
				require.NoError(t, state.api.RequestAccountsRejected(commands.RejectedArgs{RequestID: rid}))
			}

			if tc.expectedEthErr == nil {
				require.NoError(t, <-ethErrCh)
			} else {
				require.ErrorIs(t, <-ethErrCh, tc.expectedEthErr)
			}
			if tc.expectedPermErr == nil {
				require.NoError(t, <-permErrCh)
			} else {
				require.ErrorIs(t, <-permErrCh, tc.expectedPermErr)
			}

			if tc.expectPerms {
				permRes := <-permResCh
				perms, ok := permRes.([]persistence.Permission)
				require.True(t, ok, "expected []persistence.Permission, got %T", permRes)
				require.Len(t, perms, 1)
				require.Equal(t, persistence.NormalizeURL(tc.url), perms[0].Invoker)
				require.Equal(t, "eth_accounts", perms[0].ParentCapability)
			}
		})
	}
}

func TestWalletRequestPermissions_NoPendingShareAccount_ReturnsDAppNotFoundFast(t *testing.T) {
	state := setupTests(t)
	start := time.Now()
	permReq := requestPermissionsReq("https://no-pending-perm.test", "", "personal_sign")
	_, err := state.api.CallRPC(state.ctx, permReq)
	require.ErrorIs(t, err, commands.ErrDAppNotFound)
	require.Less(t, time.Since(start), 500*time.Millisecond)
}
