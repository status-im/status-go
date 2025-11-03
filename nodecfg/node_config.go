package nodecfg

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

func nodeConfigWasMigrated(tx *sql.Tx) (migrated bool, err error) {
	row := tx.QueryRow("SELECT exists(SELECT 1 FROM node_config)")
	switch err := row.Scan(&migrated); err {
	case sql.ErrNoRows, nil:
		return migrated, nil
	default:
		return migrated, err
	}
}

type insertFn func(tx *sql.Tx, c *params.NodeConfig) error

func insertNodeConfig(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
	INSERT OR REPLACE INTO node_config (
		network_id, data_dir, keystore_dir, node_key,
		api_modules, enable_ntp_sync, wallet_enabled,
		browser_enabled, permissions_enabled, connector_enabled, synthetic_id)
		VALUES (
		?, ?, ?, ?,
		?, ?, ?,
		?, ?, ?, 'id'
	)`,
		c.NetworkID, "", "", c.NodeKey, c.APIModules, true,
		c.WalletConfig.Enabled, c.BrowsersConfig.Enabled,
		c.PermissionsConfig.Enabled, c.ConnectorConfig.Enabled,
	)
	return err
}

func insertLogConfig(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
	INSERT OR REPLACE INTO log_config (
		enabled, log_dir, log_level, log_namespaces, max_backups, max_size,
		file, compress_rotated, log_to_stderr, synthetic_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'id')`,
		c.LogEnabled, c.LogDir, c.LogLevel, c.LogNamespaces, c.LogMaxBackups, c.LogMaxSize,
		c.LogFile, c.LogCompressRotated, c.LogToStderr,
	)

	return err
}

func insertClusterConfig(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO cluster_config (enabled, fleet, synthetic_id) VALUES (?, ?, 'id')`, c.ClusterConfig.Enabled, c.ClusterConfig.Fleet)
	return err
}

func insertShhExtConfig(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
	INSERT OR REPLACE INTO shhext_config (
		pfs_enabled, installation_id, mailserver_confirmations,
		verify_ens_contract_address, bandwidth_stats_enabled, synthetic_id
	) VALUES (?, ?, ?, ?, ?, 'id')`,
		c.ShhextConfig.PFSEnabled, c.ShhextConfig.InstallationID, c.ShhextConfig.MailServerConfirmations,
		c.ShhextConfig.VerifyENSContractAddress, c.ShhextConfig.BandwidthStatsEnabled)
	if err != nil {
		return err
	}

	return nil
}

func insertTorrentConfig(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
  INSERT OR REPLACE INTO torrent_config (
    enabled, port, data_dir, torrent_dir, synthetic_id
  ) VALUES (?, ?, ?, ?, 'id')`,
		c.TorrentConfig.Enabled, c.TorrentConfig.Port, c.TorrentConfig.DataDir, c.TorrentConfig.TorrentDir,
	)
	return err
}

// Insert or update codex_config table
func insertCodexConfig(tx *sql.Tx, c *params.NodeConfig) error {
	listenAddrsJSON, err := json.Marshal(c.CodexConfig.CodexNodeConfig.ListenAddrs)
	if err != nil {
		return err
	}
	bootstrapNodesJSON, err := json.Marshal(c.CodexConfig.CodexNodeConfig.BootstrapNodes)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO codex_config (
			enabled, history_archive_data_dir, log_level, log_format, metrics_enabled, metrics_address, metrics_port, data_dir,
			listen_addrs, nat, disc_port, net_privkey, bootstrap_nodes, max_peers, num_threads, agent_string,
			repo_kind, storage_quota, block_ttl, block_maintenance_interval, block_maintenance_number_of_blocks,
			block_retries, cache_size, log_file, synthetic_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'id')`,
		c.CodexConfig.Enabled,
		c.CodexConfig.HistoryArchiveDataDir,
		c.CodexConfig.CodexNodeConfig.LogLevel,
		c.CodexConfig.CodexNodeConfig.LogFormat,
		c.CodexConfig.CodexNodeConfig.MetricsEnabled,
		c.CodexConfig.CodexNodeConfig.MetricsAddress,
		c.CodexConfig.CodexNodeConfig.MetricsPort,
		c.CodexConfig.CodexNodeConfig.DataDir,
		string(listenAddrsJSON),
		c.CodexConfig.CodexNodeConfig.Nat,
		c.CodexConfig.CodexNodeConfig.DiscoveryPort,
		c.CodexConfig.CodexNodeConfig.NetPrivKeyFile,
		string(bootstrapNodesJSON),
		c.CodexConfig.CodexNodeConfig.MaxPeers,
		c.CodexConfig.CodexNodeConfig.NumThreads,
		c.CodexConfig.CodexNodeConfig.AgentString,
		c.CodexConfig.CodexNodeConfig.RepoKind,
		c.CodexConfig.CodexNodeConfig.StorageQuota,
		c.CodexConfig.CodexNodeConfig.BlockTtl,
		c.CodexConfig.CodexNodeConfig.BlockMaintenanceInterval,
		c.CodexConfig.CodexNodeConfig.BlockMaintenanceNumberOfBlocks,
		c.CodexConfig.CodexNodeConfig.BlockRetries,
		c.CodexConfig.CodexNodeConfig.CacheSize,
		c.CodexConfig.CodexNodeConfig.LogFile,
	)
	return err
}

func insertWakuV2ConfigPreMigration(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
	INSERT OR REPLACE INTO wakuv2_config (
		light_client,
		synthetic_id
	) VALUES (?, 'id')`,
		c.WakuV2Config.LightClient,
	)
	if err != nil {
		return err
	}

	return nil
}

func insertWakuV2ConfigPostMigration(tx *sql.Tx, c *params.NodeConfig) error {
	_, err := tx.Exec(`
	UPDATE wakuv2_config
	SET enable_missing_message_verification = ?,
		enable_store_confirmation_for_messages_sent = ?
	WHERE synthetic_id = 'id'`,
		c.WakuV2Config.EnableMissingMessageVerification,
		c.WakuV2Config.EnableStoreConfirmationForMessagesSent,
	)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	UPDATE cluster_config
	SET cluster_id = ?
	WHERE synthetic_id = 'id'`,
		c.ClusterConfig.ClusterID,
	)

	return err
}

// List of inserts to be executed when upgrading a node
// These INSERT queries should not be modified
func nodeConfigUpgradeInserts() []insertFn {
	return []insertFn{
		insertNodeConfig,
		insertLogConfig,
		insertClusterConfig,
		insertShhExtConfig,
		insertWakuV2ConfigPreMigration,
	}
}

func nodeConfigNormalInserts() []insertFn {
	// WARNING: if you are modifying one of the node config tables
	// you need to edit `nodeConfigUpgradeInserts` to guarantee that
	// the selects being used there are not affected.

	return []insertFn{
		insertNodeConfig,
		insertLogConfig,
		insertClusterConfig,
		insertShhExtConfig,
		insertWakuV2ConfigPreMigration,
		insertTorrentConfig,
		insertCodexConfig,
		insertWakuV2ConfigPostMigration,
	}
}

func execInsertFns(inFn []insertFn, tx *sql.Tx, c *params.NodeConfig) error {
	for _, fn := range inFn {
		err := fn(tx, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func insertNodeConfigUpgrade(tx *sql.Tx, c *params.NodeConfig) error {
	return execInsertFns(nodeConfigUpgradeInserts(), tx, c)
}

func SaveConfigWithTx(tx *sql.Tx, c *params.NodeConfig) error {
	insertFNs := nodeConfigNormalInserts()
	return execInsertFns(insertFNs, tx, c)
}

func SaveNodeConfig(db *sql.DB, c *params.NodeConfig) error {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	return SaveConfigWithTx(tx, c)
}

func migrateNodeConfig(tx *sql.Tx) error {
	nodecfg := &params.NodeConfig{}
	err := tx.QueryRow("SELECT node_config FROM settings WHERE synthetic_id = 'id'").Scan(&sqlite.JSONBlob{Data: nodecfg})

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		// Can't migrate because there's no data
		return nil
	}

	err = insertNodeConfigUpgrade(tx, nodecfg)
	if err != nil {
		return err
	}

	return nil
}

func loadNodeConfig(tx *sql.Tx) (*params.NodeConfig, error) {
	nodecfg := &params.NodeConfig{}

	err := tx.QueryRow(`
	SELECT
		network_id, node_key, api_modules,
		wallet_enabled, browser_enabled, permissions_enabled,
		connector_enabled FROM node_config
		WHERE synthetic_id = 'id'
	`).Scan(
		&nodecfg.NetworkID, &nodecfg.NodeKey, &nodecfg.APIModules,
		&nodecfg.WalletConfig.Enabled, &nodecfg.BrowsersConfig.Enabled, &nodecfg.PermissionsConfig.Enabled,
		&nodecfg.ConnectorConfig.Enabled,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Load codex_config
	var listenAddrsStr, bootstrapNodesStr string
	err = tx.QueryRow(`
	  SELECT enabled, history_archive_data_dir, log_level, log_format, metrics_enabled, metrics_address, metrics_port, data_dir,
			 listen_addrs, nat, disc_port, net_privkey, bootstrap_nodes, max_peers, num_threads, agent_string,
			 repo_kind, storage_quota, block_ttl, block_maintenance_interval, block_maintenance_number_of_blocks,
			 block_retries, cache_size, log_file
	  FROM codex_config WHERE synthetic_id = 'id'
	`).Scan(
		&nodecfg.CodexConfig.Enabled,
		&nodecfg.CodexConfig.HistoryArchiveDataDir,
		&nodecfg.CodexConfig.CodexNodeConfig.LogLevel,
		&nodecfg.CodexConfig.CodexNodeConfig.LogFormat,
		&nodecfg.CodexConfig.CodexNodeConfig.MetricsEnabled,
		&nodecfg.CodexConfig.CodexNodeConfig.MetricsAddress,
		&nodecfg.CodexConfig.CodexNodeConfig.MetricsPort,
		&nodecfg.CodexConfig.CodexNodeConfig.DataDir,
		&listenAddrsStr,
		&nodecfg.CodexConfig.CodexNodeConfig.Nat,
		&nodecfg.CodexConfig.CodexNodeConfig.DiscoveryPort,
		&nodecfg.CodexConfig.CodexNodeConfig.NetPrivKeyFile,
		&bootstrapNodesStr,
		&nodecfg.CodexConfig.CodexNodeConfig.MaxPeers,
		&nodecfg.CodexConfig.CodexNodeConfig.NumThreads,
		&nodecfg.CodexConfig.CodexNodeConfig.AgentString,
		&nodecfg.CodexConfig.CodexNodeConfig.RepoKind,
		&nodecfg.CodexConfig.CodexNodeConfig.StorageQuota,
		&nodecfg.CodexConfig.CodexNodeConfig.BlockTtl,
		&nodecfg.CodexConfig.CodexNodeConfig.BlockMaintenanceInterval,
		&nodecfg.CodexConfig.CodexNodeConfig.BlockMaintenanceNumberOfBlocks,
		&nodecfg.CodexConfig.CodexNodeConfig.BlockRetries,
		&nodecfg.CodexConfig.CodexNodeConfig.CacheSize,
		&nodecfg.CodexConfig.CodexNodeConfig.LogFile,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	// Unmarshal JSON fields
	if listenAddrsStr != "" {
		if err := json.Unmarshal([]byte(listenAddrsStr), &nodecfg.CodexConfig.CodexNodeConfig.ListenAddrs); err != nil {
			return nil, err
		}
	}
	if bootstrapNodesStr != "" {
		if err := json.Unmarshal([]byte(bootstrapNodesStr), &nodecfg.CodexConfig.CodexNodeConfig.BootstrapNodes); err != nil {
			return nil, err
		}
	}

	err = tx.QueryRow("SELECT enabled, log_dir, log_level, log_namespaces, file, max_backups, max_size, compress_rotated, log_to_stderr FROM log_config WHERE synthetic_id = 'id'").Scan(
		&nodecfg.LogEnabled, &nodecfg.LogDir, &nodecfg.LogLevel, &nodecfg.LogNamespaces, &nodecfg.LogFile, &nodecfg.LogMaxBackups, &nodecfg.LogMaxSize, &nodecfg.LogCompressRotated, &nodecfg.LogToStderr)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	rows, err := tx.Query(`SELECT
                chain_id, chain_name, rpc_url, block_explorer_url, icon_url, native_currency_name,
                native_currency_symbol, native_currency_decimals, is_test, layer, enabled, chain_color, short_name
        FROM networks ORDER BY chain_id ASC`)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n params.Network
		err = rows.Scan(&n.ChainID, &n.ChainName, &n.RPCURL, &n.BlockExplorerURL, &n.IconURL,
			&n.NativeCurrencyName, &n.NativeCurrencySymbol, &n.NativeCurrencyDecimals, &n.IsTest,
			&n.Layer, &n.Enabled, &n.ChainColor, &n.ShortName,
		)
		if err != nil {
			return nil, err
		}
		nodecfg.Networks = append(nodecfg.Networks, n)
	}

	err = tx.QueryRow("SELECT enabled, fleet, cluster_id FROM cluster_config WHERE synthetic_id = 'id'").Scan(&nodecfg.ClusterConfig.Enabled, &nodecfg.ClusterConfig.Fleet, &nodecfg.ClusterConfig.ClusterID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = tx.QueryRow(`
	SELECT pfs_enabled, installation_id, mailserver_confirmations,
	verify_ens_contract_address,
	bandwidth_stats_enabled FROM shhext_config WHERE synthetic_id = 'id'
	`).Scan(
		&nodecfg.ShhextConfig.PFSEnabled, &nodecfg.ShhextConfig.InstallationID, &nodecfg.ShhextConfig.MailServerConfirmations,
		&nodecfg.ShhextConfig.VerifyENSContractAddress,
		&nodecfg.ShhextConfig.BandwidthStatsEnabled,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = tx.QueryRow(`
  SELECT enabled, port, data_dir, torrent_dir
  FROM torrent_config WHERE synthetic_id = 'id'
  `).Scan(
		&nodecfg.TorrentConfig.Enabled, &nodecfg.TorrentConfig.Port, &nodecfg.TorrentConfig.DataDir, &nodecfg.TorrentConfig.TorrentDir,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = tx.QueryRow(`
	SELECT light_client,
	       enable_missing_message_verification,
	       enable_store_confirmation_for_messages_sent
	FROM wakuv2_config WHERE synthetic_id = 'id'
	`).Scan(
		&nodecfg.WakuV2Config.LightClient,
		&nodecfg.WakuV2Config.EnableMissingMessageVerification,
		&nodecfg.WakuV2Config.EnableStoreConfirmationForMessagesSent,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return nodecfg, nil
}

func MigrateNodeConfig(db *sql.DB) error {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	migrated, err := nodeConfigWasMigrated(tx)
	if err != nil {
		return err
	}

	if !migrated {
		return migrateNodeConfig(tx)
	}

	return nil
}

func GetNodeConfigFromDB(db *sql.DB) (*params.NodeConfig, error) {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	return loadNodeConfig(tx)
}

func SetLightClient(db *sql.DB, enabled bool) error {
	_, err := db.Exec(`UPDATE wakuv2_config SET light_client = ?`, enabled)
	return err
}

func SetStoreConfirmationForMessagesSent(db *sql.DB, enabled bool) error {
	_, err := db.Exec(`UPDATE wakuv2_config SET enable_store_confirmation_for_messages_sent = ?`, enabled)
	return err
}

func SetLogLevel(db *sql.DB, logLevel string) error {
	_, err := db.Exec(`UPDATE log_config SET log_level = ?`, logLevel)
	return err
}

func SetLogNamespaces(db *sql.DB, logNamespaces string) error {
	_, err := db.Exec(`UPDATE log_config SET log_namespaces = ?`, logNamespaces)
	return err
}

func SetLogEnabled(db *sql.DB, enabled bool) error {
	_, err := db.Exec(`UPDATE log_config SET enabled = ?`, enabled)
	return err
}

func SetMaxLogBackups(db *sql.DB, maxLogBackups uint) error {
	_, err := db.Exec(`UPDATE log_config SET max_backups = ?`, maxLogBackups)
	return err
}
