package communities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CodexClient handles basic upload/download operations with Codex storage
type CodexClient struct {
	BaseURL string
	Client  *http.Client
}

// NewCodexClient creates a new Codex client
func NewCodexClient(host string, port string) *CodexClient {
	return &CodexClient{
		BaseURL: fmt.Sprintf("http://%s:%s", host, port),
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Upload uploads data from a reader to Codex and returns the CID
func (c *CodexClient) Upload(data io.Reader, filename string) (string, error) {
	url := fmt.Sprintf("%s/api/codex/v1/data", c.BaseURL)

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, filename))

	// Send request
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload to codex: %w", err)
	}
	defer resp.Body.Close()

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("codex upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the CID response
	cidBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	cid := strings.TrimSpace(string(cidBytes))
	return cid, nil
}

// Download downloads data from Codex by CID and writes it to the provided writer
func (c *CodexClient) Download(cid string, output io.Writer) error {
	return c.DownloadWithContext(context.Background(), cid, output)
}

func (c *CodexClient) TriggerDownload(cid string) (*CodexManifest, error) {
	return c.TriggerDownloadWithContext(context.Background(), cid)
}

func (c *CodexClient) HasCid(cid string) (bool, error) {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s/exists", c.BaseURL, cid)

	resp, err := c.Client.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to check cid existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("cid check failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response: {"<cid>": <bool>}
	var result map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate the CID key matches request
	hasCid, exists := result[cid]
	if !exists {
		return false, fmt.Errorf("response missing CID key %q", cid)
	}

	return hasCid, nil
}

func (c *CodexClient) RemoveCid(cid string) error {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s", c.BaseURL, cid)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed trying to delete cid: %s, %w", cid, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cid delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadWithContext downloads data from Codex by CID with cancellation support
func (c *CodexClient) DownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s/network/stream", c.BaseURL, cid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from codex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("codex download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Use context-aware copy for cancellable streaming
	return c.copyWithContext(ctx, output, resp.Body)
}

func (c *CodexClient) LocalDownload(cid string, output io.Writer) error {
	return c.LocalDownloadWithContext(context.Background(), cid, output)
}

func (c *CodexClient) LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s", c.BaseURL, cid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from codex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("codex download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Use context-aware copy for cancellable streaming
	return c.copyWithContext(ctx, output, resp.Body)
}

func (c *CodexClient) FetchManifestWithContext(ctx context.Context, cid string) (*CodexManifest, error) {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s/network/manifest", c.BaseURL, cid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from codex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("codex fetch manifest failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response containing manifest
	var manifest CodexManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func (c *CodexClient) TriggerDownloadWithContext(ctx context.Context, cid string) (*CodexManifest, error) {
	url := fmt.Sprintf("%s/api/codex/v1/data/%s/network", c.BaseURL, cid)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger download from codex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("codex async download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response containing manifest
	var manifest CodexManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse download manifest: %w", err)
	}

	return &manifest, nil
}

// copyWithContext performs io.Copy but respects context cancellation
func (c *CodexClient) copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	// Create a buffer for chunked copying
	buf := make([]byte, 64*1024) // 64KB buffer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err() // Return cancellation error
		default:
		}

		// Read a chunk
		n, err := src.Read(buf)
		if n > 0 {
			// Write the chunk
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write data: %w", writeErr)
			}
		}

		if err == io.EOF {
			return nil // Successful completion
		}
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}
	}
}

// SetRequestTimeout sets the HTTP client timeout for requests
func (c *CodexClient) SetRequestTimeout(timeout time.Duration) {
	c.Client.Timeout = timeout
}

// UploadArchive is a convenience method for uploading archive data
func (c *CodexClient) UploadArchive(encodedArchive []byte) (string, error) {
	return c.Upload(bytes.NewReader(encodedArchive), "archive-data.bin")
}
