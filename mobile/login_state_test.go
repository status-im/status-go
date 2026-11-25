package statusgo

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/backend"
)

func TestLoginAccountIsInProgressFromTheRequestUntilNodeLogin(t *testing.T) {
	originalBackend, originalRunAsync := statusBackend, runAsync
	t.Cleanup(func() { statusBackend, runAsync = originalBackend, originalRunAsync })
	statusBackend = backend.NewStatusBackend(testutils.MustCreateTestLogger())

	var pendingLogin func() error
	runAsync = func(f func() error) <-chan error {
		pendingLogin = f
		return nil
	}

	var nodeLogins []string
	signal.SetHandler(func(data []byte) {
		if strings.Contains(string(data), `"type":"`+signal.EventLoggedIn+`"`) {
			nodeLogins = append(nodeLogins, string(data))
		}
	})
	t.Cleanup(signal.ResetHandler)

	request := `{"keyUid":"0x1234","password":"0xpassword","kdfIterations":1}`

	var accepted struct {
		Error string `json:"error"`
		Seq   uint64 `json:"seq"`
	}
	require.NoError(t, json.Unmarshal([]byte(loginAccount(request)), &accepted))
	require.Empty(t, accepted.Error)
	require.Equal(t, uint64(1), accepted.Seq, "the accepted request returns its seq")
	require.NotNil(t, pendingLogin, "the login runs asynchronously")

	require.JSONEq(t, `{"state":"inProgress","seq":1}`, getLoginState(),
		"in progress as soon as the request is accepted, before the login runs")

	rejectedJSON := loginAccount(request)
	var rejected APIResponse
	require.NoError(t, json.Unmarshal([]byte(rejectedJSON), &rejected))
	require.Equal(t, backend.ErrLoginInProgress.Error(), rejected.Error)
	require.NotContains(t, rejectedJSON, `"seq"`, "a rejected request has no seq")
	require.JSONEq(t, `{"state":"inProgress","seq":1}`, getLoginState(), "a rejected request changes nothing")

	_ = pendingLogin()

	require.JSONEq(t, `{"state":"none","seq":1}`, getLoginState())
	require.Len(t, nodeLogins, 1)
	require.Contains(t, nodeLogins[0], `"seq":1`)
	require.Contains(t, nodeLogins[0], `"error":`)
}
