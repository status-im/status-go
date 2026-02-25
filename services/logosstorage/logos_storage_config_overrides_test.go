//go:build use_logos_storage
// +build use_logos_storage

package logosstorage_test

import (
	"testing"

	"github.com/logos-storage/logos-storage-go-bindings/storage"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/params"
	. "github.com/status-im/status-go/services/logosstorage"
)

type LogosStorageConfigOverridesTestSuite struct {
	suite.Suite
}

func TestLogosStorageConfigOverridesTestSuite(t *testing.T) {
	suite.Run(t, new(LogosStorageConfigOverridesTestSuite))
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_AllFields() {
	cfg := &params.LogosStorageConfig{}

	overrides := map[string]string{
		// params.LogosStorageConfig fields
		"Enabled": "true",

		// storage.Config fields (nested under LogosStorageNodeConfig)
		"LogosStorageNodeConfig.LogLevel":                       "DEBUG",
		"LogosStorageNodeConfig.LogFormat":                      "json",
		"LogosStorageNodeConfig.MetricsEnabled":                 "true",
		"LogosStorageNodeConfig.MetricsAddress":                 "0.0.0.0",
		"LogosStorageNodeConfig.MetricsPort":                    "9090",
		"LogosStorageNodeConfig.DataDir":                        "/custom/data/dir",
		"LogosStorageNodeConfig.ListenAddrs":                    `["/ip4/0.0.0.0/tcp/4001","/ip4/0.0.0.0/tcp/4002"]`,
		"LogosStorageNodeConfig.Nat":                            "none",
		"LogosStorageNodeConfig.DiscoveryPort":                  "8091",
		"LogosStorageNodeConfig.NetPrivKeyFile":                 "/path/to/key",
		"LogosStorageNodeConfig.BootstrapNodes":                 `["spr:CiUIAhIhA1..","spr:CiUIAhIhA2.."]`,
		"LogosStorageNodeConfig.MaxPeers":                       "200",
		"LogosStorageNodeConfig.NumThreads":                     "4",
		"LogosStorageNodeConfig.AgentString":                    "CustomLogosStorage/1.0",
		"LogosStorageNodeConfig.RepoKind":                       "sqlite",
		"LogosStorageNodeConfig.StorageQuota":                   "50000000000",
		"LogosStorageNodeConfig.BlockTtl":                       "604800",
		"LogosStorageNodeConfig.BlockMaintenanceInterval":       "300",
		"LogosStorageNodeConfig.BlockMaintenanceNumberOfBlocks": "2000",
		"LogosStorageNodeConfig.BlockRetries":                   "5000",
		"LogosStorageNodeConfig.CacheSize":                      "1024",
		"LogosStorageNodeConfig.LogFile":                        "/var/log/logos-storage.log",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Require().NoError(err)

	// Verify params.LogosStorageConfig fields
	s.Equal(true, cfg.Enabled)

	// Verify storage.Config fields
	s.Equal("DEBUG", cfg.LogosStorageNodeConfig.LogLevel)
	s.Equal(storage.LogFormat("json"), cfg.LogosStorageNodeConfig.LogFormat)
	s.Equal(true, cfg.LogosStorageNodeConfig.MetricsEnabled)
	s.Equal("0.0.0.0", cfg.LogosStorageNodeConfig.MetricsAddress)
	s.Equal(9090, cfg.LogosStorageNodeConfig.MetricsPort)
	s.Equal("/custom/data/dir", cfg.LogosStorageNodeConfig.DataDir)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/tcp/4002"}, cfg.LogosStorageNodeConfig.ListenAddrs)
	s.Equal("none", cfg.LogosStorageNodeConfig.Nat)
	s.Equal(8091, cfg.LogosStorageNodeConfig.DiscoveryPort)
	s.Equal("/path/to/key", cfg.LogosStorageNodeConfig.NetPrivKeyFile)
	s.Equal([]string{"spr:CiUIAhIhA1..", "spr:CiUIAhIhA2.."}, cfg.LogosStorageNodeConfig.BootstrapNodes)
	s.Equal(200, cfg.LogosStorageNodeConfig.MaxPeers)
	s.Equal(4, cfg.LogosStorageNodeConfig.NumThreads)
	s.Equal("CustomLogosStorage/1.0", cfg.LogosStorageNodeConfig.AgentString)
	s.Equal(storage.RepoKind("sqlite"), cfg.LogosStorageNodeConfig.RepoKind)
	s.Equal(50000000000, cfg.LogosStorageNodeConfig.StorageQuota)
	s.Equal("604800", cfg.LogosStorageNodeConfig.BlockTtl)
	s.Equal("300", cfg.LogosStorageNodeConfig.BlockMaintenanceInterval)
	s.Equal(2000, cfg.LogosStorageNodeConfig.BlockMaintenanceNumberOfBlocks)
	s.Equal(5000, cfg.LogosStorageNodeConfig.BlockRetries)
	s.Equal(1024, cfg.LogosStorageNodeConfig.CacheSize)
	s.Equal("/var/log/logos-storage.log", cfg.LogosStorageNodeConfig.LogFile)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_NilConfig() {
	err := ApplyLogosStorageConfigOverrides(nil, map[string]string{"Enabled": "true"})
	s.NoError(err, "should handle nil config gracefully")
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_EmptyOverrides() {
	cfg := &params.LogosStorageConfig{}
	err := ApplyLogosStorageConfigOverrides(cfg, map[string]string{})
	s.NoError(err, "should handle empty overrides gracefully")
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_InvalidFieldName() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"NonExistentField": "value",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "unknown field")
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_InvalidBoolValue() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"Enabled": "not-a-bool",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_InvalidIntValue() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig.MetricsPort": "not-a-number",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_EmptyKey() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"":        "should-be-ignored",
		"Enabled": "true",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal(true, cfg.Enabled)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_WhitespaceKey() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"   ":     "should-be-ignored",
		"Enabled": "true",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal(true, cfg.Enabled)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_StringSliceJSON() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig.ListenAddrs": `["/ip4/0.0.0.0/tcp/4001"]`,
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001"}, cfg.LogosStorageNodeConfig.ListenAddrs)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_StringSliceCommaSeparated() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig.BootstrapNodes": "node1,node2,node3",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"node1", "node2", "node3"}, cfg.LogosStorageNodeConfig.BootstrapNodes)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_StringSliceEmpty() {
	cfg := &params.LogosStorageConfig{
		LogosStorageNodeConfig: storage.Config{
			ListenAddrs: []string{"existing"},
		},
	}
	overrides := map[string]string{
		"LogosStorageNodeConfig.ListenAddrs": "",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Nil(cfg.LogosStorageNodeConfig.ListenAddrs)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_NestedPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig.DataDir": "/test/path",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal("/test/path", cfg.LogosStorageNodeConfig.DataDir)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_InvalidNestedPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig.NonExistent.Field": "value",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "unknown field")
	s.Regexp(`\bNonExistent\b`, err.Error())
	s.Contains(err.Error(), "LogosStorageNodeConfig.NonExistent.Field")
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_EmptySegmentInPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"LogosStorageNodeConfig..DataDir": "/test/path",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "invalid empty segment")
}
