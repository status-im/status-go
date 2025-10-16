//go:build !disable_torrent
// +build !disable_torrent

package communities

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// CodexArchiveProcessor handles processing of downloaded archive data
type CodexArchiveProcessor interface {
	// ProcessArchiveData processes the raw archive data and returns any error
	// The processor is responsible for extracting messages, handling them,
	// and saving the archive ID to persistence
	ProcessArchiveData(communityID types.HexBytes, archiveHash string, archiveData []byte, from, to uint64) error
}

// CodexArchiveDownloader handles downloading individual archive files from Codex storage
type CodexArchiveDownloader struct {
	codexClient        *CodexClient
	index              *protobuf.CodexWakuMessageArchiveIndex
	communityID        string
	existingArchiveIDs []string
	cancelChan         chan struct{} // for cancellation support

	// Progress tracking
	totalArchivesCount           int
	totalDownloadedArchivesCount int
	currentArchiveHash           string
	archiveDownloadProgress      map[string]int64 // hash -> bytes downloaded
	mu                           sync.RWMutex

	// Download control
	downloadComplete bool
	downloadError    error
	cancelled        bool

	// Callback for signaling archive completion with raw data
	onArchiveDownloaded func(hash string, from, to uint64, archiveData []byte)
}

// NewCodexArchiveDownloader creates a new archive downloader
func NewCodexArchiveDownloader(codexClient *CodexClient, index *protobuf.CodexWakuMessageArchiveIndex, communityID string, existingArchiveIDs []string, cancelChan chan struct{}) *CodexArchiveDownloader {
	return &CodexArchiveDownloader{
		codexClient:                  codexClient,
		index:                        index,
		communityID:                  communityID,
		existingArchiveIDs:           existingArchiveIDs,
		cancelChan:                   cancelChan,
		totalArchivesCount:           len(index.Archives),
		totalDownloadedArchivesCount: len(existingArchiveIDs),
		archiveDownloadProgress:      make(map[string]int64),
	}
}

// SetOnArchiveDownloaded sets a callback function to be called when an archive is successfully downloaded
func (d *CodexArchiveDownloader) SetOnArchiveDownloaded(callback func(hash string, from, to uint64, archiveData []byte)) {
	d.onArchiveDownloaded = callback
}

// GetTotalArchivesCount returns the total number of archives to download
func (d *CodexArchiveDownloader) GetTotalArchivesCount() int {
	return d.totalArchivesCount
}

// GetTotalDownloadedArchivesCount returns the number of archives already downloaded
func (d *CodexArchiveDownloader) GetTotalDownloadedArchivesCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.totalDownloadedArchivesCount
}

// GetCurrentArchiveHash returns the hash of the currently downloading archive
func (d *CodexArchiveDownloader) GetCurrentArchiveHash() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentArchiveHash
}

// GetArchiveDownloadProgress returns the download progress for a specific archive
func (d *CodexArchiveDownloader) GetArchiveDownloadProgress(hash string) int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.archiveDownloadProgress[hash]
}

// IsDownloadComplete returns whether all archives have been downloaded
func (d *CodexArchiveDownloader) IsDownloadComplete() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.downloadComplete
}

// GetDownloadError returns any error that occurred during download
func (d *CodexArchiveDownloader) GetDownloadError() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.downloadError
}

// IsCancelled returns whether the download was cancelled
func (d *CodexArchiveDownloader) IsCancelled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cancelled
}

// StartDownload begins downloading all missing archives
func (d *CodexArchiveDownloader) StartDownload() error {
	go func() {
		err := d.downloadAllArchives()
		d.mu.Lock()
		d.downloadError = err
		d.downloadComplete = true
		d.mu.Unlock()
	}()
	return nil
}

// downloadAllArchives handles the main download loop for all archives
func (d *CodexArchiveDownloader) downloadAllArchives() error {
	// Create sorted list of archives (newest first, like torrent version)
	type archiveInfo struct {
		hash string
		from uint64
		cid  string
	}

	var archivesList []archiveInfo
	for hash, metadata := range d.index.Archives {
		archivesList = append(archivesList, archiveInfo{
			hash: hash,
			from: metadata.Metadata.From,
			cid:  metadata.Cid,
		})
	}

	// Sort by timestamp (newest first) - same as torrent version
	for i := 0; i < len(archivesList)-1; i++ {
		for j := i + 1; j < len(archivesList); j++ {
			if archivesList[i].from < archivesList[j].from {
				archivesList[i], archivesList[j] = archivesList[j], archivesList[i]
			}
		}
	}

	// Download each missing archive
	for _, archive := range archivesList {
		// Check for cancellation
		select {
		case <-d.cancelChan:
			d.mu.Lock()
			d.cancelled = true
			d.mu.Unlock()
			return nil // Return nil to indicate graceful cancellation
		default:
		}

		// Check if we already have this archive
		hasArchive := false
		for _, existingHash := range d.existingArchiveIDs {
			if existingHash == archive.hash {
				hasArchive = true
				break
			}
		}
		if hasArchive {
			continue
		}

		// Set current archive
		d.mu.Lock()
		d.currentArchiveHash = archive.hash
		d.archiveDownloadProgress[archive.hash] = 0
		d.mu.Unlock()

		// Download this archive using its CID
		err := d.downloadSingleArchive(archive.hash, archive.cid)
		if err != nil {
			return fmt.Errorf("failed to download archive %s: %w", archive.hash, err)
		}

		// Update progress (callback is handled in downloadSingleArchive)
		d.mu.Lock()
		d.totalDownloadedArchivesCount++
		d.mu.Unlock()
	}

	return nil
}

// downloadSingleArchive downloads a single archive by its CID
func (d *CodexArchiveDownloader) downloadSingleArchive(hash, cid string) error {
	// Create a context that can be cancelled via our cancel channel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Monitor for cancellation in a separate goroutine
	go func() {
		select {
		case <-d.cancelChan:
			cancel() // Cancel the download immediately
		case <-ctx.Done():
			// Context already cancelled, nothing to do
		}
	}()

	// Download the archive data into a buffer
	var archiveBuffer bytes.Buffer
	progressWriter := &archiveProgressWriter{
		buffer:   &archiveBuffer,
		hash:     hash,
		progress: &d.archiveDownloadProgress,
		mu:       &d.mu,
	}

	// Use context-aware download for immediate cancellation
	err := d.codexClient.DownloadWithContext(ctx, cid, progressWriter)
	if err != nil {
		return fmt.Errorf("failed to download archive data for CID %s: %w", cid, err)
	}

	// Get metadata for this archive
	metadata := d.index.Archives[hash]
	if metadata == nil {
		return fmt.Errorf("metadata not found for archive hash %s", hash)
	}

	// Call the callback with the downloaded archive data
	if d.onArchiveDownloaded != nil {
		d.onArchiveDownloaded(hash, metadata.Metadata.From, metadata.Metadata.To, archiveBuffer.Bytes())
	}

	return nil
}

// archiveProgressWriter tracks download progress and collects data for individual archives
type archiveProgressWriter struct {
	buffer   *bytes.Buffer
	hash     string
	progress *map[string]int64
	mu       *sync.RWMutex
}

func (apw *archiveProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	// Update progress tracking
	apw.mu.Lock()
	(*apw.progress)[apw.hash] += int64(n)
	apw.mu.Unlock()

	// Write data to buffer for processing
	return apw.buffer.Write(p)
}
