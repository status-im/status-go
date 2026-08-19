//go:build use_logos_storage

package logosstorage

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/protocol/protobuf"

	"go.uber.org/zap"
)

type LogosStorageArchiveDownloader struct {
	logosStorageClient LogosStorageClientInterface
	index              *protobuf.LogosStorageWakuMessageArchiveIndex
	communityID        string
	existingArchiveIDs []string
	cancelChan         chan struct{} // for cancellation support
	logger             *zap.Logger

	// Progress tracking
	totalArchivesCount           int
	totalDownloadedArchivesCount int
	archiveDownloadProgress      map[string]int64 // hash -> bytes downloaded
	archiveDownloadCancel        map[string]chan struct{}
	mu                           sync.RWMutex

	// Download control
	downloadComplete bool
	cancelled        bool
	pollingInterval  time.Duration // configurable polling interval for HasCid checks
	pollingTimeout   time.Duration // configurable timeout for HasCid polling

	// Callbacks
	onArchiveDownloaded       func(hash string, from, to uint64)
	onStartingArchiveDownload func(hash string, from, to uint64)
}

// NewLogosStorageArchiveDownloader creates a new archive downloader
func NewLogosStorageArchiveDownloader(logosStorageClient LogosStorageClientInterface, index *protobuf.LogosStorageWakuMessageArchiveIndex, communityID string, existingArchiveIDs []string, cancelChan chan struct{}, logger *zap.Logger) *LogosStorageArchiveDownloader {
	return &LogosStorageArchiveDownloader{
		logosStorageClient:           logosStorageClient,
		index:                        index,
		communityID:                  communityID,
		existingArchiveIDs:           existingArchiveIDs,
		cancelChan:                   cancelChan,
		logger:                       logger,
		totalArchivesCount:           len(index.Archives),
		totalDownloadedArchivesCount: len(existingArchiveIDs),
		archiveDownloadProgress:      make(map[string]int64),
		archiveDownloadCancel:        make(map[string]chan struct{}),
		pollingInterval:              1 * time.Second,  // Default production polling interval
		pollingTimeout:               30 * time.Second, // Default production polling timeout
	}
}

// SetPollingInterval sets the polling interval for HasCid checks (useful for testing)
func (d *LogosStorageArchiveDownloader) SetPollingInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pollingInterval = interval
}

// SetPollingTimeout sets the timeout for HasCid polling (useful for testing)
func (d *LogosStorageArchiveDownloader) SetPollingTimeout(timeout time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pollingTimeout = timeout
}

// SetOnArchiveDownloaded sets a callback function to be called when an archive is successfully downloaded
func (d *LogosStorageArchiveDownloader) SetOnArchiveDownloaded(callback func(hash string, from, to uint64)) {
	d.onArchiveDownloaded = callback
}

// SetOnStartingArchiveDownload sets a callback function to be called before starting an archive download
// This callback is called on the main thread before launching goroutines, making it useful for testing
// the deterministic order in which archives are processed (sorted newest first)
func (d *LogosStorageArchiveDownloader) SetOnStartingArchiveDownload(callback func(hash string, from, to uint64)) {
	d.onStartingArchiveDownload = callback
}

// GetTotalArchivesCount returns the total number of archives to download
func (d *LogosStorageArchiveDownloader) GetTotalArchivesCount() int {
	return d.totalArchivesCount
}

// GetTotalDownloadedArchivesCount returns the number of archives already downloaded
func (d *LogosStorageArchiveDownloader) GetTotalDownloadedArchivesCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.totalDownloadedArchivesCount
}

func (d *LogosStorageArchiveDownloader) GetPendingArchivesCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.archiveDownloadCancel)
}

// GetArchiveDownloadProgress returns the download progress for a specific archive
func (d *LogosStorageArchiveDownloader) GetArchiveDownloadProgress(hash string) int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.archiveDownloadProgress[hash]
}

// IsDownloadComplete returns whether all archives have been downloaded
func (d *LogosStorageArchiveDownloader) IsDownloadComplete() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.downloadComplete
}

// IsCancelled returns whether the download was cancelled
func (d *LogosStorageArchiveDownloader) IsCancelled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cancelled
}

// StartDownload begins downloading all missing archives
func (d *LogosStorageArchiveDownloader) StartDownload() {
	defer panics.LogOnPanic()
	d.downloadAllArchives()
}

// downloadAllArchives handles the main download loop for all archives
func (d *LogosStorageArchiveDownloader) downloadAllArchives() {
	defer panics.LogOnPanic()
	// Create sorted list of archives (newest first, like torrent version)
	type archiveInfo struct {
		hash string
		from uint64
		to   uint64
		cid  string
	}

	var archivesList []archiveInfo
	for hash, metadata := range d.index.Archives {
		// Skip archives we already have
		if slices.Contains(d.existingArchiveIDs, hash) {
			continue
		}
		archivesList = append(archivesList, archiveInfo{
			hash: hash,
			from: metadata.Metadata.From,
			to:   metadata.Metadata.To,
			cid:  metadata.Cid,
		})
	}

	// Sort by timestamp (newest first)
	slices.SortFunc(archivesList, func(a, b archiveInfo) int {
		if a.from > b.from {
			return -1 // a is newer, should come first
		}
		if a.from < b.from {
			return 1 // b is newer, should come first
		}
		return 0 // equal timestamps
	})

	// Pre-populate archiveDownloadCancel with all archives that need downloading
	// This ensures len(archiveDownloadCancel) correctly represents total pending work
	// and prevents race condition where fast-completing goroutines set downloadComplete=true
	// before all archives are added to the map
	d.mu.Lock()
	for _, archive := range archivesList {
		d.archiveDownloadCancel[archive.hash] = make(chan struct{})
		d.archiveDownloadProgress[archive.hash] = 0
	}
	d.mu.Unlock()

	// Monitor for cancellation in a separate goroutine
	go func() {
		defer panics.LogOnPanic()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-d.cancelChan:
				d.mu.Lock()
				for hash, cancelChan := range d.archiveDownloadCancel {
					select {
					case <-cancelChan:
						// Already closed
					default:
						close(cancelChan) // Safe to close
					}
					delete(d.archiveDownloadCancel, hash)
				}
				d.cancelled = true
				d.downloadComplete = true // Mark as complete even on cancellation
				d.mu.Unlock()
				return // Exit goroutine after cancellation
			case <-ticker.C:
				// Check if downloads are complete
				d.mu.RLock()
				complete := d.downloadComplete
				d.mu.RUnlock()

				if complete {
					return // Exit goroutine when downloads complete
				}
			}
		}
	}()

	// Download each missing archive
	for _, archive := range archivesList {
		// Get the pre-created cancel channel for this archive
		d.mu.RLock()
		archiveCancelChan := d.archiveDownloadCancel[archive.hash]
		d.mu.RUnlock()

		// Call callback before starting
		if d.onStartingArchiveDownload != nil {
			d.onStartingArchiveDownload(archive.hash, archive.from, archive.to)
		}

		// Trigger archive download and track progress in a goroutine
		go func(archiveHash, archiveCid string, archiveFrom, archiveTo uint64, archiveCancel chan struct{}) {
			defer panics.LogOnPanic()
			defer func() {
				// Always clean up: remove from active downloads and check completion
				d.mu.Lock()
				delete(d.archiveDownloadCancel, archiveHash)
				d.downloadComplete = len(d.archiveDownloadCancel) == 0
				d.mu.Unlock()
			}()

			err := d.triggerSingleArchiveDownload(archiveHash, archiveCid, archiveCancel)
			if err != nil {
				// Don't proceed to polling if trigger failed (could be cancellation or other error)
				d.logger.Debug("[LogosStorage] failed to trigger download",
					zap.String("cid", archiveCid),
					zap.String("hash", archiveHash),
					zap.Error(err))
				return
			}

			// Poll at configured interval until we confirm it's downloaded
			// or timeout, or get cancelled
			timeout := time.After(d.pollingTimeout)
			ticker := time.NewTicker(d.pollingInterval)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					d.logger.Debug("[LogosStorage] timeout waiting for CID to be available locally",
						zap.String("cid", archiveCid),
						zap.String("hash", archiveHash),
						zap.Duration("timeout", d.pollingTimeout))
					return
				case <-archiveCancel:
					d.logger.Debug("[LogosStorage] download cancelled",
						zap.String("cid", archiveCid),
						zap.String("hash", archiveHash))
					return
				case <-ticker.C:
					hasCid, err := d.logosStorageClient.HasCid(archiveCid)
					if err != nil {
						// Log error but continue polling
						d.logger.Debug("[LogosStorage] error checking CID availability",
							zap.String("cid", archiveCid),
							zap.String("hash", archiveHash),
							zap.Error(err))
						continue
					}
					if hasCid {
						// CID is now available locally - handle success immediately
						d.mu.Lock()
						d.totalDownloadedArchivesCount++
						d.mu.Unlock()

						d.logger.Debug(
							"[LogosStorage] archive download completed",
							zap.String("cid", archiveCid),
							zap.String("totalDownloadedArchivesCount", fmt.Sprintf("%d", d.totalDownloadedArchivesCount)),
						)

						// Call success callback
						if d.onArchiveDownloaded != nil {
							d.onArchiveDownloaded(archiveHash, archiveFrom, archiveTo)
						}
						return
					}
				}
			}
		}(archive.hash, archive.cid, archive.from, archive.to, archiveCancelChan)
	}
}

// triggerSingleArchiveDownload downloads a single archive by its CID
func (d *LogosStorageArchiveDownloader) triggerSingleArchiveDownload(_, cid string, cancelChan <-chan struct{}) error {
	defer panics.LogOnPanic()
	// Create a context that can be cancelled via our cancel channel
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Monitor for cancellation in a separate goroutine
	go func() {
		defer panics.LogOnPanic()
		select {
		case <-cancelChan:
			cancel() // Cancel the download immediately
		case <-ctx.Done():
			// Context already cancelled, nothing to do
		}
	}()

	manifest, err := d.logosStorageClient.TriggerDownloadWithContext(ctx, cid)
	if err != nil {
		return fmt.Errorf("failed to trigger archive download with CID %s: %w", cid, err)
	}

	if manifest.Cid != cid {
		return fmt.Errorf("unexpected manifest CID %s, expected %s", manifest.Cid, cid)
	}

	return nil
}
