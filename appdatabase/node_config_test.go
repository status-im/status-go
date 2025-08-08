package appdatabase

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/p2p/discv5"

	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, cleanup, err := helpers.SetupTestSQLDB(DbInitializer{}, "settings-tests-")
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, cleanup()) })
	return db
}

func randomNodeConfig(t *testing.T) *params.NodeConfig {
	nodeConfig := &params.NodeConfig{}

	err := gofakeit.Struct(nodeConfig)
	require.NoError(t, err)

	nodeConfig.ShhextConfig.DefaultPushNotificationsServers = []*params.PushNotificationServer{}
	nodeConfig.NetworkID &= 0 << 63

	return nodeConfig
}

func TestGetNodeConfig(t *testing.T) {
	db := setupTestDB(t)

	nodeConfig := randomNodeConfig(t)
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)
	require.Equal(t, nodeConfig, dbNodeConfig)
}

func TestSaveNodeConfig(t *testing.T) {
	db := setupTestDB(t)

	nodeConfig := randomNodeConfig(t)
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)
	require.Equal(t, *nodeConfig, *dbNodeConfig)
}

func TestMigrateNodeConfig(t *testing.T) {
	// Migration will be run in setupTestDB. If there's an error, that function will fail
	db := setupTestDB(t)

	// node_config column should be empty
	var result string
	err := db.QueryRow("SELECT COALESCE(NULL, 'empty')").Scan(&result)
	require.NoError(t, err)
	require.Equal(t, "empty", result)
}

func randomString() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)[:10]
}

func randomBool() bool {
	return randomInt(2) == 1
}

func randomInt(max int64) int {
	r, _ := rand.Int(rand.Reader, big.NewInt(max))
	return int(r.Int64())
}

func randomStringSlice() []string {
	m := randomInt(7)
	var result []string
	for i := 0; i < m; i++ {
		result = append(result, randomString())
	}
	sort.Strings(result)
	return result
}

func randomTopicSlice() []discv5.Topic {
	randomValues := randomStringSlice()
	var result []discv5.Topic
	for _, v := range randomValues {
		result = append(result, discv5.Topic(v))
	}
	return result
}

func randomTopicLimits() map[discv5.Topic]params.Limits {
	result := make(map[discv5.Topic]params.Limits)
	m := randomInt(7) + 1
	for i := 0; i < m; i++ {
		result[discv5.Topic(fmt.Sprint(i))] = params.Limits{Min: randomInt(2), Max: randomInt(10)}
	}
	return result
}

func randomCustomNodes() map[string]string {
	result := make(map[string]string)
	m := randomInt(7)
	for i := 0; i < m; i++ {
		result[randomString()] = randomString()
	}
	return result
}

func TestConfigValidate(t *testing.T) {
	// GIVEN
	db := setupTestDB(t)

	tmpdir := t.TempDir()
	nodeConfig, err := params.NewNodeConfig(tmpdir, 1777)

	require.NoError(t, err)
	require.NoError(t, nodeConfig.Validate())
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	// WHEN
	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)

	// THEN
	require.NoError(t, dbNodeConfig.Validate())
}

func TestRepairLoadedTorrentConfig(t *testing.T) {
	// GIVEN
	db := setupTestDB(t)

	tmpdir := t.TempDir()
	nodeConfig, err := params.NewNodeConfig(tmpdir, 1777)
	require.NoError(t, err)

	require.NoError(t, nodeConfig.Validate())

	// Write config to db
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	// WHEN: Corrupt the torrent config data as described in the ticket
	// (https://github.com/status-im/status-desktop/issues/14643)
	// Write invalid torrent config to database
	nodeConfig.TorrentConfig.DataDir = ""
	nodeConfig.TorrentConfig.TorrentDir = ""
	nodeConfig.TorrentConfig.Enabled = true
	require.Error(t, nodeConfig.Validate())

	_, err = db.Exec(`INSERT OR REPLACE INTO torrent_config (
    		enabled, port, data_dir, torrent_dir, synthetic_id
		) VALUES (?, ?, ?, ?, 'id')`,
		nodeConfig.TorrentConfig.Enabled,
		nodeConfig.TorrentConfig.Port,
		nodeConfig.TorrentConfig.DataDir,
		nodeConfig.TorrentConfig.TorrentDir,
	)
	require.NoError(t, err)

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)

	// THEN The invalid torrent config should be repaired
	require.Error(t, dbNodeConfig.Validate())
	require.NoError(t, dbNodeConfig.UpdateWithDefaults())
	require.NoError(t, dbNodeConfig.Validate())
}
