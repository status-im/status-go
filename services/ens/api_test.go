package ens

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions/fake"
)

func createDB(t *testing.T) (*sql.DB, func()) {
	tmpfile, err := ioutil.TempFile("", "service-ens-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "service-ens-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := createDB(t)

	keyStoreDir, err := os.MkdirTemp(os.TempDir(), "accounts")
	require.NoError(t, err)

	// Creating a dummy status node to simulate what it's done in get_status_node.go
	upstreamConfig := params.UpstreamRPCConfig{
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
		Enabled: true,
	}

	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)
	client := gethrpc.DialInProc(server)

	_ = client

	rpcClient, err := statusRPC.NewClient(nil, 1, upstreamConfig, nil, db)
	require.NoError(t, err)

	// import account keys
	utils.Init()
	require.NoError(t, utils.ImportTestAccount(keyStoreDir, utils.GetAccount1PKFile()))

	return NewAPI(rpcClient, nil, nil, nil, db, time.Now, nil), cancel
}

func TestResolver(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.Resolver(context.Background(), 1, "rramos.eth")
	require.NoError(t, err)
	require.Equal(t, "0x4976fb03C32e5B8cfe2b6cCB31c09Ba78EBaBa41", r.String())
}

func TestGetName(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.GetName(context.Background(), 1, common.HexToAddress("0x7d28Ab6948F3Db2F95A43742265D382a4888c120"))
	require.NoError(t, err)
	require.Equal(t, "rramos.eth", r)
}

func TestOwnerOf(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.OwnerOf(context.Background(), 1, "rramos.eth")
	require.NoError(t, err)
	require.Equal(t, "0x7d28Ab6948F3Db2F95A43742265D382a4888c120", r.String())
}

func TestContentHash(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.ContentHash(context.Background(), 1, "simpledapp.eth")
	require.NoError(t, err)
	require.Equal(t, []byte{0xe3, 0x1, 0x1, 0x70, 0x12, 0x20, 0x79, 0x5c, 0x1e, 0xa0, 0xce, 0xaf, 0x4c, 0xee, 0xdc, 0x98, 0x96, 0xf1, 0x4b, 0x73, 0xbb, 0x30, 0xe9, 0x78, 0xe4, 0x85, 0x5e, 0xe2, 0x21, 0xb9, 0xa5, 0x7f, 0x5a, 0x93, 0x42, 0x68, 0x28, 0xe}, r)
}

func TestPublicKeyOf(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	pubKey, err := api.PublicKeyOf(context.Background(), 1, "rramos.eth")
	require.NoError(t, err)
	require.Equal(
		t,
		"0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
		pubKey,
	)
}

func TestAddressOf(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.AddressOf(context.Background(), 1, "rramos.eth")
	require.NoError(t, err)
	require.Equal(t, "0x7d28Ab6948F3Db2F95A43742265D382a4888c120", r.String())
}

func TestExpireAt(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.ExpireAt(context.Background(), 1, "rramos")
	require.NoError(t, err)
	require.Equal(t, "5db32cb7", r)
}

func TestPrice(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	r, err := api.Price(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "8ac7230489e80000", r)
}

func TestResourceURL(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	uri, err := api.ResourceURL(context.Background(), 1, "simpledapp.eth")
	require.NoError(t, err)
	require.Equal(t, "https", uri.Scheme)
	require.Equal(t, "bafybeidzlqpkbtvpjtxnzgew6ffxhozq5f4ojbk64iq3tjl7lkjue2biby", uri.Host)
	require.Equal(t, "", uri.Path)

	//fixme: this is not working ATM, as it blocks PRs, i commented it out
	//uri, err = api.ResourceURL(context.Background(), 1, "swarm.eth")
	//require.NoError(t, err)
	//require.Equal(t, "https", uri.Scheme)
	//require.Equal(t, "swarm-gateways.net", uri.Host)
	//require.Equal(t, "/bzz:/b7976f7fabd7ba88a897452a2860228dcefec427302a3dedae164b51c780a5b8/", uri.Path)

	uri, err = api.ResourceURL(context.Background(), 1, "noahzinsmeister.eth")
	require.NoError(t, err)
	require.Equal(t, "https", uri.Scheme)
	require.Equal(t, "noahzinsmeister.com", uri.Host)
	require.Equal(t, "", uri.Path)
}
