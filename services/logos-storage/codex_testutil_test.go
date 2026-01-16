package logosstorage_test

import (
	"path/filepath"
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"

	"github.com/status-im/status-go/params"
	logosstorage "github.com/status-im/status-go/services/logos-storage"
)

func NewCodexClientTest(t *testing.T) logosstorage.CodexClientInterface {
	client, err := logosstorage.NewCodexClient(params.CodexConfig{
		Enabled: true,
		CodexNodeConfig: codex.Config{
			DataDir:        filepath.Join(t.TempDir(), "codex", "codexdata"),
			LogFormat:      codex.LogFormatNoColors,
			MetricsEnabled: false,
			LogLevel:       "ERROR",
			Nat:            "none",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create Codex client: %v", err)
	}

	t.Cleanup(func() {
		if err := client.Stop(); err != nil {
			t.Fatalf("Failed to stop codex: %v", err)
		}

		if err := client.Destroy(); err != nil {
			t.Fatalf("Failed to destroy codex: %v", err)
		}
	})

	if err = client.Start(); err != nil {
		t.Fatalf("Failed to start Codex node: %v", err)
	}

	return client
}
