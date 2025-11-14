package communities

import (
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/params"
)

type CodexConfigOverridesTestSuite struct {
	suite.Suite
}

func TestCodexConfigOverridesTestSuite(t *testing.T) {
	suite.Run(t, new(CodexConfigOverridesTestSuite))
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_AllFields() {
	cfg := &params.CodexConfig{}

	overrides := map[string]string{
		// params.CodexConfig fields
		"Enabled":               "true",
		"HistoryArchiveDataDir": "/custom/archive/path",

		// codex.Config fields (nested under CodexNodeConfig)
		"CodexNodeConfig.LogLevel":                       "DEBUG",
		"CodexNodeConfig.LogFormat":                      "json",
		"CodexNodeConfig.MetricsEnabled":                 "true",
		"CodexNodeConfig.MetricsAddress":                 "0.0.0.0",
		"CodexNodeConfig.MetricsPort":                    "9090",
		"CodexNodeConfig.DataDir":                        "/custom/data/dir",
		"CodexNodeConfig.ListenAddrs":                    `["/ip4/0.0.0.0/tcp/4001","/ip4/0.0.0.0/tcp/4002"]`,
		"CodexNodeConfig.Nat":                            "none",
		"CodexNodeConfig.DiscoveryPort":                  "8091",
		"CodexNodeConfig.NetPrivKeyFile":                 "/path/to/key",
		"CodexNodeConfig.BootstrapNodes":                 `["spr:CiUIAhIhA1..","spr:CiUIAhIhA2.."]`,
		"CodexNodeConfig.MaxPeers":                       "200",
		"CodexNodeConfig.NumThreads":                     "4",
		"CodexNodeConfig.AgentString":                    "CustomCodex/1.0",
		"CodexNodeConfig.RepoKind":                       "sqlite",
		"CodexNodeConfig.StorageQuota":                   "50000000000",
		"CodexNodeConfig.BlockTtl":                       "604800",
		"CodexNodeConfig.BlockMaintenanceInterval":       "300",
		"CodexNodeConfig.BlockMaintenanceNumberOfBlocks": "2000",
		"CodexNodeConfig.BlockRetries":                   "5000",
		"CodexNodeConfig.CacheSize":                      "1024",
		"CodexNodeConfig.LogFile":                        "/var/log/codex.log",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Require().NoError(err)

	// Verify params.CodexConfig fields
	s.Equal(true, cfg.Enabled)
	s.Equal("/custom/archive/path", cfg.HistoryArchiveDataDir)

	// Verify codex.Config fields
	s.Equal("DEBUG", cfg.CodexNodeConfig.LogLevel)
	s.Equal(codex.LogFormat("json"), cfg.CodexNodeConfig.LogFormat)
	s.Equal(true, cfg.CodexNodeConfig.MetricsEnabled)
	s.Equal("0.0.0.0", cfg.CodexNodeConfig.MetricsAddress)
	s.Equal(9090, cfg.CodexNodeConfig.MetricsPort)
	s.Equal("/custom/data/dir", cfg.CodexNodeConfig.DataDir)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/tcp/4002"}, cfg.CodexNodeConfig.ListenAddrs)
	s.Equal("none", cfg.CodexNodeConfig.Nat)
	s.Equal(8091, cfg.CodexNodeConfig.DiscoveryPort)
	s.Equal("/path/to/key", cfg.CodexNodeConfig.NetPrivKeyFile)
	s.Equal([]string{"spr:CiUIAhIhA1..", "spr:CiUIAhIhA2.."}, cfg.CodexNodeConfig.BootstrapNodes)
	s.Equal(200, cfg.CodexNodeConfig.MaxPeers)
	s.Equal(4, cfg.CodexNodeConfig.NumThreads)
	s.Equal("CustomCodex/1.0", cfg.CodexNodeConfig.AgentString)
	s.Equal(codex.RepoKind("sqlite"), cfg.CodexNodeConfig.RepoKind)
	s.Equal(50000000000, cfg.CodexNodeConfig.StorageQuota)
	s.Equal("604800", cfg.CodexNodeConfig.BlockTtl)
	s.Equal("300", cfg.CodexNodeConfig.BlockMaintenanceInterval)
	s.Equal(2000, cfg.CodexNodeConfig.BlockMaintenanceNumberOfBlocks)
	s.Equal(5000, cfg.CodexNodeConfig.BlockRetries)
	s.Equal(1024, cfg.CodexNodeConfig.CacheSize)
	s.Equal("/var/log/codex.log", cfg.CodexNodeConfig.LogFile)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_NilConfig() {
	err := ApplyCodexConfigOverrides(nil, map[string]string{"Enabled": "true"})
	s.NoError(err, "should handle nil config gracefully")
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_EmptyOverrides() {
	cfg := &params.CodexConfig{}
	err := ApplyCodexConfigOverrides(cfg, map[string]string{})
	s.NoError(err, "should handle empty overrides gracefully")
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_InvalidFieldName() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"NonExistentField": "value",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "unknown field")
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_InvalidBoolValue() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"Enabled": "not-a-bool",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Error(err)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_InvalidIntValue() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig.MetricsPort": "not-a-number",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Error(err)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_EmptyKey() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"":        "should-be-ignored",
		"Enabled": "true",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal(true, cfg.Enabled)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_WhitespaceKey() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"   ":     "should-be-ignored",
		"Enabled": "true",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal(true, cfg.Enabled)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_StringSliceJSON() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig.ListenAddrs": `["/ip4/0.0.0.0/tcp/4001"]`,
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"/ip4/0.0.0.0/tcp/4001"}, cfg.CodexNodeConfig.ListenAddrs)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_StringSliceCommaSeparated() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig.BootstrapNodes": "node1,node2,node3",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal([]string{"node1", "node2", "node3"}, cfg.CodexNodeConfig.BootstrapNodes)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_StringSliceEmpty() {
	cfg := &params.CodexConfig{
		CodexNodeConfig: codex.Config{
			ListenAddrs: []string{"existing"},
		},
	}
	overrides := map[string]string{
		"CodexNodeConfig.ListenAddrs": "",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Nil(cfg.CodexNodeConfig.ListenAddrs)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_NestedPath() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig.DataDir": "/test/path",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.NoError(err)
	s.Equal("/test/path", cfg.CodexNodeConfig.DataDir)
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_InvalidNestedPath() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig.NonExistent.Field": "value",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "unknown field")
	s.Regexp(`\bNonExistent\b`, err.Error())
	s.Contains(err.Error(), "CodexNodeConfig.NonExistent.Field")
}

func (s *CodexConfigOverridesTestSuite) TestApplyCodexConfigOverrides_EmptySegmentInPath() {
	cfg := &params.CodexConfig{}
	overrides := map[string]string{
		"CodexNodeConfig..DataDir": "/test/path",
	}

	err := ApplyCodexConfigOverrides(cfg, overrides)
	s.Error(err)
	s.Contains(err.Error(), "invalid empty segment")
}
