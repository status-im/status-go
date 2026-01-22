package logosstorage_test

import (
	"path/filepath"
	"testing"

	"github.com/logos-storage/logos-storage-go-bindings/storage"

	"github.com/status-im/status-go/params"
	logosstorage "github.com/status-im/status-go/services/logos-storage"
)

func NewLogosStorageClientTest(t *testing.T) logosstorage.LogosStorageClientInterface {
	client, err := logosstorage.NewLogosStorageClient(params.LogosStorageConfig{
		Enabled: true,
		LogosStorageNodeConfig: storage.Config{
			DataDir:        filepath.Join(t.TempDir(), "logos-storage", "data"),
			LogFormat:      storage.LogFormatNoColors,
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
