package communities

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CodexClient handles uploading data to Codex storage
type CodexClient struct {
	baseURL string
	client  *http.Client
}

// NewCodexClient creates a new Codex client
func NewCodexClient(host string, port string) *CodexClient {
	return &CodexClient{
		baseURL: fmt.Sprintf("http://%s:%s", host, port),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// UploadData uploads binary data to Codex and returns the CID
func (c *CodexClient) UploadData(data []byte, filename string) (string, error) {
	url := fmt.Sprintf("%s/api/codex/v1/data", c.baseURL)

	// Create a bytes reader from the data
	reader := bytes.NewReader(data)

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, filename))

	// Send request
	resp, err := c.client.Do(req)
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

// UploadArchive is a convenience method for uploading archive data
func (c *CodexClient) UploadArchive(encodedArchive []byte) (string, error) {
	return c.UploadData(encodedArchive, "archive-data.bin")
}
