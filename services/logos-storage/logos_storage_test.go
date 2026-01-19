//go:build use_logos_storage
// +build use_logos_storage

package logosstorage_test

import (
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"
)

func TestLogosStorageStart(t *testing.T) {
	node, err := codex.New(codex.Config{
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
