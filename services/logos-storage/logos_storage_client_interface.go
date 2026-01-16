package logosstorage

import (
	"context"
	"io"

	"github.com/codex-storage/codex-go-bindings/codex"
)

//go:generate go tool mockgen -package=mock_logosstorage -source=logos_storage_client_interface.go -destination=mock/logos_storage_client_interface.go

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
	TriggerDownload(cid string) (codex.Manifest, error)
	TriggerDownloadWithContext(ctx context.Context, cid string) (codex.Manifest, error)

	// Manifest methods
	FetchManifestWithContext(ctx context.Context, cid string) (codex.Manifest, error)

	// CID management methods
	HasCid(cid string) (bool, error)
	RemoveCid(cid string) error

	// lifecycle management methods
	Start() error
	Stop() error
	Destroy() error

	// Peer Management methods
	PeerId() (string, error)
	Debug() (codex.DebugInfo, error)
	Connect(peerId string, peerAddresses []string) error

	// logging methods
	UpdateLogLevel(logLevel string) error
}
