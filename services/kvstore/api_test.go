package kvstore2

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

	require.NoError(t, api.SaveRlnRateLimitEnabled(context.TODO(), true))

	configs, err := api.GetKvstoreConfigs(context.TODO())
	require.NoError(t, err)

	expected := KvstoreConfigs{
		RlnRateLimitEnabled: true,
	}
	require.Equal(t, expected, *configs)
}
