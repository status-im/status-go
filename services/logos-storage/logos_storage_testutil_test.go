package logosstorage_test

import (
	"path/filepath"
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"

	"github.com/status-im/status-go/params"
	logosstorage "github.com/status-im/status-go/services/logos-storage"
)

func NewLogosStorageClientTest(t *testing.T) logosstorage.LogosStorageClientInterface {
	client, err := logosstorage.NewLogosStorageClient(params.LogosStorageConfig{
		Enabled: true,
		LogosStorageNodeConfig: codex.Config{
			DataDir:        filepath.Join(t.TempDir(), "logos-storage", "data"),
			LogFormat:      codex.LogFormatNoColors,
			MetricsEnabled: false,
			LogLevel:       "ERROR",
			Nat:            "none",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create LogosStorage client: %v", err)
	}

	t.Cleanup(func() {
		if err := client.Stop(); err != nil {
			t.Fatalf("Failed to stop LogosStorage: %v", err)
		}

		if err := client.Destroy(); err != nil {
			t.Fatalf("Failed to destroy LogosStorage: %v", err)
		}
	})

	if err = client.Start(); err != nil {
		t.Fatalf("Failed to start LogosStorage node: %v", err)
	}

	return client
}
