package logosstorage_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/codex-storage/codex-go-bindings/codex"

	logosstorage "github.com/status-im/status-go/services/logos-storage"
)

type CodexClientTestSuite struct {
	suite.Suite
	client       logosstorage.CodexClientInterface
	uploadedCIDs []string // Track uploaded CIDs for cleanup
}

// TestCodexClientTestSuite runs the test suite
func TestCodexClientTestSuite(t *testing.T) {
	suite.Run(t, new(CodexClientTestSuite))
}

func (suite *CodexClientTestSuite) UploadRandomDataToCodex(size int) (string, []byte) {
	// Generate random payload to ensure proper round-trip verification
	payload := make([]byte, size)
	_, err := rand.Read(payload)
	require.NoError(suite.T(), err, "failed to generate random payload")
	suite.T().Logf("Generated payload (first 32 bytes hex): %s", hex.EncodeToString(payload[:32]))

	cid, err := suite.client.Upload(bytes.NewReader(payload), "payload.bin")
	require.NoError(suite.T(), err, "upload failed")

	suite.uploadedCIDs = append(suite.uploadedCIDs, cid)
	return cid, payload
}

func safeCancel(ctx context.Context, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
	default:
		cancel()
	}
}

// SetupTest runs before each test method
func (suite *CodexClientTestSuite) SetupTest() {
	suite.client = NewCodexClientTest(suite.T())
	suite.uploadedCIDs = []string{}
}

// TearDownTest runs after each test method
func (suite *CodexClientTestSuite) TearDownTest() {
	// Clean up all uploaded CIDs
	for _, cid := range suite.uploadedCIDs {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		} else {
			suite.T().Logf("Successfully removed CID: %s", cid)
		}
	}
}

func (suite *CodexClientTestSuite) TestUpload_Success() {
	cid, err := suite.client.Upload(bytes.NewReader([]byte("payload")), "hello.txt")

	require.NoError(suite.T(), err)
	// Codex uses CIDv1 with base58btc encoding (prefix: zDv)
	assert.Equal(suite.T(), "zDvZRwzmBEaJ338xaCHbKbGAJ4X41YyccS6eyorrYBbmPnWuLxCh", cid)
}

func (suite *CodexClientTestSuite) TestDownload_Success() {
	cid, payload := suite.UploadRandomDataToCodex(1024) // 1KB payload

	var buf bytes.Buffer
	err := suite.client.Download(cid, &buf)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), payload, buf.Bytes())
}

func (suite *CodexClientTestSuite) TestDownloadWithContext_Cancel() {
	// skip test - flaky
	// suite.T().Skip("Flaky test - needs investigation")

	cid, _ := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx, cancel := context.WithCancel(context.Background())
	defer safeCancel(ctx, cancel)

	channelError := make(chan error, 1)
	go func() {
		err := suite.client.DownloadWithContext(ctx, cid, io.Discard)
		channelError <- err
	}()

	cancel()

	select {
	case err := <-channelError:
		require.Error(suite.T(), err)
		assert.ErrorIs(suite.T(), err, context.Canceled)
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Test timed out - download didn't respond to cancellation")
	}
}

func (suite *CodexClientTestSuite) TestDownloadWithContext_ContextAlreadyCancelled() {
	// skip test - flaky
	// suite.T().Skip("Flaky test - needs investigation")

	cid, _ := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := suite.client.DownloadWithContext(ctx, cid, io.Discard)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, context.Canceled)
}

func (suite *CodexClientTestSuite) TestHasCid_Success() {
	cid, _ := suite.UploadRandomDataToCodex(1024)

	// yes, we could compute here a fresh CID by using any valid CID will do
	nonExistingCid := "zDvZRwzmBEaJ338xaCHbKbGAJ4X41YyccS6eyorrYBbmPnWuLxCw"

	tests := []struct {
		name     string
		cid      string
		wantBool bool
	}{
		{"has CID returns true", cid, true},
		{"has CID returns false", nonExistingCid, false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got, err := suite.client.HasCid(tt.cid)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.wantBool, got, "HasCid(%q) = %v, want %v", tt.cid, got, tt.wantBool)
		})
	}
}

func (suite *CodexClientTestSuite) TestRemoveCid_Success() {
	cid, _ := suite.UploadRandomDataToCodex(1024)

	err := suite.client.RemoveCid(cid)
	require.NoError(suite.T(), err)
}

func (suite *CodexClientTestSuite) TestTriggerDownload() {
	cid, payload := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx := context.Background()
	manifest, err := suite.client.TriggerDownloadWithContext(ctx, cid)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), cid, manifest.Cid)
	assert.NotEmpty(suite.T(), manifest.TreeCid)
	assert.Equal(suite.T(), len(payload), manifest.DatasetSize)
	assert.Equal(suite.T(), "payload.bin", manifest.Filename)
	assert.Equal(suite.T(), "application/octet-stream", manifest.Mimetype)

	// Poll HasCid to verify async download completion
	downloadComplete := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			hasCid, err := suite.client.HasCid(cid)
			if err != nil {
				suite.T().Logf("HasCid check failed: %v", err)
				continue
			}
			if hasCid {
				suite.T().Logf("CID is now available locally")
				downloadComplete <- true
				return
			}
		}
	}()

	// Wait for download completion or timeout
	select {
	case <-downloadComplete:
		// Download completed successfully
	case <-time.After(10 * time.Second):
		suite.T().Fatalf("Timeout waiting for CID to be available locally after 10 seconds")
	}

	// Verify we can download the actual content from local storage
	var downloadBuf bytes.Buffer
	err = suite.client.LocalDownloadWithContext(ctx, cid, &downloadBuf)
	require.NoError(suite.T(), err, "LocalDownload after trigger download failed")
	assert.Equal(suite.T(), payload, downloadBuf.Bytes(), "Downloaded data does not match uploaded data")
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_Cancellation() {
	suite.T().Skip("Not sure if we are going to have cancellation in trigger download")

	cid, _ := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx, cancel := context.WithCancel(context.Background())
	defer safeCancel(ctx, cancel)

	channelError := make(chan error, 1)
	var manifest codex.Manifest
	go func() {
		var err error
		manifest, err = suite.client.TriggerDownloadWithContext(ctx, cid)
		channelError <- err
	}()

	// Give the goroutine time to start the blocking operation
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-channelError:
		require.Error(suite.T(), err)
		assert.ErrorIs(suite.T(), err, context.Canceled)
		assert.Nil(suite.T(), manifest, "expected nil manifest on cancellation")
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Test timed out - trigger download didn't respond to cancellation")
	}
}

func (suite *CodexClientTestSuite) TestLocalDownload() {
	cid, payload := suite.UploadRandomDataToCodex(1024)

	var buf bytes.Buffer
	err := suite.client.LocalDownload(cid, &buf)
	require.NoError(suite.T(), err, "LocalDownload failed")
	assert.Equal(suite.T(), payload, buf.Bytes(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Success() {
	cid, payload := suite.UploadRandomDataToCodex(1024)

	ctx := context.Background()
	var buf bytes.Buffer
	err := suite.client.LocalDownloadWithContext(ctx, cid, &buf)
	require.NoError(suite.T(), err, "LocalDownloadWithContext failed")
	assert.Equal(suite.T(), payload, buf.Bytes(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Cancellation() {
	// Create a context with a very short timeout
	cid, _ := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx, cancel := context.WithCancel(context.Background())
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer safeCancel(ctx, cancel)

	channelError := make(chan error, 1)
	go func() {
		err := suite.client.LocalDownloadWithContext(ctx, cid, io.Discard)
		channelError <- err
	}()

	cancel()

	select {
	case err := <-channelError:
		require.Error(suite.T(), err)
		assert.ErrorIs(suite.T(), err, context.Canceled)
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Test timed out - download didn't respond to cancellation")
	}
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_Success() {
	cid, payload := suite.UploadRandomDataToCodex(1024)

	ctx := context.Background()
	manifest, err := suite.client.FetchManifestWithContext(ctx, cid)

	require.NoError(suite.T(), err, "Expected no error")
	require.NotNil(suite.T(), manifest, "Expected manifest, got nil")

	assert.Equal(suite.T(), cid, manifest.Cid)
	assert.NotEmpty(suite.T(), manifest.TreeCid)
	assert.Equal(suite.T(), len(payload), manifest.DatasetSize)
	assert.Equal(suite.T(), 65536, manifest.BlockSize)
	assert.True(suite.T(), !manifest.Protected, "Expected Protected to be false")
	assert.Equal(suite.T(), "payload.bin", manifest.Filename)
	assert.Equal(suite.T(), "application/octet-stream", manifest.Mimetype)
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_NonExistentCID() {
	ctx := context.Background()
	nonExistentCID := "zDvZRwzmNonExistentCID123456789"

	_, err := suite.client.FetchManifestWithContext(ctx, nonExistentCID)
	assert.Error(suite.T(), err, "Expected error when fetching manifest for non-existent CID")
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_Cancellation() {
	suite.T().Skip("Not sure if we are going to have cancellation in fetch manifest")

	cid, _ := suite.UploadRandomDataToCodex(50 * 1024 * 1024)

	ctx, cancel := context.WithCancel(context.Background())
	defer safeCancel(ctx, cancel)

	channelError := make(chan error, 1)
	var manifest codex.Manifest
	go func() {
		var err error
		manifest, err = suite.client.FetchManifestWithContext(ctx, cid)
		channelError <- err
	}()

	cancel()

	select {
	case err := <-channelError:
		require.Error(suite.T(), err)
		assert.ErrorIs(suite.T(), err, context.Canceled)
		assert.Nil(suite.T(), manifest, "expected nil manifest on cancellation")
	case <-time.After(5 * time.Second):
		suite.T().Fatal("Test timed out - fetch manifest didn't respond to cancellation")
	}
}
