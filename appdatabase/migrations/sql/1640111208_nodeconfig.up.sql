CREATE TABLE node_config (
  network_id UNSIGNED INT NOT NULL,
  data_dir VARCHAR NOT NULL,
  keystore_dir VARCHAR NOT NULL,
  node_key VARCHAR NOT NULL DEFAULT "",
  no_discovery BOOLEAN DEFAULT false,
  rendezvous BOOLEAN DEFAULT false,
  listen_addr VARCHAR NOT NULL DEFAULT "",
  advertise_addr VARCHAR NOT NULL DEFAULT "",
  name VARCHAR NOT NULL DEFAULT "",
  version VARCHAR NOT NULL DEFAULT "",
  api_modules VARCHAR NOT NULL DEFAULT "",
  tls_enabled BOOLEAN DEFAULT false,
  max_peers UNSIGNED INT,
  max_pending_peers UNSIGNED INT,
  enable_status_service BOOLEAN DEFAULT false,
  enable_ntp_sync BOOLEAN DEFAULT false,
  waku_enabled BOOLEAN DEFAULT false,
  waku2_enabled BOOOLEAN DEFAULT false,
  bridge_enabled BOOLEAN DEFAULT false,
  wallet_enabled BOOLEAN DEFAULT false,
  local_notifications_enabled BOOLEAN DEFAULT false,
  browser_enabled BOOLEAN DEFAULT false,
  permissions_enabled BOOLEAN DEFAULT false,
  mailservers_enabled BOOLEAN DEFAULT false,
  swarm_enabled BOOLEAN DEFAULT false,
  mailserver_registry_address VARCHAR NOT NULL DEFAULT "",
  web3provider_enabled BOOLEAN DEFAULT false,
  ens_enabled BOOLEAN DEFAULT false,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE http_config (
  enabled BOOLEAN DEFAULT false,
  host VARCHAR NOT NULL DEFAULT "",
  port UNSIGNED INT,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE http_virtual_hosts (
  host VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY(host, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE http_cors (
  cors VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY(cors, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE ipc_config (  
  enabled BOOLEAN DEFAULT false,
  file VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE log_config (
  enabled BOOLEAN DEFAULT false,
  mobile_system BOOLEAN DEFAULT false,
  log_dir VARCHAR NOT NULL DEFAULT "",
  file VARCHAR NOT NULL DEFAULT "",
  log_level VARCHAR NOT NULL DEFAULT "INFO",
  max_backups UNSIGNED INT,
  max_size UNSIGNED INT,
  compress_rotated BOOLEAN DEFAULT false,
  log_to_stderr BOOLEAN DEFAULT false,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;
  
CREATE TABLE upstream_config (
  enabled BOOLEAN DEFAULT false,
  url VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE network_config (
  chain_id  UNSIGNED INT,
  chain_name VARCHAR NOT NULL DEFAULT "",
  rpc_url VARCHAR NOT NULL DEFAULT "",
  block_explorer_url VARCHAR NOT NULL DEFAULT "",
  icon_url VARCHAR NOT NULL DEFAULT "",
  native_currency_name VARCHAR NOT NULL DEFAULT "",
  native_currency_symbol VARCHAR NOT NULL DEFAULT "",
  native_currency_decimals UNSIGNED INT,
  is_test BOOLEAN DEFAULT false,
  layer UNSIGNED INT,
  enabled BOOLEAN DEFAULT false,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (chain_id, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE cluster_config (
  enabled BOOLEAN DEFAULT false,
  fleet VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (synthetic_id)
) WITHOUT ROWID;

CREATE TABLE cluster_nodes (
  node VARCHAR NOT NULL,
  type  VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (node, type, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE light_eth_config (
  enabled BOOLEAN DEFAULT false,
  database_cache UNSIGNED INT,
  min_trusted_fraction UNSIGNED INT,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE light_eth_trusted_nodes (
   node VARCHAR NOT NULL,
   synthetic_id VARCHAR DEFAULT 'id',
   PRIMARY KEY (node, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE register_topics (
  topic VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (topic, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE require_topics (
  topic VARCHAR NOT NULL,
  min UNSIGNED INT NOT NULL DEFAULT 0,
  max UNSIGNED INT NOT NULL DEFAULT 0,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (topic, synthetic_id)
) WITHOUT ROWID;
   
CREATE TABLE push_notifications_server_config (
  enabled BOOLEAN DEFAULT false,
  identity VARCHAR NOT NULL DEFAULT "",
  gorush_url VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;
  
CREATE TABLE waku_config (
  enabled BOOLEAN DEFAULT false,
  light_client BOOLEAN DEFAULT false,
  full_node BOOLEAN DEFAULT false,
  enable_mailserver BOOLEAN DEFAULT false,
  data_dir VARCHAR NOT NULL DEFAULT "",
  minimum_pow REAL,
  mailserver_password VARCHAR NOT NULL DEFAULT "",
  mailserver_rate_limit UNSIGNED INT,
  mailserver_data_retention UNSIGNED INT,
  ttl UNSIGNED INT,
  max_message_size UNSIGNED INT,
  enable_rate_limiter BOOLEAN DEFAULT false,
  packet_rate_limit_ip UNSIGNED INT,
  packet_rate_limit_peer_id UNSIGNED INT,
  bytes_rate_limit_ip UNSIGNED INT,
  bytes_rate_limit_peer_id UNSIGNED INT,
  rate_limit_tolerance UNSIGNED INT,
  bloom_filter_mode BOOLEAN DEFAULT false,
  enable_confirmations BOOLEAN DEFAULT false,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE waku_config_db_pg (
  enabled BOOLEAN DEFAULT false,
  uri VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE waku_softblacklisted_peers (
  peer_id VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (peer_id, synthetic_id)
) WITHOUT ROWID;
     
CREATE TABLE wakuv2_config (
  enabled BOOLEAN DEFAULT false,
  host VARCHAR NOT NULL DEFAULT "",
  port UNSIGNED INT,
  keep_alive_interval UNSIGNED INT,
  light_client BOOLEAN DEFAULT false,
  full_node BOOLEAN DEFAULT false,
  discovery_limit UNSIGNED INT,
  persist_peers BOOLEAN DEFAULT false,
  data_dir VARCHAR NOT NULL DEFAULT "",
  max_message_size UNSIGNED INT,
  enable_confirmations BOOLEAN DEFAULT false,
  peer_exchange BOOLEAN DEFAULT true,
  enable_discv5 BOOLEAN DEFAULT false,
  udp_port UNSIGNED INT,
  auto_update BOOLEAN default false,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE wakuv2_custom_nodes (
  name VARCHAR NOT NULL,
  multiaddress VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY(name, synthetic_id)
) WITHOUT ROWID;

CREATE TABLE shhext_config (
  pfs_enabled BOOLEAN DEFAULT false,
  backup_disabled_data_dir VARCHAR NOT NULL DEFAULT "",
  installation_id VARCHAR NOT NULL DEFAULT "",
  mailserver_confirmations BOOLEAN DEFAULT false,
  enable_connection_manager BOOLEAN DEFAULT false,
  enable_last_used_monitor BOOLEAN DEFAULT false,
  connection_target UNSIGNED INT,
  request_delay UNSIGNED BIGINT,
  max_server_failures UNSIGNED INT,
  max_message_delivery_attempts UNSIGNED INT,
  whisper_cache_dir VARCHAR NOT NULL DEFAULT "",
  disable_generic_discovery_topic BOOLEAN DEFAULT false,
  send_v1_messages BOOLEAN DEFAULT false,
  data_sync_enabled BOOLEAN DEFAULT false,
  verify_transaction_url VARCHAR NOT NULL DEFAULT "",
  verify_ens_url VARCHAR NOT NULL DEFAULT "",
  verify_ens_contract_address VARCHAR NOT NULL DEFAULT "",
  verify_transaction_chain_id UNSIGNED INT,
  anon_metrics_server_enabled BOOLEAN DEFAULT false,
  anon_metrics_send_id VARCHAR NOT NULL DEFAULT "",
  anon_metrics_server_postgres_uri VARCHAR NOT NULL DEFAULT "",
  bandwidth_stats_enabled BOOLEAN DEFAULT false,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE shhext_default_push_notification_servers (
  public_key VARCHAR NOT NULL DEFAULT "",
  synthetic_id VARCHAR DEFAULT 'id',
  PRIMARY KEY (public_key, synthetic_id)
) WITHOUT ROWID;
