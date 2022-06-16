package appdatabase

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/p2p/discv5"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	tmpfile, err := ioutil.TempFile("", "settings-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name(), "settings-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestGetNodeConfig(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	nodeConfig := randomNodeConfig()
	require.NoError(t, nodecfg.SaveNodeConfig(db, nodeConfig))

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)
	require.Equal(t, nodeConfig, dbNodeConfig)
}

func TestSaveNodeConfig(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	newNodeConfig := randomNodeConfig()

	require.NoError(t, nodecfg.SaveNodeConfig(db, newNodeConfig))

	dbNodeConfig, err := nodecfg.GetNodeConfigFromDB(db)
	require.NoError(t, err)
	require.Equal(t, *newNodeConfig, *dbNodeConfig)
}

func TestMigrateNodeConfig(t *testing.T) {
	// Migration will be run in setupTestDB. If there's an error, that function will fail
	db, stop := setupTestDB(t)
	defer stop()

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

func randomFloat(max int64) float64 {
	r, _ := rand.Int(rand.Reader, big.NewInt(max))
	return float64(r.Int64()) / (1 << 63)
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

func randomNetworkSlice() []params.Network {
	m := randomInt(7) + 1
	var result []params.Network
	for i := 0; i < m; i++ {
		n := params.Network{
			ChainID:                uint64(i),
			ChainName:              randomString(),
			RPCURL:                 randomString(),
			BlockExplorerURL:       randomString(),
			IconURL:                randomString(),
			NativeCurrencyName:     randomString(),
			NativeCurrencySymbol:   randomString(),
			NativeCurrencyDecimals: uint64(int64(randomInt(math.MaxInt64))),
			IsTest:                 randomBool(),
			Layer:                  uint64(int64(randomInt(math.MaxInt64))),
			Enabled:                randomBool(),
			ChainColor:             randomString(),
			ShortName:              randomString(),
		}
		result = append(result, n)
	}
	return result
}

func randomNodeConfig() *params.NodeConfig {
	privK, _ := crypto.GenerateKey()

	return &params.NodeConfig{
		NetworkID:                 uint64(int64(randomInt(math.MaxInt64))),
		DataDir:                   randomString(),
		KeyStoreDir:               randomString(),
		NodeKey:                   randomString(),
		NoDiscovery:               randomBool(),
		Rendezvous:                randomBool(),
		ListenAddr:                randomString(),
		AdvertiseAddr:             randomString(),
		Name:                      randomString(),
		Version:                   randomString(),
		APIModules:                randomString(),
		TLSEnabled:                randomBool(),
		MaxPeers:                  randomInt(math.MaxInt64),
		MaxPendingPeers:           randomInt(math.MaxInt64),
		EnableStatusService:       randomBool(),
		EnableNTPSync:             randomBool(),
		BridgeConfig:              params.BridgeConfig{Enabled: randomBool()},
		WalletConfig:              params.WalletConfig{Enabled: randomBool()},
		LocalNotificationsConfig:  params.LocalNotificationsConfig{Enabled: randomBool()},
		BrowsersConfig:            params.BrowsersConfig{Enabled: randomBool()},
		PermissionsConfig:         params.PermissionsConfig{Enabled: randomBool()},
		MailserversConfig:         params.MailserversConfig{Enabled: randomBool()},
		Web3ProviderConfig:        params.Web3ProviderConfig{Enabled: randomBool()},
		SwarmConfig:               params.SwarmConfig{Enabled: randomBool()},
		MailServerRegistryAddress: randomString(),
		HTTPEnabled:               randomBool(),
		HTTPHost:                  randomString(),
		HTTPPort:                  randomInt(math.MaxInt64),
		HTTPVirtualHosts:          randomStringSlice(),
		HTTPCors:                  randomStringSlice(),
		IPCEnabled:                randomBool(),
		IPCFile:                   randomString(),
		LogEnabled:                randomBool(),
		LogMobileSystem:           randomBool(),
		LogDir:                    randomString(),
		LogFile:                   randomString(),
		LogLevel:                  randomString(),
		LogMaxBackups:             randomInt(math.MaxInt64),
		LogMaxSize:                randomInt(math.MaxInt64),
		LogCompressRotated:        randomBool(),
		LogToStderr:               randomBool(),
		UpstreamConfig:            params.UpstreamRPCConfig{Enabled: randomBool(), URL: randomString()},
		Networks:                  randomNetworkSlice(),
		ClusterConfig: params.ClusterConfig{
			Enabled:     randomBool(),
			Fleet:       randomString(),
			StaticNodes: randomStringSlice(),
			BootNodes:   randomStringSlice(),
		},
		LightEthConfig: params.LightEthConfig{
			Enabled:            randomBool(),
			DatabaseCache:      randomInt(math.MaxInt64),
			TrustedNodes:       randomStringSlice(),
			MinTrustedFraction: randomInt(math.MaxInt64),
		},
		RegisterTopics: randomTopicSlice(),
		RequireTopics:  randomTopicLimits(),
		PushNotificationServerConfig: pushnotificationserver.Config{
			Enabled:   randomBool(),
			GorushURL: randomString(),
			Identity:  privK,
		},
		ShhextConfig: params.ShhextConfig{
			PFSEnabled:                   randomBool(),
			BackupDisabledDataDir:        randomString(),
			InstallationID:               randomString(),
			MailServerConfirmations:      randomBool(),
			EnableConnectionManager:      randomBool(),
			EnableLastUsedMonitor:        randomBool(),
			ConnectionTarget:             randomInt(math.MaxInt64),
			RequestsDelay:                time.Duration(randomInt(math.MaxInt64)),
			MaxServerFailures:            randomInt(math.MaxInt64),
			MaxMessageDeliveryAttempts:   randomInt(math.MaxInt64),
			WhisperCacheDir:              randomString(),
			DisableGenericDiscoveryTopic: randomBool(),
			SendV1Messages:               randomBool(),
			DataSyncEnabled:              randomBool(),
			VerifyTransactionURL:         randomString(),
			VerifyENSURL:                 randomString(),
			VerifyENSContractAddress:     randomString(),
			VerifyTransactionChainID:     int64(randomInt(math.MaxInt64)),
			AnonMetricsSendID:            randomString(),
			AnonMetricsServerEnabled:     randomBool(),
			AnonMetricsServerPostgresURI: randomString(),
			BandwidthStatsEnabled:        randomBool(),
		},
		WakuV2Config: params.WakuV2Config{
			Enabled:             randomBool(),
			Host:                randomString(),
			Port:                randomInt(math.MaxInt64),
			KeepAliveInterval:   randomInt(math.MaxInt64),
			LightClient:         randomBool(),
			FullNode:            randomBool(),
			DiscoveryLimit:      randomInt(math.MaxInt64),
			PersistPeers:        randomBool(),
			DataDir:             randomString(),
			MaxMessageSize:      uint32(randomInt(math.MaxInt64)),
			EnableConfirmations: randomBool(),
			CustomNodes:         randomCustomNodes(),
			PeerExchange:        randomBool(),
			EnableDiscV5:        randomBool(),
			UDPPort:             randomInt(math.MaxInt64),
			AutoUpdate:          randomBool(),
		},
		WakuConfig: params.WakuConfig{
			Enabled:                 randomBool(),
			LightClient:             randomBool(),
			FullNode:                randomBool(),
			EnableMailServer:        randomBool(),
			DataDir:                 randomString(),
			MinimumPoW:              randomFloat(math.MaxInt64),
			MailServerPassword:      randomString(),
			MailServerRateLimit:     randomInt(math.MaxInt64),
			MailServerDataRetention: randomInt(math.MaxInt64),
			TTL:                     randomInt(math.MaxInt64),
			MaxMessageSize:          uint32(randomInt(math.MaxInt64)),
			DatabaseConfig: params.DatabaseConfig{
				PGConfig: params.PGConfig{
					Enabled: randomBool(),
					URI:     randomString(),
				},
			},
			EnableRateLimiter:      randomBool(),
			PacketRateLimitIP:      int64(randomInt(math.MaxInt64)),
			PacketRateLimitPeerID:  int64(randomInt(math.MaxInt64)),
			BytesRateLimitIP:       int64(randomInt(math.MaxInt64)),
			BytesRateLimitPeerID:   int64(randomInt(math.MaxInt64)),
			RateLimitTolerance:     int64(randomInt(math.MaxInt64)),
			BloomFilterMode:        randomBool(),
			SoftBlacklistedPeerIDs: randomStringSlice(),
			EnableConfirmations:    randomBool(),
		},
	}
}
