package communities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ManifestResponse represents the response from Codex manifest API
type ManifestResponse struct {
	CID      string `json:"cid"`
	Manifest struct {
		TreeCID     string `json:"treeCid"`
		DatasetSize int64  `json:"datasetSize"`
		BlockSize   int    `json:"blockSize"`
		Protected   bool   `json:"protected"`
		Filename    string `json:"filename"`
		Mimetype    string `json:"mimetype"`
	} `json:"manifest"`
}

// CodexIndexDownloader handles downloading index files from Codex storage
type CodexIndexDownloader struct {
	codexClient    *CodexClient
	indexCid       string
	filePath       string
	datasetSize    int64           // stores the dataset size from the manifest
	bytesCompleted int64           // tracks download progress
	cancelChan     <-chan struct{} // for cancellation support
}

// NewCodexIndexDownloader creates a new index downloader
func NewCodexIndexDownloader(codexClient *CodexClient, indexCid string, filePath string, cancelChan <-chan struct{}) *CodexIndexDownloader {
	return &CodexIndexDownloader{
		codexClient: codexClient,
		indexCid:    indexCid,
		filePath:    filePath,
		cancelChan:  cancelChan,
	}
}

// GotManifest returns a channel that is closed when the Codex manifest file
// for the configured CID is successfully fetched. On error, the channel is not closed
// (allowing timeout to handle failures). Check GetDatasetSize() > 0 to verify success.
func (d *CodexIndexDownloader) GotManifest() <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		// Reset datasetSize to 0 to indicate no successful fetch yet
		d.datasetSize = 0

		// Check for cancellation before starting
		select {
		case <-d.cancelChan:
			return // Exit without closing channel - cancellation
		default:
		}

		// Create cancellable context for HTTP request
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Monitor for cancellation in separate goroutine
		go func() {
			select {
			case <-d.cancelChan:
				cancel()
			case <-ctx.Done():
			}
		}()

		// Fetch manifest from Codex
		url := fmt.Sprintf("%s/api/codex/v1/data/%s/network/manifest", d.codexClient.BaseURL, d.indexCid)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return
		}

		resp, err := d.codexClient.Client.Do(req)
		if err != nil {
			// Don't close channel on error - let timeout handle it
			return
		}
		defer resp.Body.Close()

		// Check if request was successful
		if resp.StatusCode != http.StatusOK {
			// Don't close channel on error - let timeout handle it
			return
		}

		// Parse the JSON response
		var manifestResp ManifestResponse
		if err := json.NewDecoder(resp.Body).Decode(&manifestResp); err != nil {
			// Don't close channel on error - let timeout handle it
			return
		}

		// Verify that the CID matches our configured indexCid
		if manifestResp.CID != d.indexCid {
			// Don't close channel on error - let timeout handle it
			return
		}

		// Store the dataset size for later use - this indicates success
		d.datasetSize = manifestResp.Manifest.DatasetSize

		// Success! Close the channel to signal completion
		close(ch)
	}()

	return ch
}

// GetDatasetSize returns the dataset size from the last successfully fetched manifest
func (d *CodexIndexDownloader) GetDatasetSize() int64 {
	return d.datasetSize
}

// DownloadIndexFile starts downloading the index file from Codex and writes it to the configured file path
func (d *CodexIndexDownloader) DownloadIndexFile() error {
	// Reset progress counter
	d.bytesCompleted = 0

	go func() {
		// Check for cancellation before starting
		select {
		case <-d.cancelChan:
			return // Exit early if cancelled
		default:
		}

		// Create cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Monitor for cancellation
		go func() {
			select {
			case <-d.cancelChan:
				cancel() // Cancel download immediately
			case <-ctx.Done():
				// Context already cancelled, nothing to do
			}
		}()

		// Create the output file
		file, err := os.Create(d.filePath)
		if err != nil {
			// TODO: Consider logging the error or exposing it somehow
			return
		}
		defer file.Close()

		// Create a progress tracking writer
		progressWriter := &progressWriter{
			writer:    file,
			completed: &d.bytesCompleted,
		}

		// Use CodexClient to download and stream to file with context for cancellation
		err = d.codexClient.DownloadWithContext(ctx, d.indexCid, progressWriter)
		if err != nil {
			// TODO: Consider logging the error or exposing it somehow
			return
		}
	}()

	return nil
}

// BytesCompleted returns the number of bytes downloaded so far
func (d *CodexIndexDownloader) BytesCompleted() int64 {
	return d.bytesCompleted
}

// Length returns the total dataset size (equivalent to torrent file length)
func (d *CodexIndexDownloader) Length() int64 {
	return d.datasetSize
}

// progressWriter wraps an io.Writer to track bytes written
type progressWriter struct {
	writer    io.Writer
	completed *int64
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	if n > 0 {
		*pw.completed += int64(n)
	}
	return n, err
}
