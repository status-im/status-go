//go:build use_logos_storage && !lint

package logosstorage

import (
	"github.com/logos-storage/logos-storage-go-bindings/storage"

	"github.com/status-im/status-go/params"
)

func toStorageConfig(cfg params.LogosStorageNodeConfig) storage.Config {
	return storage.Config{
		LogLevel:                       cfg.LogLevel,
		LogFormat:                      storage.LogFormat(cfg.LogFormat),
		MetricsEnabled:                 cfg.MetricsEnabled,
		MetricsAddress:                 cfg.MetricsAddress,
		MetricsPort:                    cfg.MetricsPort,
		DataDir:                        cfg.DataDir,
		ListenAddrs:                    cfg.ListenAddrs,
		Nat:                            cfg.Nat,
		DiscoveryPort:                  cfg.DiscoveryPort,
		NetPrivKeyFile:                 cfg.NetPrivKeyFile,
		BootstrapNodes:                 cfg.BootstrapNodes,
		MaxPeers:                       cfg.MaxPeers,
		NumThreads:                     cfg.NumThreads,
		AgentString:                    cfg.AgentString,
		RepoKind:                       storage.RepoKind(cfg.RepoKind),
		StorageQuota:                   cfg.StorageQuota,
		BlockTtl:                       cfg.BlockTtl,
		BlockMaintenanceInterval:       cfg.BlockMaintenanceInterval,
		BlockMaintenanceNumberOfBlocks: cfg.BlockMaintenanceNumberOfBlocks,
		BlockRetries:                   cfg.BlockRetries,
		CacheSize:                      cfg.CacheSize,
		LogFile:                        cfg.LogFile,
	}
}
