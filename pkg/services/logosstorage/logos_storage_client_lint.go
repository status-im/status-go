//go:build use_logos_storage && lint

package logosstorage

import (
	"context"
	"errors"
	"io"

	"github.com/status-im/status-go/params"
)

// ErrLintBuild indicates this is a lint-only stub without native libstorage bindings.
var ErrLintBuild = errors.New("logosstorage: lint-only build stub: native libstorage not linked")

// LogosStorageManifest is duplicated in lint builds to avoid importing native bindings.
type LogosStorageManifest struct {
	Cid         string
	TreeCid     string
	DatasetSize int
	BlockSize   int
	Filename    string
	Mimetype    string
	Protected   bool
}

// LogosStorageClient is a lint-only stub implementation.
type LogosStorageClient struct{}

func NewLogosStorageClient(_ params.LogosStorageConfig) (LogosStorageClientInterface, error) {
	return &LogosStorageClient{}, nil
}

func (c *LogosStorageClient) Start() error {
	return ErrLintBuild
}

func (c *LogosStorageClient) Stop() error {
	return ErrLintBuild
}

func (c *LogosStorageClient) Destroy() error {
	return ErrLintBuild
}

func (c *LogosStorageClient) UpdateLogLevel(_ string) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) Upload(_ io.Reader, _ string) (string, error) {
	return "", ErrLintBuild
}

func (c *LogosStorageClient) UploadArchive(_ []byte) (string, error) {
	return "", ErrLintBuild
}

func (c *LogosStorageClient) Download(_ string, _ io.Writer) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) DownloadWithContext(_ context.Context, _ string, _ io.Writer) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) LocalDownload(_ string, _ io.Writer) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) LocalDownloadWithContext(_ context.Context, _ string, _ io.Writer) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) TriggerDownload(_ string) (LogosStorageManifest, error) {
	return LogosStorageManifest{}, ErrLintBuild
}

func (c *LogosStorageClient) TriggerDownloadWithContext(_ context.Context, _ string) (LogosStorageManifest, error) {
	return LogosStorageManifest{}, ErrLintBuild
}

func (c *LogosStorageClient) FetchManifestWithContext(_ context.Context, _ string) (LogosStorageManifest, error) {
	return LogosStorageManifest{}, ErrLintBuild
}

func (c *LogosStorageClient) HasCid(_ string) (bool, error) {
	return false, ErrLintBuild
}

func (c *LogosStorageClient) RemoveCid(_ string) error {
	return ErrLintBuild
}

func (c *LogosStorageClient) PeerId() (string, error) {
	return "", ErrLintBuild
}

func (c *LogosStorageClient) Debug() (LogosStorageDebugInfo, error) {
	return LogosStorageDebugInfo{}, ErrLintBuild
}

func (c *LogosStorageClient) Connect(_ string, _ []string) error {
	return ErrLintBuild
}
