package codex

import (
	"bytes"
	"context"
	"testing"
)

func defaultConfigHelper(t *testing.T) Config {
	t.Helper()

	return Config{
		DataDir:        t.TempDir(),
		LogFormat:      LogFormatNoColors,
		MetricsEnabled: false,
		BlockRetries:   3000,
		Nat:            "none",
	}
}

func newCodexNode(t *testing.T, opts ...Config) *CodexNode {
	config := defaultConfigHelper(t)

	if len(opts) > 0 {
		c := opts[0]

		if c.BlockRetries > 0 {
			config.BlockRetries = c.BlockRetries
		}

		if c.LogLevel != "" {
			config.LogLevel = c.LogLevel
		}

		if c.LogFile != "" {
			config.LogFile = c.LogFile
		}

		if len(c.BootstrapNodes) != 0 {
			config.BootstrapNodes = c.BootstrapNodes
		}

		if c.DiscoveryPort != 0 {
			config.DiscoveryPort = c.DiscoveryPort
		}

		if c.StorageQuota != 0 {
			config.StorageQuota = c.StorageQuota
		}
	}

	node, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Codex node: %v", err)
	}

	err = node.Start()
	if err != nil {
		t.Fatalf("Failed to start Codex node: %v", err)
	}

	t.Cleanup(func() {
		if err := node.Stop(); err != nil {
			t.Logf("cleanup codex: %v", err)
		}

		if err := node.Destroy(); err != nil {
			t.Logf("cleanup codex: %v", err)
		}
	})

	return node
}

func uploadHelper(t *testing.T, codex *CodexNode) (string, int) {
	t.Helper()

	buf := bytes.NewBuffer([]byte("Hello World!"))
	len := buf.Len()
	cid, err := codex.UploadReader(context.Background(), UploadOptions{Filepath: "hello.txt"}, buf)
	if err != nil {
		t.Fatalf("Error happened during upload: %v\n", err)
	}

	return cid, len
}

func uploadBigFileHelper(t *testing.T, codex *CodexNode) (string, int) {
	t.Helper()

	len := 1024 * 1024 * 50
	buf := bytes.NewBuffer(make([]byte, len))

	cid, err := codex.UploadReader(context.Background(), UploadOptions{Filepath: "hello.txt"}, buf)
	if err != nil {
		t.Fatalf("Error happened during upload: %v\n", err)
	}

	return cid, len
}
