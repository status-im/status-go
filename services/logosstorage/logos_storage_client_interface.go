//go:build use_logos_storage
// +build use_logos_storage

package logosstorage

import (
	"context"
	"io"

	"github.com/logos-storage/logos-storage-go-bindings/storage"
)

//go:generate sh -c "go tool mockgen -package=mock_logosstorage -source=logos_storage_client_interface.go -destination=mock/logos_storage_client_interface.go && { printf '//go:build use_logos_storage\n// +build use_logos_storage\n\n'; cat mock/logos_storage_client_interface.go; } > mock/logos_storage_client_interface.go.tmp && mv mock/logos_storage_client_interface.go.tmp mock/logos_storage_client_interface.go"

// LogosStorageClientInterface defines the interface for LogosStorageClient operations needed by the downloader
type LogosStorageClientInterface interface {
	// Upload methods
	Upload(data io.Reader, filename string) (string, error)
	UploadArchive(encodedArchive []byte) (string, error)

	// Download methods
	Download(cid string, output io.Writer) error
	DownloadWithContext(ctx context.Context, cid string, output io.Writer) error
	LocalDownload(cid string, output io.Writer) error
	LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error

	// Async download methods
	TriggerDownload(cid string) (storage.Manifest, error)
	TriggerDownloadWithContext(ctx context.Context, cid string) (storage.Manifest, error)

	// Manifest methods
	FetchManifestWithContext(ctx context.Context, cid string) (storage.Manifest, error)
	// CID management methods
	HasCid(cid string) (bool, error)
	RemoveCid(cid string) error

	// lifecycle management methods
	Start() error
	Stop() error
	Destroy() error

	// Peer Management methods
	PeerId() (string, error)
	Debug() (storage.DebugInfo, error)
	Connect(peerId string, peerAddresses []string) error

	// logging methods
	UpdateLogLevel(logLevel string) error
}
