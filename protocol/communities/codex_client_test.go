package communities_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/communities"
)

// CodexClientTestSuite demonstrates testify's suite functionality for CodexClient tests
type CodexClientTestSuite struct {
	suite.Suite
	client *communities.CodexClient
	server *httptest.Server
}

// SetupTest runs before each test method
func (suite *CodexClientTestSuite) SetupTest() {
	suite.client = communities.NewCodexClient("localhost", "8080")
}

// TearDownTest runs after each test method
func (suite *CodexClientTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
		suite.server = nil
	}
}

// TestCodexClientTestSuite runs the test suite
func TestCodexClientTestSuite(t *testing.T) {
	suite.Run(t, new(CodexClientTestSuite))
}

func (suite *CodexClientTestSuite) TestUpload_Success() {
	// Arrange a fake Codex server that validates headers and returns a CID
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/codex/v1/data" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if cd := r.Header.Get("Content-Disposition"); cd != "filename=\"hello.txt\"" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, _ = io.ReadAll(r.Body) // consume body
		_ = r.Body.Close()

		w.WriteHeader(http.StatusOK)
		// Codex returns CIDv1 base58btc
		// prefix: zDv
		//   - z = multibase prefix for base58btc
		//	 - Dv = CIDv1 prefix for raw codex
		// we add a newline to simulate real response
		_, _ = w.Write([]byte("zDvZRwzmTestCID123\n"))
	}))

	suite.client.BaseURL = suite.server.URL

	// Act
	cid, err := suite.client.Upload(bytes.NewReader([]byte("payload")), "hello.txt")

	// Assert
	require.NoError(suite.T(), err)
	// Codex uses CIDv1 with base58btc encoding (prefix: zDv)
	assert.Equal(suite.T(), "zDvZRwzmTestCID123", cid)
}

func (suite *CodexClientTestSuite) TestDownload_Success() {
	const wantCID = "zDvZRwzm"
	const payload = "hello from codex"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/codex/v1/data/"+wantCID+"/network/stream" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))

	suite.client.BaseURL = suite.server.URL

	var buf bytes.Buffer
	err := suite.client.Download(wantCID, &buf)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), payload, buf.String())
}

func (suite *CodexClientTestSuite) TestDownloadWithContext_Cancel() {
	const cid = "zDvZRwzm"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/codex/v1/data/"+cid+"/network/stream" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		flusher, _ := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		// Stream data slowly so the request can be canceled
		for i := 0; i < 1000; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			if _, err := w.Write([]byte("x")); err != nil {
				// Client likely went away; stop writing
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}))

	suite.client.BaseURL = suite.server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := suite.client.DownloadWithContext(ctx, cid, io.Discard)
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
	tests := []struct {
		name     string
		cid      string
		hasIt    bool
		wantBool bool
	}{
		{"has CID returns true", "zDvZRwzmTestCID", true, true},
		{"has CID returns false", "zDvZRwzmTestCID", false, false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/codex/v1/data/"+tt.cid+"/exists" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// Return JSON: {"<cid>": <bool>}
				fmt.Fprintf(w, `{"%s": %t}`, tt.cid, tt.hasIt)
			}))

			suite.client.BaseURL = suite.server.URL

			got, err := suite.client.HasCid(tt.cid)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.wantBool, got, "HasCid(%q) = %v, want %v", tt.cid, got, tt.wantBool)
		})
	}
}

func (suite *CodexClientTestSuite) TestHasCid_RequestError() {
	// Create a server and immediately close it to trigger connection error
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	suite.server.Close() // Close immediately so connection fails

	suite.client.BaseURL = suite.server.URL // Use the closed server's URL

	got, err := suite.client.HasCid("zDvZRwzmTestCID")
	require.Error(suite.T(), err)
	assert.False(suite.T(), got, "expected false on error")
}

func (suite *CodexClientTestSuite) TestHasCid_CidMismatch() {
	const requestCid = "zDvZRwzmRequestCID"
	const responseCid = "zDvZRwzmDifferentCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a different CID in the response
		fmt.Fprintf(w, `{"%s": true}`, responseCid)
	}))

	suite.client.BaseURL = suite.server.URL

	got, err := suite.client.HasCid(requestCid)
	require.Error(suite.T(), err, "expected error for CID mismatch")
	assert.False(suite.T(), got, "expected false on CID mismatch")
	// Check error message mentions the missing/mismatched CID
	assert.Contains(suite.T(), err.Error(), requestCid, "error should mention request CID")
}

func (suite *CodexClientTestSuite) TestRemoveCid_Success() {
	const testCid = "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/codex/v1/data/"+testCid {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// DELETE should return 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}))

	suite.client.BaseURL = suite.server.URL

	err := suite.client.RemoveCid(testCid)
	require.NoError(suite.T(), err)
}

func (suite *CodexClientTestSuite) TestRemoveCid_Error() {
	const testCid = "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error status
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))

	suite.client.BaseURL = suite.server.URL

	err := suite.client.RemoveCid(testCid)
	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "500", "error should mention status 500")
}

func (suite *CodexClientTestSuite) TestTriggerDownload() {
	const testCid = "zDvZRwzmTestCID"
	const expectedManifest = `{
		"cid": "zDvZRwzmTestCID",
		"manifest": {
			"treeCid": "zDvZRwzmTreeCID",
			"datasetSize": 1024,
			"blockSize": 65536,
			"protected": false,
			"filename": "test-file.bin",
			"mimetype": "application/octet-stream"
		}
	}`

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/codex/v1/data/"+testCid+"/network" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedManifest))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.TriggerDownloadWithContext(ctx, testCid)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testCid, manifest.CID)
	assert.Equal(suite.T(), "zDvZRwzmTreeCID", manifest.Manifest.TreeCid)
	assert.Equal(suite.T(), int64(1024), manifest.Manifest.DatasetSize)
	assert.Equal(suite.T(), "test-file.bin", manifest.Manifest.Filename)
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_RequestError() {
	// Create a server and immediately close it to trigger connection error
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	suite.server.Close()

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.TriggerDownloadWithContext(ctx, "zDvZRwzmRigWseNB7WqmudkKAPgZmrDCE9u5cY4KvCqhRo9Ki")
	require.Error(suite.T(), err)
	assert.Nil(suite.T(), manifest, "expected nil manifest on error")
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_JSONParseError() {
	const testCid = "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return invalid JSON
		w.Write([]byte(`{"invalid": json}`))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.TriggerDownloadWithContext(ctx, testCid)
	require.Error(suite.T(), err, "expected JSON parse error")
	assert.Nil(suite.T(), manifest, "expected nil manifest on parse error")
	assert.Contains(suite.T(), err.Error(), "failed to parse download manifest", "error should mention parse failure")
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_HTTPError() {
	const testCid = "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("CID not found"))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.TriggerDownloadWithContext(ctx, testCid)
	require.Error(suite.T(), err, "expected error for 404 status")
	assert.Nil(suite.T(), manifest, "expected nil manifest on HTTP error")
	assert.Contains(suite.T(), err.Error(), "404", "error should mention status 404")
}

func (suite *CodexClientTestSuite) TestTriggerDownloadWithContext_Cancellation() {
	const testCid = "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response to allow cancellation
		select {
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"cid": "test"}`))
		}
	}))

	suite.client.BaseURL = suite.server.URL

	// Cancel after 50ms (before server responds)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	manifest, err := suite.client.TriggerDownloadWithContext(ctx, testCid)
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
	testData := []byte("test data for local download")
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(suite.T(), "GET", r.Method, "Expected GET request")
		expectedPath := "/api/codex/v1/data/" + testCid
		assert.Equal(suite.T(), expectedPath, r.URL.Path, "Expected correct path")

		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))

	suite.client.BaseURL = suite.server.URL

	var buf bytes.Buffer
	err := suite.client.LocalDownload(testCid, &buf)
	require.NoError(suite.T(), err, "LocalDownload failed")
	assert.Equal(suite.T(), testData, buf.Bytes(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Success() {
	testData := []byte("test data for local download with context")
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(suite.T(), "GET", r.Method, "Expected GET request")
		expectedPath := "/api/codex/v1/data/" + testCid
		assert.Equal(suite.T(), expectedPath, r.URL.Path, "Expected correct path")

		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	var buf bytes.Buffer
	err := suite.client.LocalDownloadWithContext(ctx, testCid, &buf)
	require.NoError(suite.T(), err, "LocalDownloadWithContext failed")
	assert.Equal(suite.T(), testData, buf.Bytes(), "Downloaded data mismatch")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_RequestError() {
	// Create a server and immediately close it to trigger connection error
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	suite.server.Close()

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	var buf bytes.Buffer
	err := suite.client.LocalDownloadWithContext(ctx, "zDvZRwzmTestCID", &buf)
	require.Error(suite.T(), err, "Expected error due to closed server")
	assert.Contains(suite.T(), err.Error(), "failed to download from codex")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_HTTPError() {
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("CID not found in local storage"))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	var buf bytes.Buffer
	err := suite.client.LocalDownloadWithContext(ctx, testCid, &buf)
	require.Error(suite.T(), err, "Expected error for HTTP 404")
	assert.Contains(suite.T(), err.Error(), "404", "Expected '404' in error message")
}

func (suite *CodexClientTestSuite) TestLocalDownloadWithContext_Cancellation() {
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("slow response"))
	}))

	suite.client.BaseURL = suite.server.URL

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	var buf bytes.Buffer
	err := suite.client.LocalDownloadWithContext(ctx, testCid, &buf)
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
	testCid := "zDvZRwzmTestCID"
	expectedManifest := `{
		"cid": "zDvZRwzmTestCID",
		"manifest": {
			"treeCid": "zDvZRwzmTreeCID123",
			"datasetSize": 1024,
			"blockSize": 256,
			"protected": true,
			"filename": "test-file.bin",
			"mimetype": "application/octet-stream"
		}
	}`

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(suite.T(), http.MethodGet, r.Method)
		expectedPath := fmt.Sprintf("/api/codex/v1/data/%s/network/manifest", testCid)
		assert.Equal(suite.T(), expectedPath, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedManifest))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.FetchManifestWithContext(ctx, testCid)
	require.NoError(suite.T(), err, "Expected no error")
	require.NotNil(suite.T(), manifest, "Expected manifest, got nil")

	assert.Equal(suite.T(), testCid, manifest.CID)
	assert.Equal(suite.T(), "zDvZRwzmTreeCID123", manifest.Manifest.TreeCid)
	assert.Equal(suite.T(), int64(1024), manifest.Manifest.DatasetSize)
	assert.Equal(suite.T(), 256, manifest.Manifest.BlockSize)
	assert.True(suite.T(), manifest.Manifest.Protected, "Expected Protected to be true")
	assert.Equal(suite.T(), "test-file.bin", manifest.Manifest.Filename)
	assert.Equal(suite.T(), "application/octet-stream", manifest.Manifest.Mimetype)
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_RequestError() {
	// Create a server and immediately close it to trigger connection error
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	suite.server.Close()

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.FetchManifestWithContext(ctx, "test-cid")
	require.Error(suite.T(), err, "Expected error for closed server")
	assert.Nil(suite.T(), manifest, "Expected nil manifest on error")
	assert.Contains(suite.T(), err.Error(), "failed to fetch manifest from codex")
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_HTTPError() {
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Manifest not found"))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.FetchManifestWithContext(ctx, testCid)
	require.Error(suite.T(), err, "Expected error for HTTP 404")
	assert.Nil(suite.T(), manifest, "Expected nil manifest on error")
	assert.Contains(suite.T(), err.Error(), "404", "Expected '404' in error message")
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_JSONParseError() {
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json {"))
	}))

	suite.client.BaseURL = suite.server.URL

	ctx := context.Background()
	manifest, err := suite.client.FetchManifestWithContext(ctx, testCid)
	require.Error(suite.T(), err, "Expected error for invalid JSON")
	assert.Nil(suite.T(), manifest, "Expected nil manifest on JSON parse error")
	assert.Contains(suite.T(), err.Error(), "failed to parse manifest", "Expected 'failed to parse manifest' in error message")
}

func (suite *CodexClientTestSuite) TestFetchManifestWithContext_Cancellation() {
	testCid := "zDvZRwzmTestCID"

	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"cid": "test"}`))
	}))

	suite.client.BaseURL = suite.server.URL

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	manifest, err := suite.client.FetchManifestWithContext(ctx, testCid)
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
}
