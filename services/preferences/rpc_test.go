package preferences

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	rpc2 "github.com/status-im/status-go/pkg/backend/node/rpc"
)

func TestPreferencesRPCMethods(t *testing.T) {
	store := setupTestStore(t)
	api := NewAPI(store)

	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("preferences", api))

	call := func(method string, params string) string {
		t.Helper()
		input := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
		codec := rpc2.NewSingleRequestCodec(input)
		server.ServeCodec(codec.GethCodec(), 0)
		return codec.Output()
	}

	category := "BrowserSettings_test"
	require.NotContains(t, call("preferences_getAll", `["`+category+`"]`), `"error"`)
	out := call("preferences_set", `["`+category+`","restoreOpenTabs","true"]`)
	t.Log("set:", out)
	require.NotContains(t, out, `"error"`)

	all, err := api.GetAll(context.Background(), category)
	require.NoError(t, err)
	require.Equal(t, "true", all["restoreOpenTabs"])
}

func TestPreferencesListCategoriesRPC(t *testing.T) {
	store := setupTestStore(t)
	api := NewAPI(store)

	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("preferences", api))

	call := func(method string, params string) string {
		t.Helper()
		input := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
		codec := rpc2.NewSingleRequestCodec(input)
		server.ServeCodec(codec.GethCodec(), 0)
		return codec.Output()
	}

	require.NoError(t, store.Set("chat", "a", "1"))
	require.NoError(t, store.Set("wallet", "b", "2"))

	out := call("preferences_listCategories", `[]`)
	require.NotContains(t, out, `"error"`)

	var resp struct {
		Result []string `json:"result"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &resp))
	require.Equal(t, []string{"chat", "wallet"}, resp.Result)

	categories, err := api.ListCategories(context.Background())
	require.NoError(t, err)
	require.Equal(t, resp.Result, categories)
}
