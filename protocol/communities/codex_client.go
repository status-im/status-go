package communities

import (
	"bytes"
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
	url := fmt.Sprintf("%s/api/codex/v1/data/%s/network/stream", c.BaseURL, cid)
	
	resp, err := c.Client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download from codex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("codex download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Stream the data to the output writer
	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded data: %w", err)
	}

	return nil
}

// SetRequestTimeout sets the HTTP client timeout for requests
func (c *CodexClient) SetRequestTimeout(timeout time.Duration) {
	c.Client.Timeout = timeout
}

// UploadArchive is a convenience method for uploading archive data
func (c *CodexClient) UploadArchive(encodedArchive []byte) (string, error) {
	return c.Upload(bytes.NewReader(encodedArchive), "archive-data.bin")
}
