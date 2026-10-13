package appdatabase

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/dbsetup"
	sqlite "github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/nodecfg"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, cleanup, err := testutils.SetupTestSQLDB(DbInitializer{}, "settings-tests-")
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, cleanup()) })
	return db
}

func randomNodeConfig() *params.NodeConfig {
	return &params.NodeConfig{
		NetworkID:          uint64(int64(randomInt(math.MaxInt64))),
		NodeKey:            randomString(),
		APIModules:         randomString(),
		WalletConfig:       params.WalletConfig{Enabled: randomBool(), EFPBaseURL: randomString()},
		BrowsersConfig:     params.BrowsersConfig{Enabled: randomBool()},
		PermissionsConfig:  params.PermissionsConfig{Enabled: randomBool()},
		ConnectorConfig:    params.ConnectorConfig{Enabled: randomBool()},
		LogEnabled:         randomBool(),
		LogDir:             randomString(),
		LogFile:            randomString(),
		LogLevel:           randomString(),
		LogMaxBackups:      randomInt(math.MaxInt64),
		LogMaxSize:         randomInt(math.MaxInt64),
		LogCompressRotated: randomBool(),
		LogToStderr:        randomBool(),
		ClusterConfig: params.ClusterConfig{
			Enabled: randomBool(),
			Fleet:   randomString(),
		},
		ShhextConfig: params.ShhextConfig{
			PFSEnabled:               randomBool(),
			InstallationID:           randomString(),
			MailServerConfirmations:  randomBool(),
			VerifyENSContractAddress: randomString(),
			BandwidthStatsEnabled:    randomBool(),
		},
		WakuV2Config: params.WakuV2Config{
			LightClient: randomBool(),
		},
		LogosStorageConfig: params.LogosStorageConfig{
			Enabled: randomBool(),
			NodeConfig: params.LogosStorageNodeConfig{
				DataDir:                        randomString(),
				DiscoveryPort:                  randomInt(65535),
				BlockRetries:                   randomInt(10),
				LogLevel:                       randomString(),
				LogFormat:                      randomString(),
				MetricsEnabled:                 randomBool(),
				MetricsAddress:                 randomString(),
				MetricsPort:                    randomInt(65535),
				ListenAddrs:                    randomStringSlice(),
				Nat:                            randomString(),
				NetPrivKeyFile:                 randomString(),
				BootstrapNodes:                 randomStringSlice(),
				MaxPeers:                       randomInt(1000),
				NumThreads:                     randomInt(64),
				AgentString:                    randomString(),
				RepoKind:                       randomString(),
				StorageQuota:                   randomInt(math.MaxInt64),
				BlockTtl:                       randomString(),
				BlockMaintenanceInterval:       randomString(),
				BlockMaintenanceNumberOfBlocks: randomInt(1000),
				CacheSize:                      randomInt(math.MaxInt64),
				LogFile:                        randomString(),
			},
		},
	}
}

func TestGetNodeConfig(t *testing.T) {
	db := setupTestDB(t)

	nodeConfig := randomNodeConfig()
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)
	require.Equal(t, nodeConfig, dbNodeConfig)
}

func TestSaveNodeConfig(t *testing.T) {
	db := setupTestDB(t)

	nodeConfig := randomNodeConfig()
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

func TestMigrateNodeConfigBeforeEFPBaseURLColumn(t *testing.T) {
	db, err := sqlite.OpenDB(sqlite.InMemoryPath, "1234567890", dbsetup.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`
		CREATE TABLE settings (
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY,
			node_config BLOB
		);
		CREATE TABLE node_config (
			network_id UNSIGNED INT NOT NULL,
			data_dir VARCHAR NOT NULL,
			keystore_dir VARCHAR NOT NULL,
			node_key VARCHAR NOT NULL DEFAULT "",
			api_modules VARCHAR NOT NULL DEFAULT "",
			enable_ntp_sync BOOLEAN DEFAULT false,
			wallet_enabled BOOLEAN DEFAULT false,
			browser_enabled BOOLEAN DEFAULT false,
			permissions_enabled BOOLEAN DEFAULT false,
			connector_enabled BOOLEAN DEFAULT false,
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
		) WITHOUT ROWID;
		CREATE TABLE log_config (
			enabled BOOLEAN DEFAULT false,
			log_dir VARCHAR NOT NULL DEFAULT "",
			file VARCHAR NOT NULL DEFAULT "",
			log_level VARCHAR NOT NULL DEFAULT "INFO",
			log_namespaces VARCHAR DEFAULT '',
			max_backups UNSIGNED INT,
			max_size UNSIGNED INT,
			compress_rotated BOOLEAN DEFAULT false,
			log_to_stderr BOOLEAN DEFAULT false,
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
		) WITHOUT ROWID;
		CREATE TABLE cluster_config (
			enabled BOOLEAN DEFAULT false,
			fleet VARCHAR NOT NULL DEFAULT "",
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
		) WITHOUT ROWID;
		CREATE TABLE shhext_config (
			pfs_enabled BOOLEAN DEFAULT false,
			installation_id VARCHAR NOT NULL DEFAULT "",
			mailserver_confirmations BOOLEAN DEFAULT false,
			verify_ens_contract_address VARCHAR NOT NULL DEFAULT "",
			bandwidth_stats_enabled BOOLEAN DEFAULT false,
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
		) WITHOUT ROWID;
		CREATE TABLE wakuv2_config (
			light_client BOOLEAN DEFAULT false,
			synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
		) WITHOUT ROWID;
	`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO settings (synthetic_id, node_config) VALUES ('id', ?)`,
		`{"NetworkId":1,"NodeKey":"key","APIModules":"eth","WalletConfig":{"Enabled":true,"EFPBaseURL":"https://efp.example"},"BrowsersConfig":{"Enabled":true},"PermissionsConfig":{"Enabled":false},"ConnectorConfig":{"Enabled":true},"LogEnabled":true,"LogLevel":"INFO","ClusterConfig":{"Enabled":true,"Fleet":"status.prod"},"ShhextConfig":{"PFSEnabled":true,"InstallationID":"inst"},"WakuV2Config":{"LightClient":true}}`)
	require.NoError(t, err)

	require.NoError(t, nodecfg.MigrateNodeConfig(db))

	var networkID int
	var walletEnabled, browserEnabled, connectorEnabled bool
	err = db.QueryRow(`SELECT network_id, wallet_enabled, browser_enabled, connector_enabled FROM node_config WHERE synthetic_id = 'id'`).
		Scan(&networkID, &walletEnabled, &browserEnabled, &connectorEnabled)
	require.NoError(t, err)
	require.Equal(t, 1, networkID)
	require.True(t, walletEnabled)
	require.True(t, browserEnabled)
	require.True(t, connectorEnabled)

	var columnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('node_config') WHERE name = 'efp_base_url'`).Scan(&columnCount)
	require.NoError(t, err)
	require.Equal(t, 0, columnCount)
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
