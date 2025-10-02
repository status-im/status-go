//go:build !disable_torrent
// +build !disable_torrent

package communities

import (
	"fmt"
	"sync"

	"github.com/status-im/status-go/protocol/protobuf"
)

// CodexArchiveDownloader handles downloading individual archive files from Codex storage
type CodexArchiveDownloader struct {
	codexClient         *CodexClient
	index              *protobuf.CodexWakuMessageArchiveIndex
	communityID        string
	existingArchiveIDs []string
	
	// Progress tracking
	totalArchivesCount           int
	totalDownloadedArchivesCount int
	currentArchiveHash          string
	archiveDownloadProgress     map[string]int64 // hash -> bytes downloaded
	mu                          sync.RWMutex
	
	// Download control
	downloadComplete            bool
	downloadError              error
	
	// Callback for signaling archive completion
	onArchiveDownloaded func(hash string, from, to uint64)
}

// NewCodexArchiveDownloader creates a new archive downloader
func NewCodexArchiveDownloader(codexClient *CodexClient, index *protobuf.CodexWakuMessageArchiveIndex, communityID string, existingArchiveIDs []string) *CodexArchiveDownloader {
	return &CodexArchiveDownloader{
		codexClient:                 codexClient,
		index:                      index,
		communityID:                communityID,
		existingArchiveIDs:         existingArchiveIDs,
		totalArchivesCount:         len(index.Archives),
		totalDownloadedArchivesCount: len(existingArchiveIDs),
		archiveDownloadProgress:    make(map[string]int64),
	}
}

// SetOnArchiveDownloaded sets a callback function to be called when an archive is successfully downloaded
func (d *CodexArchiveDownloader) SetOnArchiveDownloaded(callback func(hash string, from, to uint64)) {
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
		
		// Update progress and call callback
		d.mu.Lock()
		d.totalDownloadedArchivesCount++
		d.mu.Unlock()
		
		// Notify about successful download
		if d.onArchiveDownloaded != nil {
			metadata := d.index.Archives[archive.hash]
			d.onArchiveDownloaded(archive.hash, metadata.Metadata.From, metadata.Metadata.To)
		}
	}
	
	return nil
}

// downloadSingleArchive downloads a single archive by its CID
func (d *CodexArchiveDownloader) downloadSingleArchive(hash, cid string) error {
	// For now, we'll use a simple approach - download the entire archive into memory
	// In the future, this could be enhanced to stream directly to the message processing pipeline
	// TODO: Implement streaming directly to message processor to avoid memory usage
	
	// Create a progress tracking writer that discards the data for now
	progressWriter := &archiveProgressWriter{
		hash:     hash,
		progress: &d.archiveDownloadProgress,
		mu:       &d.mu,
	}
	
	// Download the archive data
	err := d.codexClient.Download(cid, progressWriter)
	if err != nil {
		return fmt.Errorf("failed to download archive data for CID %s: %w", cid, err)
	}
	
	return nil
}

// archiveProgressWriter tracks download progress for individual archives
type archiveProgressWriter struct {
	hash     string
	progress *map[string]int64
	mu       *sync.RWMutex
}

func (apw *archiveProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	apw.mu.Lock()
	(*apw.progress)[apw.hash] += int64(n)
	apw.mu.Unlock()
	
	// For now, we discard the data since we're not processing it yet
	// TODO: Process the downloaded archive data here
	return n, nil
}