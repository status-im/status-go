//go:build use_logos_storage
// +build use_logos_storage

package logosstorage_test

import (
	"path/filepath"
	"testing"

	"github.com/status-im/status-go/params"
	logosstorage "github.com/status-im/status-go/services/logosstorage"
)

func NewLogosStorageClientTest(t *testing.T) logosstorage.LogosStorageClientInterface {
	client, err := logosstorage.NewLogosStorageClient(params.LogosStorageConfig{
		Enabled: true,
		NodeConfig: params.LogosStorageNodeConfig{
			DataDir:        filepath.Join(t.TempDir(), "logos-storage", "data"),
			LogFormat:      "nocolors",
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
