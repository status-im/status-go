package kvstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := setupTestDB(t)
	return &API{db: db}, cancel
}

func TestGetKvstoreConfigs(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	require.NoError(t, api.SetRlnRateLimitEnabled(context.TODO(), true))

	configs, err := api.GetStoreEntry(context.TODO())
	require.NoError(t, err)

	expected := StoreEntry{
		RlnRateLimitEnabled: true,
	}
	require.Equal(t, expected, *configs)
}
