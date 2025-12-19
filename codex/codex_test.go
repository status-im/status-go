//go:build use_codex
// +build use_codex

package codexstatus

import (
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"
)

func TestCodexStart(t *testing.T) {
	node, err := codex.New(codex.Config{
		BlockRetries: 5,
		DataDir:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Failed to create codex node: %v", err)
	}

	// Test the node's functionality
	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start codex node: %v", err)
	}

	if err := node.Stop(); err != nil {
		t.Fatalf("Failed to stop codex node: %v", err)
	}

	if err := node.Destroy(); err != nil {
		t.Fatalf("Failed to destroy codex node: %v", err)
	}
}
