package communitytokens

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/communities/token"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestMain(m *testing.M) {
	communityTokenDeployRetryDelay = 5 * time.Millisecond
	os.Exit(m.Run())
}

func newTokenDeployService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	require.NoError(t, sqlite.Migrate(db))

	manager := communities.NewManagerWithDBForTests(db)
	return &Service{Messenger: protocol.NewMessengerWithCommunitiesManagerForTests(manager)}, db
}

func insertCommunityTokenRow(db *sql.DB, chainID int, address string) error {
	_, err := db.Exec(`INSERT INTO community_tokens (
		community_id, address, type, name, symbol, description, chain_id, deploy_state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"community-id", address, protobuf.CommunityTokenType_ERC20, "Token", "TOK", "desc", chainID, token.InProgress)
	return err
}

func TestHandleDeployCommunityTokenAppliesReceiptWhenRowExists(t *testing.T) {
	svc, db := newTokenDeployService(t)
	const chainID = walletCommon.ChainID(42161)
	address := common.HexToAddress("0x1111111111111111111111111111111111111111")
	require.NoError(t, insertCommunityTokenRow(db, int(chainID), address.String()))

	start := time.Now()
	got, err := svc.handleDeployCommunityToken(ac.Failed, address, chainID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, token.Failed, got.DeployState)
	require.Equal(t, address.String(), got.Address)
	require.Equal(t, int(chainID), got.ChainID)
	require.Less(t, time.Since(start), 100*time.Millisecond)
}

func TestHandleDeployCommunityTokenWaitsUntilRowAppears(t *testing.T) {
	svc, db := newTokenDeployService(t)
	const chainID = walletCommon.ChainID(42161)
	address := common.HexToAddress("0x2222222222222222222222222222222222222222")

	inserted := make(chan error, 1)
	go func() {
		defer panics.LogOnPanic()
		time.Sleep(communityTokenDeployRetryDelay / 2)
		inserted <- insertCommunityTokenRow(db, int(chainID), address.String())
	}()

	got, err := svc.handleDeployCommunityToken(ac.Failed, address, chainID)

	require.NoError(t, <-inserted)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, token.Failed, got.DeployState)
	require.Equal(t, address.String(), got.Address)
}

func TestHandleDeployCommunityTokenGivesUpWhenRowNeverAppears(t *testing.T) {
	svc, _ := newTokenDeployService(t)
	address := common.HexToAddress("0x3333333333333333333333333333333333333333")

	got, err := svc.handleDeployCommunityToken(ac.Failed, address, walletCommon.ChainID(42161))

	require.Nil(t, got)
	require.ErrorContains(t, err, "token does not exist")
}

func TestHandleDeployCommunityTokenReturnsLookupError(t *testing.T) {
	svc, db := newTokenDeployService(t)
	_, err := db.Exec(`DROP TABLE community_tokens`)
	require.NoError(t, err)

	start := time.Now()
	got, err := svc.handleDeployCommunityToken(ac.Failed, common.HexToAddress("0x4444444444444444444444444444444444444444"), walletCommon.ChainID(1))

	require.Nil(t, got)
	require.Error(t, err)
	require.Less(t, time.Since(start), 100*time.Millisecond)
}
