package wallet

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestBrowsers(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	api := API{&Service{db: db}}
	require.NoError(t, api.AddBrowser(context.TODO(), Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: hexutil.Uint64(100),
	}))
	require.NoError(t, api.AddBrowser(context.TODO(), Browser{
		ID:   "2",
		Name: "second",
	}))
	require.NoError(t, api.AddBrowser(context.TODO(), Browser{
		ID:           "3",
		Name:         "third",
		HistoryIndex: hexutil.Uint(3),
		History:      []string{"hist1", "hist2"},
	}))
	browsers, err := api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, browsers, 3)

	encoded, err := api.GetBrowsersTransit(context.TODO())
	require.NoError(t, err)
	fmt.Println(string(encoded))

	require.NoError(t, api.DeleteBrowser(context.TODO(), "1"))
	browsers, err = api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, browsers, 2)
}

func TestTransit(t *testing.T) {
	buffer := new(bytes.Buffer)
	enc := NewEncoder(buffer)
	b1 := Browser{
		ID:           "3",
		Name:         "third",
		HistoryIndex: hexutil.Uint(3),
		History:      []string{"hist1", "hist2"},
	}
	b2 := Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: hexutil.Uint64(100),
	}
	require.NoError(t, enc.Encode([]Browser{b1, b2}))
	fmt.Println(buffer.String())
}
