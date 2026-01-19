CREATE TABLE logos_storage_config (
  enabled BOOLEAN DEFAULT false,
  log_level TEXT DEFAULT 'info',
  log_format TEXT DEFAULT 'auto',
  metrics_enabled BOOLEAN DEFAULT false,
  metrics_address TEXT DEFAULT '127.0.0.1',
  metrics_port INTEGER DEFAULT 9090,
  data_dir VARCHAR NOT NULL,
  listen_addrs TEXT DEFAULT '["/ip4/0.0.0.0/tcp/0"]',
  nat TEXT DEFAULT 'any',
  disc_port INTEGER DEFAULT 8090,
  net_privkey TEXT DEFAULT 'key',
  bootstrap_nodes TEXT,
  max_peers INTEGER DEFAULT 160,
  num_threads INTEGER DEFAULT 0,
  agent_string TEXT DEFAULT 'LogosStorage',
  repo_kind TEXT DEFAULT 'fs',
  storage_quota INTEGER DEFAULT 21474836480, -- 20 GiB
  block_ttl INTEGER DEFAULT 2592000, -- 30 days
  block_maintenance_interval INTEGER DEFAULT 600, -- 10 min
  block_maintenance_number_of_blocks INTEGER DEFAULT 1000,
  block_retries INTEGER DEFAULT 3000,
  cache_size INTEGER DEFAULT 0,
  log_file TEXT DEFAULT '',
  
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;
