package communities

import (
	"context"
	"io"
	"time"
)

//go:generate go tool mockgen -package=mock_communities -source=codex_client_interface.go -destination=mock/communities/codex_client_interface.go

// CodexClientInterface defines the interface for CodexClient operations needed by the downloader
type CodexClientInterface interface {
	// Upload methods
	Upload(data io.Reader, filename string) (string, error)
	UploadArchive(encodedArchive []byte) (string, error)

	// Download methods
	Download(cid string, output io.Writer) error
	DownloadWithContext(ctx context.Context, cid string, output io.Writer) error
	LocalDownload(cid string, output io.Writer) error
	LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error

	// Async download methods
	TriggerDownload(cid string) (*CodexManifest, error)
	TriggerDownloadWithContext(ctx context.Context, cid string) (*CodexManifest, error)

	// Manifest methods
	FetchManifestWithContext(ctx context.Context, cid string) (*CodexManifest, error)

	// CID management methods
	HasCid(cid string) (bool, error)
	RemoveCid(cid string) error

	// Configuration methods
	SetRequestTimeout(timeout time.Duration)
}
