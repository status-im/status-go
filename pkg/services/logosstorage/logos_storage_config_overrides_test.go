//go:build use_logos_storage

package logosstorage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/params"
	. "github.com/status-im/status-go/pkg/services/logosstorage"
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

		// storage.Config fields (nested under NodeConfig)
		"NodeConfig.LogLevel":                       "DEBUG",
		"NodeConfig.LogFormat":                      "json",
		"NodeConfig.MetricsEnabled":                 "true",
		"NodeConfig.MetricsAddress":                 "0.0.0.0",
		"NodeConfig.MetricsPort":                    "9090",
		"NodeConfig.DataDir":                        "/custom/data/dir",
		"NodeConfig.ListenAddrs":                    `["/ip4/0.0.0.0/tcp/4001","/ip4/0.0.0.0/tcp/4002"]`,
		"NodeConfig.Nat":                            "none",
		"NodeConfig.DiscoveryPort":                  "8091",
		"NodeConfig.NetPrivKeyFile":                 "/path/to/key",
		"NodeConfig.BootstrapNodes":                 `["spr:CiUIAhIhA1..","spr:CiUIAhIhA2.."]`,
		"NodeConfig.MaxPeers":                       "200",
		"NodeConfig.NumThreads":                     "4",
		"NodeConfig.AgentString":                    "CustomLogosStorage/1.0",
		"NodeConfig.RepoKind":                       "sqlite",
		"NodeConfig.StorageQuota":                   "50000000000",
		"NodeConfig.BlockTtl":                       "604800",
		"NodeConfig.BlockMaintenanceInterval":       "300",
		"NodeConfig.BlockMaintenanceNumberOfBlocks": "2000",
		"NodeConfig.BlockRetries":                   "5000",
		"NodeConfig.CacheSize":                      "1024",
		"NodeConfig.LogFile":                        "/var/log/logos-storage.log",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Require().NoError(err)

	// Verify params.LogosStorageConfig fields
	s.Equal(true, cfg.Enabled)

	// Verify storage.Config fields
	s.Equal("DEBUG", cfg.NodeConfig.LogLevel)
	s.Equal("json", cfg.NodeConfig.LogFormat)
	s.Equal(true, cfg.NodeConfig.MetricsEnabled)
	s.Equal("0.0.0.0", cfg.NodeConfig.MetricsAddress)
	s.Equal(9090, cfg.NodeConfig.MetricsPort)
	s.Equal("/custom/data/dir", cfg.NodeConfig.DataDir)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/tcp/4002"}, cfg.NodeConfig.ListenAddrs)
	s.Equal("none", cfg.NodeConfig.Nat)
	s.Equal(8091, cfg.NodeConfig.DiscoveryPort)
	s.Equal("/path/to/key", cfg.NodeConfig.NetPrivKeyFile)
	s.Equal([]string{"spr:CiUIAhIhA1..", "spr:CiUIAhIhA2.."}, cfg.NodeConfig.BootstrapNodes)
	s.Equal(200, cfg.NodeConfig.MaxPeers)
	s.Equal(4, cfg.NodeConfig.NumThreads)
	s.Equal("CustomLogosStorage/1.0", cfg.NodeConfig.AgentString)
	s.Equal("sqlite", cfg.NodeConfig.RepoKind)
	s.Equal(50000000000, cfg.NodeConfig.StorageQuota)
	s.Equal("604800", cfg.NodeConfig.BlockTtl)
	s.Equal("300", cfg.NodeConfig.BlockMaintenanceInterval)
	s.Equal(2000, cfg.NodeConfig.BlockMaintenanceNumberOfBlocks)
	s.Equal(5000, cfg.NodeConfig.BlockRetries)
	s.Equal(1024, cfg.NodeConfig.CacheSize)
	s.Equal("/var/log/logos-storage.log", cfg.NodeConfig.LogFile)
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
		"NodeConfig.MetricsPort": "not-a-number",
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
		"NodeConfig.ListenAddrs": `["/ip4/0.0.0.0/tcp/4001"]`,
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001"}, cfg.NodeConfig.ListenAddrs)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_StringSliceCommaSeparated() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"NodeConfig.BootstrapNodes": "node1,node2,node3",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"node1", "node2", "node3"}, cfg.NodeConfig.BootstrapNodes)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_StringSliceEmpty() {
	cfg := &params.LogosStorageConfig{
		NodeConfig: params.LogosStorageNodeConfig{
			ListenAddrs: []string{"existing"},
		},
	}
	overrides := map[string]string{
		"NodeConfig.ListenAddrs": "",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Nil(cfg.NodeConfig.ListenAddrs)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_NestedPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"NodeConfig.DataDir": "/test/path",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal("/test/path", cfg.NodeConfig.DataDir)
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_InvalidNestedPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"NodeConfig.NonExistent.Field": "value",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "unknown field")
	s.Regexp(`\bNonExistent\b`, err.Error())
	s.Contains(err.Error(), "NodeConfig.NonExistent.Field")
}

func (s *LogosStorageConfigOverridesTestSuite) TestApplyLogosStorageConfigOverrides_EmptySegmentInPath() {
	cfg := &params.LogosStorageConfig{}
	overrides := map[string]string{
		"NodeConfig..DataDir": "/test/path",
	}

	err := ApplyLogosStorageConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "invalid empty segment")
}
