package codex

import (
	"bytes"
	"testing"
)

type codexNodeTestOption func(*codexNodeTestOptions)

type codexNodeTestOptions struct {
	noStart      bool
	blockRetries int
}

func withNoStart() codexNodeTestOption {
	return func(o *codexNodeTestOptions) { o.noStart = true }
}

func withBlockRetries(n int) codexNodeTestOption {
	return func(o *codexNodeTestOptions) { o.blockRetries = n }
}

func newCodexNode(t *testing.T, opts ...codexNodeTestOption) *CodexNode {
	o := codexNodeTestOptions{
		blockRetries: 3000,
	}
	for _, opt := range opts {
		opt(&o)
	}

	node, err := New(Config{
		DataDir:        t.TempDir(),
		LogFormat:      LogFormatNoColors,
		MetricsEnabled: false,
		BlockRetries:   o.blockRetries,
	})
	if err != nil {
		t.Fatalf("Failed to create Codex node: %v", err)
	}

	if !o.noStart {
		err = node.Start()
		if err != nil {
			t.Fatalf("Failed to start Codex node: %v", err)
		}
	}

	t.Cleanup(func() {
		if !o.noStart {
			if err := node.Stop(); err != nil {
				t.Logf("cleanup codex: %v", err)
			}
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
	cid, err := codex.UploadReader(UploadOptions{Filepath: "hello.txt"}, buf)
	if err != nil {
		t.Fatalf("Error happened during upload: %v\n", err)
	}

	return cid, len
}
