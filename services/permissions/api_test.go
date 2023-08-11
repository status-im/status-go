package permissions

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "perm-tests-")
	require.NoError(t, err)
	return NewDB(db), func() { require.NoError(t, cleanup()) }
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := setupTestDB(t)
	return &API{db: db}, cancel
}

func TestDappPermissionsStored(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	expected := []DappPermissions{
		{
			Name:        "first",
			Permissions: []string{"r", "w"},
		},
		{
			Name:        "second",
			Permissions: []string{"r", "x"},
		},
		{
			Name: "third",
		},
	}
	for _, perms := range expected {
		require.NoError(t, api.AddDappPermissions(context.TODO(), perms))
	}
	rst, err := api.GetDappPermissions(context.TODO())
	require.NoError(t, err)
	// sort in lexicographic order by name
	sort.Slice(rst, func(i, j int) bool {
		return rst[i].Name < rst[j].Name
	})
	require.Equal(t, expected[0].Name, rst[0].Name)
	require.Equal(t, expected[0].Permissions, rst[0].Permissions)

	data, err := json.Marshal(rst)
	require.NoError(t, err)
	fmt.Println(string(data))
}

func TestDappPermissionsReplacedOnUpdated(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	perms := DappPermissions{
		Name:        "first",
		Permissions: []string{"r", "w"},
	}
	require.NoError(t, api.AddDappPermissions(context.TODO(), perms))
	perms.Permissions = append(perms.Permissions, "x")
	require.NoError(t, api.AddDappPermissions(context.TODO(), perms))
	rst, err := api.GetDappPermissions(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, perms.Name, rst[0].Name)
	require.Equal(t, perms.Permissions, rst[0].Permissions)

}

func TestDappPermissionsDeleted(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	perms := DappPermissions{
		Name: "first",
	}
	require.NoError(t, api.AddDappPermissions(context.TODO(), perms))
	rst, err := api.GetDappPermissions(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.NoError(t, api.DeleteDappPermissions(context.TODO(), perms.Name))
	rst, err = api.GetDappPermissions(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 0)
}
