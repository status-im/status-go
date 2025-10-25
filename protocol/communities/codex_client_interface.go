package communities

import (
	"context"
	"io"
	"time"
)

// Mock generation instruction above will create a mock in package `mock_communities`
// (folder `mock/`) so tests can import it as e.g. `go-codex-client/communities/mock` or
// with an alias like `mocks` to avoid import-cycle issues.
//
// CodexClientInterface defines the interface for CodexClient operations needed by the downloader
//
//go:generate mockgen -package=mock_communities -source=codex_client_interface.go -destination=mock/codex_client_interface.go
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
