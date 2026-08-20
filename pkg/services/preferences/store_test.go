package preferences

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/testutils"
)

const testBrowserAccountPubKey = "0xdeadbeef"

const browserSnapshotKeyPrefix = "snapshot/"

const (
	browserKeyRestoreOpenTabs = "restoreOpenTabs"
	browserKeyOpenTabs        = "openTabs"
	browserKeyCurrentTabIndex = "currentTabIndex"
	browserKeyActiveTabIndex  = "activeTabIndex"
)

const (
	sampleOpenTabsOneTabJSON = `[{"uuid":"tab-1","url":"https://status.app","title":"Status","icon":""}]`
	sampleOpenTabsTwoTabJSON = `[{"uuid":"tab-1","url":"https://a.example","title":"A","icon":""},{"uuid":"tab-2","url":"https://b.example","title":"B","icon":""}]`
	sampleSnapshotData       = "data:image/png;base64,abc"
)

type browserTab struct {
	UUID  string `json:"uuid"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

func browserCategory(accountPubKey string) string {
	return "BrowserSettings_" + accountPubKey
}

func browserSnapshotKey(tabUUID string) string {
	return browserSnapshotKeyPrefix + tabUUID
}

func browserValidKeys(openTabsJSON string) ([]string, error) {
	keys := []string{
		browserKeyRestoreOpenTabs,
		browserKeyOpenTabs,
		browserKeyActiveTabIndex,
	}
	if openTabsJSON == "" {
		return keys, nil
	}
	var tabs []browserTab
	if err := json.Unmarshal([]byte(openTabsJSON), &tabs); err != nil {
		return nil, err
	}
	for _, tab := range tabs {
		if tab.UUID != "" {
			keys = append(keys, browserSnapshotKeyPrefix+tab.UUID)
		}
	}
	return keys, nil
}

func browserOwnerInit(t *testing.T, store *Store, category string) map[string]string {
	t.Helper()
	all, err := store.GetAll(category)
	require.NoError(t, err)
	validKeys, err := browserValidKeys(all[browserKeyOpenTabs])
	require.NoError(t, err)
	_, err = store.PurgeUnknown(category, validKeys)
	require.NoError(t, err)
	all, err = store.GetAll(category)
	require.NoError(t, err)
	return all
}

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	return NewStore(db)
}

func TestStore(t *testing.T) {
	t.Run("SetGet", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "lastChannel", `{"id":"general"}`))
		value, found, err := store.Get("chat", "lastChannel")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, `{"id":"general"}`, value)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		store := setupTestStore(t)
		value, found, err := store.Get("chat", "missing")
		require.NoError(t, err)
		require.False(t, found)
		require.Empty(t, value)
	})

	t.Run("SetUpdatesExistingValue", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "theme", "dark"))
		require.NoError(t, store.Set("chat", "theme", "light"))
		value, found, err := store.Get("chat", "theme")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "light", value)
	})

	t.Run("GetAll", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		require.NoError(t, store.Set("chat", "b", "2"))
		require.NoError(t, store.Set("wallet", "c", "3"))
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"a": "1", "b": "2"}, all)
	})

	t.Run("ListKeys", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "b", "2"))
		require.NoError(t, store.Set("chat", "a", "1"))
		keys, err := store.ListKeys("chat")
		require.NoError(t, err)
		require.Equal(t, []string{"a", "b"}, keys)
	})

	t.Run("ListCategoriesEmpty", func(t *testing.T) {
		store := setupTestStore(t)
		categories, err := store.ListCategories()
		require.NoError(t, err)
		require.Empty(t, categories)
	})

	t.Run("ListCategoriesDistinct", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		require.NoError(t, store.Set("chat", "b", "2"))
		require.NoError(t, store.Set("wallet", "c", "3"))
		categories, err := store.ListCategories()
		require.NoError(t, err)
		require.Equal(t, []string{"chat", "wallet"}, categories)
	})

	t.Run("SetMany", func(t *testing.T) {
		store := setupTestStore(t)
		err := store.SetMany("chat", map[string]string{"width": "800", "height": "600"})
		require.NoError(t, err)
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"width": "800", "height": "600"}, all)
	})

	t.Run("Delete", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "temp", "value"))
		require.NoError(t, store.Delete("chat", "temp"))
		_, found, err := store.Get("chat", "temp")
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("DeleteCategory", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		require.NoError(t, store.Set("chat", "b", "2"))
		require.NoError(t, store.Set("wallet", "c", "3"))
		removed, err := store.DeleteCategory("chat")
		require.NoError(t, err)
		require.Equal(t, 2, removed)
		all, err := store.GetAll("wallet")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"c": "3"}, all)
	})

	t.Run("PurgeUnknown", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "keep", "1"))
		require.NoError(t, store.Set("chat", "drop", "2"))
		require.NoError(t, store.Set("chat", "also-drop", "3"))
		removed, err := store.PurgeUnknown("chat", []string{"keep"})
		require.NoError(t, err)
		require.Equal(t, 2, removed)
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"keep": "1"}, all)
	})

	t.Run("PurgeUnknownEmptyWhitelistRejected", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		_, err := store.PurgeUnknown("chat", nil)
		require.ErrorIs(t, err, errEmptyValidKeys)
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"a": "1"}, all)
	})

	t.Run("LoadAndPurgeUnknown", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "keep", "1"))
		require.NoError(t, store.Set("chat", "drop", "2"))
		require.NoError(t, store.Set("chat", "also-drop", "3"))
		values, err := store.LoadAndPurgeUnknown("chat", []string{"keep"})
		require.NoError(t, err)
		require.Equal(t, map[string]string{"keep": "1"}, values)
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"keep": "1"}, all)
	})

	t.Run("LoadAndPurgeUnknownEmptyCategory", func(t *testing.T) {
		store := setupTestStore(t)
		_, err := store.LoadAndPurgeUnknown("", []string{"keep"})
		require.ErrorIs(t, err, errEmptyCategory)
	})

	t.Run("LoadAndPurgeUnknownEmptyValidKeysRejected", func(t *testing.T) {
		store := setupTestStore(t)
		require.NoError(t, store.Set("chat", "a", "1"))
		_, err := store.LoadAndPurgeUnknown("chat", nil)
		require.ErrorIs(t, err, errEmptyValidKeys)
		all, err := store.GetAll("chat")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"a": "1"}, all)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		store := setupTestStore(t)
		_, _, err := store.Get("", "key")
		require.ErrorIs(t, err, errEmptyCategory)
		_, _, err = store.Get("chat", "")
		require.ErrorIs(t, err, errEmptyKey)
		require.ErrorIs(t, store.Set("chat", "", "value"), errEmptyKey)
	})
}

func TestBrowserOwner(t *testing.T) {
	t.Run("PersistOpenTabsAndDynamicSnapshots", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyRestoreOpenTabs:   "true",
			browserKeyOpenTabs:          sampleOpenTabsOneTabJSON,
			browserSnapshotKey("tab-1"): sampleSnapshotData,
			browserKeyActiveTabIndex:    "0",
		}))
		all, err := store.GetAll(category)
		require.NoError(t, err)
		require.JSONEq(t, sampleOpenTabsOneTabJSON, all[browserKeyOpenTabs])
		require.Equal(t, sampleSnapshotData, all[browserSnapshotKey("tab-1")])
	})

	t.Run("PurgeRemovesRenamedCurrentTabIndex", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyRestoreOpenTabs: "true",
			browserKeyOpenTabs:        sampleOpenTabsOneTabJSON,
			browserKeyCurrentTabIndex: "2",
		}))
		all := browserOwnerInit(t, store, category)
		require.NotContains(t, all, browserKeyCurrentTabIndex)
		require.NoError(t, store.Set(category, browserKeyActiveTabIndex, "0"))
	})

	t.Run("PurgeKeepsSnapshotsListedInOpenTabs", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		require.NoError(t, store.SetMany(category, map[string]string{
			browserKeyOpenTabs:          sampleOpenTabsTwoTabJSON,
			browserSnapshotKey("tab-1"): sampleSnapshotData,
			browserSnapshotKey("tab-2"): sampleSnapshotData,
			browserKeyCurrentTabIndex:   "1",
		}))
		all := browserOwnerInit(t, store, category)
		require.Contains(t, all, browserSnapshotKey("tab-1"))
		require.Contains(t, all, browserSnapshotKey("tab-2"))
		require.NotContains(t, all, browserKeyCurrentTabIndex)
	})

	t.Run("PurgeDoesNotMigrateRenamedValue", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		require.NoError(t, store.Set(category, browserKeyCurrentTabIndex, "7"))
		all := browserOwnerInit(t, store, category)
		require.NotContains(t, all, browserKeyCurrentTabIndex)
		require.NotContains(t, all, browserKeyActiveTabIndex)
	})

	t.Run("FreshStartNoLegacyImport", func(t *testing.T) {
		store := setupTestStore(t)
		category := browserCategory(testBrowserAccountPubKey)
		all := browserOwnerInit(t, store, category)
		require.Empty(t, all)
	})

	t.Run("ValidKeysIncludesSnapshotPerTabUUID", func(t *testing.T) {
		keys, err := browserValidKeys(sampleOpenTabsTwoTabJSON)
		require.NoError(t, err)
		require.Contains(t, keys, browserSnapshotKey("tab-1"))
		require.Contains(t, keys, browserSnapshotKey("tab-2"))
	})

	t.Run("IsolatedPerAccountCategory", func(t *testing.T) {
		store := setupTestStore(t)
		catA := browserCategory("0xaaa")
		catB := browserCategory("0xbbb")
		require.NoError(t, store.Set(catA, browserKeyOpenTabs, `[{"uuid":"a","url":"https://a.example"}]`))
		require.NoError(t, store.Set(catB, browserKeyOpenTabs, `[{"uuid":"b","url":"https://b.example"}]`))
		allA, err := store.GetAll(catA)
		require.NoError(t, err)
		allB, err := store.GetAll(catB)
		require.NoError(t, err)
		require.Contains(t, allA[browserKeyOpenTabs], "a.example")
		require.Contains(t, allB[browserKeyOpenTabs], "b.example")
	})
}
