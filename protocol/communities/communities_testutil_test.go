package communities_test

import (
	"testing"

	"github.com/codex-storage/codex-go-bindings/codex"

	"github.com/status-im/status-go/protocol/communities"
)

func NewCodexClientTest(t *testing.T) communities.CodexClient {
	client, err := communities.NewCodexClient(codex.Config{
		DataDir:        t.TempDir(),
		LogFormat:      codex.LogFormatNoColors,
		MetricsEnabled: false,
		BlockRetries:   5,
		LogLevel:       "ERROR",
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
