package communities_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/communities"
)

func upload(client communities.CodexClient, t *testing.T, buf *bytes.Buffer) string {
	filename := "hello.txt"
	cid, err := client.Upload(buf, filename)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	if cid == "" {
		t.Fatalf("Expected non-empty CID after upload")
	}

	return cid
}

// CodexClientTestSuite demonstrates testify's suite functionality for CodexClient tests
type CodexClientTestSuite struct {
	suite.Suite
}

// SetupTest runs before each test method
func (suite *CodexClientTestSuite) SetupTest() {

}

// TearDownTest runs after each test method
func (suite *CodexClientTestSuite) TearDownTest() {
	// Nothing to do
}

// TestCodexClientTestSuite runs the test suite
func TestCodexClientTestSuite(t *testing.T) {
	suite.Run(t, new(CodexClientTestSuite))
}

func (suite *CodexClientTestSuite) TestUpload_Success() {
	client := NewCodexClientTest(suite.T())

	// Act
	cid, err := client.Upload(bytes.NewReader([]byte("payload")), "hello.txt")

	// Assert
	require.NoError(suite.T(), err)
	// Codex uses CIDv1 with base58btc encoding (prefix: zDv)
	assert.Equal(suite.T(), "zDvZRwzmBEaJ338xaCHbKbGAJ4X41YyccS6eyorrYBbmPnWuLxCh", cid)
}

func (suite *CodexClientTestSuite) TestDownload_Success() {
	client := NewCodexClientTest(suite.T())

	const payload = "hello from codex"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	var buf bytes.Buffer
	err := client.Download(cid, &buf)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), payload, buf.String())
}

func (suite *CodexClientTestSuite) TestDownloadWithContext_Cancel() {
	// skip test
	suite.T().Skip("Wait for cancellation support PR to be merged in codex-go-bindings")

	client := NewCodexClientTest(suite.T())

	len := 1024 * 1024 * 50
	buf := bytes.NewBuffer(make([]byte, len))
	cid := upload(client, suite.T(), buf)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)

	channelError := make(chan error, 1)
	go func() {
		err := client.DownloadWithContext(ctx, cid, io.Discard)
		channelError <- err
	}()

	cancel()
	err := <-channelError

	require.Error(suite.T(), err)
	// Accept either canceled or deadline exceeded depending on timing
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// net/http may wrap the context error; check error string as a fallback
		es := err.Error()
		if !(es == context.Canceled.Error() || es == context.DeadlineExceeded.Error()) {
			suite.T().Fatalf("expected context cancellation, got: %v", err)
		}
	}
}

func (suite *CodexClientTestSuite) TestHasCid_Success() {
	client := NewCodexClientTest(suite.T())

	const payload = "hello from codex"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	tests := []struct {
		name     string
		cid      string
		wantBool bool
	}{
		{"has CID returns true", cid, true},
		{"has CID returns false", "zDvZRwzmBEaJ338xaCHbKbGAJ4X41YyccS6eyorrYBbmPnWuLxCw", false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got, err := client.HasCid(tt.cid)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.wantBool, got, "HasCid(%q) = %v, want %v", tt.cid, got, tt.wantBool)
		})
	}
}

func (suite *CodexClientTestSuite) TestRemoveCid_Success() {
	client := NewCodexClientTest(suite.T())

	const payload = "hello from codex"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	err := client.RemoveCid(cid)
	require.NoError(suite.T(), err)
}

func (suite *CodexClientTestSuite) TestTriggerDownload() {
	client := NewCodexClientTest(suite.T())

	const payload = "hello from codex"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	ctx := context.Background()
	manifest, err := client.TriggerDownloadWithContext(ctx, cid)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), cid, manifest.Cid)
	assert.Equal(suite.T(), "zDzSvJTf7mGkC3yuiVGco7Qc6s4LA8edye9inT4w2QqHnfbuRvMr", manifest.TreeCid)
	assert.Equal(suite.T(), len(payload), manifest.DatasetSize)
	assert.Equal(suite.T(), "hello.txt", manifest.Filename)
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_Cancellation() {
	suite.T().Skip("Not sure if we are going to have cancellation in trigger download")

	client := NewCodexClientTest(suite.T())

	const testCid = "zDvZRwzmTestCID"

	// Cancel after 50ms (before server responds)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	manifest, err := client.TriggerDownloadWithContext(ctx, testCid)
	require.Error(suite.T(), err, "expected cancellation error")
	assert.Nil(suite.T(), manifest, "expected nil manifest on cancellation")
	// Accept either canceled or deadline exceeded depending on timing
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// net/http may wrap the context error; check error string as a fallback
		es := err.Error()
		if !(es == context.Canceled.Error() || es == context.DeadlineExceeded.Error()) {
			suite.T().Fatalf("expected context cancellation, got: %v", err)
		}
	}
}

func (suite *CodexClientTestSuite) TestLocalDownload() {
	client := NewCodexClientTest(suite.T())

	const payload = "test data for local download"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	var buf bytes.Buffer
	err := client.LocalDownload(cid, &buf)
	require.NoError(suite.T(), err, "LocalDownload failed")
	assert.Equal(suite.T(), payload, buf.String(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Success() {
	client := NewCodexClientTest(suite.T())

	const payload = "test data for local download with context"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	ctx := context.Background()
	var buf bytes.Buffer
	err := client.LocalDownloadWithContext(ctx, cid, &buf)
	require.NoError(suite.T(), err, "LocalDownloadWithContext failed")
	assert.Equal(suite.T(), payload, buf.String(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Cancellation() {
	suite.T().Skip("Wait for cancellation support PR to be merged in codex-go-bindings")

	client := NewCodexClientTest(suite.T())

	// Create a context with a very short timeout
	len := 1024 * 1024 * 50
	buf := bytes.NewBuffer(make([]byte, len))
	cid := upload(client, suite.T(), buf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)

	channelError := make(chan error, 1)
	go func() {
		err := client.LocalDownloadWithContext(ctx, cid, io.Discard)
		channelError <- err
	}()

	cancel()
	err := <-channelError

	require.Error(suite.T(), err, "Expected context cancellation error")
	// Accept either canceled or deadline exceeded depending on timing
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// net/http may wrap the context error; check error string as a fallback
		es := err.Error()
		if !(es == context.Canceled.Error() || es == context.DeadlineExceeded.Error()) {
			suite.T().Fatalf("expected context cancellation, got: %v", err)
		}
	}
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_Success() {
	client := NewCodexClientTest(suite.T())

	const payload = "hello from codex"
	cid := upload(client, suite.T(), bytes.NewBuffer([]byte(payload)))

	ctx := context.Background()
	manifest, err := client.FetchManifestWithContext(ctx, cid)
	require.NoError(suite.T(), err, "Expected no error")
	require.NotNil(suite.T(), manifest, "Expected manifest, got nil")

	assert.Equal(suite.T(), cid, manifest.Cid)
	assert.Equal(suite.T(), "zDzSvJTf7mGkC3yuiVGco7Qc6s4LA8edye9inT4w2QqHnfbuRvMr", manifest.TreeCid)
	assert.Equal(suite.T(), len(payload), manifest.DatasetSize)
	assert.Equal(suite.T(), 65536, manifest.BlockSize)
	assert.True(suite.T(), !manifest.Protected, "Expected Protected to be false")
	assert.Equal(suite.T(), "hello.txt", manifest.Filename)
	assert.Equal(suite.T(), "text/plain", manifest.Mimetype)
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_Cancellation() {
	suite.T().Skip("Not sure if we are going to have cancellation in fetch manifest")

	client := NewCodexClientTest(suite.T())

	testCid := "zDvZRwzmTestCID"

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	manifest, err := client.FetchManifestWithContext(ctx, testCid)
	require.Error(suite.T(), err, "Expected context cancellation error")
	assert.Nil(suite.T(), manifest, "Expected nil manifest on cancellation")

	// Accept either canceled or deadline exceeded depending on timing
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// net/http may wrap the context error; check error string as a fallback
		es := err.Error()
		if !(es == context.Canceled.Error() || es == context.DeadlineExceeded.Error()) {
			suite.T().Fatalf("expected context cancellation, got: %v", err)
		}
	}

	buf := bytes.NewBuffer([]byte("Hello World!"))
	if buf.Len() != manifest.DatasetSize {
		suite.T().Errorf("expected size %d, got %d", buf.Len(), manifest.DatasetSize)
	}

	defaultBlockSize := 1024 * 64
	if manifest.BlockSize != defaultBlockSize {
		suite.T().Errorf("expected block size %d, got %d", defaultBlockSize, manifest.BlockSize)
	}

	if manifest.Filename != "test.txt" {
		suite.T().Errorf("expected filename %q, got %q", "test.txt", manifest.Filename)
	}

	if manifest.Protected {
		suite.T().Errorf("expected protected to be false, got true")
	}

	if manifest.Mimetype != "text/plain" {
		suite.T().Errorf("expected mimetype %q, got %q", "text/plain", manifest.Mimetype)
	}

	if manifest.TreeCid == "" {
		suite.T().Errorf("expected non-empty TreeCid")
	}
}
