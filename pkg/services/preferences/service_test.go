package preferences

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_preferences "github.com/status-im/status-go/pkg/services/preferences/mock"
)

func setupTestAPI(t *testing.T) (*API, *mock_preferences.MockPreferenceStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mock_preferences.NewMockPreferenceStore(ctrl)
	svc := NewServiceWithStore(store)
	api, ok := svc.APIs()[0].Service.(*API)
	require.True(t, ok)
	return api, store
}

func setupTestAPIWithStore(t *testing.T) (*API, *Store) {
	t.Helper()
	store := setupTestStore(t)
	svc := NewServiceWithStore(store)
	api, ok := svc.APIs()[0].Service.(*API)
	require.True(t, ok)
	require.Equal(t, "preferences", svc.APIs()[0].Namespace)
	return api, store
}

func TestAPIMock(t *testing.T) {
	ctx := context.Background()

	t.Run("GetStoreError", func(t *testing.T) {
		api, store := setupTestAPI(t)
		storeErr := errors.New("store failure")
		store.EXPECT().Get("chat", "key").Return("", false, storeErr)
		result, err := api.Get(ctx, "chat", "key")
		require.ErrorIs(t, err, storeErr)
		require.False(t, result.Found)
	})

	t.Run("DeleteCategoryStoreError", func(t *testing.T) {
		api, store := setupTestAPI(t)
		storeErr := errors.New("delete category failed")
		store.EXPECT().DeleteCategory("chat").Return(0, storeErr)
		result, err := api.DeleteCategory(ctx, "chat")
		require.ErrorIs(t, err, storeErr)
		require.Zero(t, result.Removed)
	})

	t.Run("SetDelegatesToStore", func(t *testing.T) {
		api, store := setupTestAPI(t)
		store.EXPECT().Set("chat", "theme", "dark").Return(nil)
		require.NoError(t, api.Set(ctx, "chat", "theme", "dark"))
	})

	t.Run("LoadAndPurgeUnknownDelegatesToStore", func(t *testing.T) {
		api, store := setupTestAPI(t)
		expected := map[string]string{"keep": "1"}
		store.EXPECT().LoadAndPurgeUnknown("chat", []string{"keep"}).Return(expected, nil)
		got, err := api.LoadAndPurgeUnknown(ctx, "chat", []string{"keep"})
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})

	t.Run("LoadAndPurgeUnknownStoreError", func(t *testing.T) {
		api, store := setupTestAPI(t)
		storeErr := errors.New("load and purge failed")
		store.EXPECT().LoadAndPurgeUnknown("chat", []string{"keep"}).Return(nil, storeErr)
		got, err := api.LoadAndPurgeUnknown(ctx, "chat", []string{"keep"})
		require.ErrorIs(t, err, storeErr)
		require.Nil(t, got)
	})
}

func TestBrowserOwnerE2E(t *testing.T) {
	t.Run("FullSessionLifecycle", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)

		all := browserOwnerInit(t, store, category)
		require.Empty(t, all)

		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyRestoreOpenTabs:   "true",
			browserKeyOpenTabs:          sampleOpenTabsTwoTabJSON,
			browserKeyActiveTabIndex:    "1",
			browserSnapshotKey("tab-1"): sampleSnapshotData,
			browserSnapshotKey("tab-2"): sampleSnapshotData,
		}))

		all = browserOwnerInit(t, store, category)
		require.Len(t, all, 5)

		oneTab := `[{"uuid":"tab-1","url":"https://a.example","title":"A","icon":""}]`
		require.NoError(t, store.Set(category, browserKeyOpenTabs, oneTab))

		all = browserOwnerInit(t, store, category)
		require.Contains(t, all, browserSnapshotKey("tab-1"))
		require.NotContains(t, all, browserSnapshotKey("tab-2"))
	})

	t.Run("PurgesLegacyMonolithicSnapshotKey", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyOpenTabs:       sampleOpenTabsOneTabJSON,
			"tabLatestSnapshots":     `{"tab-1":"data:image/png;base64,old"}`,
			browserKeyActiveTabIndex: "0",
		}))
		all := browserOwnerInit(t, store, category)
		require.NotContains(t, all, "tabLatestSnapshots")
	})

	t.Run("APIPurgeOrphanSnapshot", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		ctx := context.Background()
		category := browserCategory(testBrowserAccountPubKey)

		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyOpenTabs:          sampleOpenTabsTwoTabJSON,
			browserSnapshotKey("tab-1"): sampleSnapshotData,
			browserSnapshotKey("tab-2"): sampleSnapshotData,
		}))
		oneTab := `[{"uuid":"tab-1","url":"https://a.example","title":"A","icon":""}]`
		require.NoError(t, store.Set(category, browserKeyOpenTabs, oneTab))

		validKeys, err := browserValidKeys(oneTab)
		require.NoError(t, err)
		result, err := api.PurgeUnknown(ctx, category, validKeys)
		require.NoError(t, err)
		require.Equal(t, 1, result.Removed)

		got, err := api.Get(ctx, category, browserSnapshotKey("tab-2"))
		require.NoError(t, err)
		require.False(t, got.Found)
		got, err = api.Get(ctx, category, browserSnapshotKey("tab-1"))
		require.NoError(t, err)
		require.True(t, got.Found)
	})
}

func TestAPIE2E(t *testing.T) {
	ctx := context.Background()

	t.Run("GetFound", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "theme", "dark"))
		result, err := api.Get(ctx, "chat", "theme")
		require.NoError(t, err)
		require.True(t, result.Found)
		require.Equal(t, "dark", result.Value)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		api, _ := setupTestAPIWithStore(t)
		result, err := api.Get(ctx, "chat", "missing")
		require.NoError(t, err)
		require.False(t, result.Found)
	})

	t.Run("SetMany", func(t *testing.T) {
		api, _ := setupTestAPIWithStore(t)
		kvs := map[string]string{"width": "800", "height": "600"}
		require.NoError(t, api.SetMany(ctx, "chat", kvs))
		all, err := api.GetAll(ctx, "chat")
		require.NoError(t, err)
		require.Equal(t, kvs, all)
	})

	t.Run("ListCategories", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		require.NoError(t, store.Set("wallet", "b", "2"))
		categories, err := api.ListCategories(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"chat", "wallet"}, categories)
	})

	t.Run("DeleteCategory", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		require.NoError(t, store.Set("chat", "b", "2"))
		result, err := api.DeleteCategory(ctx, "chat")
		require.NoError(t, err)
		require.Equal(t, 2, result.Removed)
	})

	t.Run("PurgeUnknown", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "keep", "1"))
		require.NoError(t, store.Set("chat", "drop", "2"))
		result, err := api.PurgeUnknown(ctx, "chat", []string{"keep"})
		require.NoError(t, err)
		require.Equal(t, 1, result.Removed)
	})

	t.Run("PurgeUnknownEmptyValidKeysRejected", func(t *testing.T) {
		api, _ := setupTestAPIWithStore(t)
		_, err := api.PurgeUnknown(ctx, "chat", nil)
		require.ErrorIs(t, err, errEmptyValidKeys)
	})

	t.Run("LoadAndPurgeUnknown", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "keep", "1"))
		require.NoError(t, store.Set("chat", "drop", "2"))
		values, err := api.LoadAndPurgeUnknown(ctx, "chat", []string{"keep"})
		require.NoError(t, err)
		require.Equal(t, map[string]string{"keep": "1"}, values)
		all, err := api.GetAll(ctx, "chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"keep": "1"}, all)
	})

	t.Run("LoadAndPurgeUnknownEmptyCategoryRejected", func(t *testing.T) {
		api, _ := setupTestAPIWithStore(t)
		_, err := api.LoadAndPurgeUnknown(ctx, "", []string{"keep"})
		require.ErrorIs(t, err, errEmptyCategory)
	})

	t.Run("LoadAndPurgeUnknownEmptyValidKeysRejected", func(t *testing.T) {
		api, store := setupTestAPIWithStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		_, err := api.LoadAndPurgeUnknown(ctx, "chat", nil)
		require.ErrorIs(t, err, errEmptyValidKeys)
		all, err := api.GetAll(ctx, "chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"a": "1"}, all)
	})
}
