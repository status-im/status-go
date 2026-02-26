//go:build use_logos_storage && !lint
// +build use_logos_storage,!lint

package logosstorage_test

import (
	"testing"

	"github.com/logos-storage/logos-storage-go-bindings/storage"
)

func TestLogosStorageStart(t *testing.T) {
	node, err := storage.New(storage.Config{
		BlockRetries: 5,
		DataDir:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Failed to create LogosStorage node: %v", err)
	}

	// Test the node's functionality
	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start LogosStorage node: %v", err)
	}

	if err := node.Stop(); err != nil {
		t.Fatalf("Failed to stop LogosStorage node: %v", err)
	}

	if err := node.Destroy(); err != nil {
		t.Fatalf("Failed to destroy LogosStorage node: %v", err)
	}
}
