//go:build use_logos_storage
// +build use_logos_storage

package logosstorage

import (
	"bytes"
	"context"
	"io"

	"github.com/logos-storage/logos-storage-go-bindings/storage"

	"github.com/status-im/status-go/params"
)

// LogosStorageClient handles basic upload/download operations with LogosStorage storage
type LogosStorageClient struct {
	config    storage.Config
	node      *storage.StorageNode
	enabled   bool
	started   bool
	stopped   bool
	destroyed bool
}

// NewLogosStorageClient creates a new LogosStorage client
func NewLogosStorageClient(config params.LogosStorageConfig) (LogosStorageClientInterface, error) {
	node, err := storage.New(config.LogosStorageNodeConfig)
	if err != nil {
		return nil, err
	}

	return &LogosStorageClient{
		config:  config.LogosStorageNodeConfig,
		node:    node,
		enabled: config.Enabled,
	}, nil
}

func (c *LogosStorageClient) Start() error {
	if c.started {
		return nil
	}
	err := c.node.Start()
	if err != nil {
		return err
	}
	c.started = true
	return nil
}

func (c *LogosStorageClient) Stop() error {
	if c.stopped {
		return nil
	}
	err := c.node.Stop()
	if err != nil {
		return err
	}
	c.stopped = true
	return nil
}

func (c *LogosStorageClient) Destroy() error {
	if c.destroyed {
		return nil
	}
	err := c.node.Destroy()
	if err != nil {
		return err
	}
	c.destroyed = true
	return nil
}

func (c *LogosStorageClient) UpdateLogLevel(logLevel string) error {
	return c.node.UpdateLogLevel(logLevel)
}

// Upload uploads data from a reader to LogosStorage and returns the CID
func (c *LogosStorageClient) Upload(data io.Reader, filename string) (string, error) {
	options := storage.UploadOptions{
		Filepath: filename,
	}
	return c.node.UploadReader(context.Background(), options, data)
}

// Download downloads data from LogosStorage by CID and writes it to the provided writer
func (c *LogosStorageClient) Download(cid string, output io.Writer) error {
	return c.DownloadWithContext(context.Background(), cid, output)
}

func (c *LogosStorageClient) TriggerDownload(cid string) (storage.Manifest, error) {
	return c.TriggerDownloadWithContext(context.Background(), cid)
}

// HasCid checks if the given CID exists in LogosStorage storage
func (c *LogosStorageClient) HasCid(cid string) (bool, error) {
	return c.node.Exists(cid)
}

func (c *LogosStorageClient) RemoveCid(cid string) error {
	return c.node.Delete(cid)
}

// DownloadWithContext downloads data from LogosStorage by CID with cancellation support
func (c *LogosStorageClient) DownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	options := storage.DownloadStreamOptions{
		Writer: output,
	}
	return c.node.DownloadStream(ctx, cid, options)
}

func (c *LogosStorageClient) LocalDownload(cid string, output io.Writer) error {
	return c.LocalDownloadWithContext(context.Background(), cid, output)
}

func (c *LogosStorageClient) LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	return c.node.DownloadStream(ctx, cid, storage.DownloadStreamOptions{
		Writer: output,
		Local:  true,
	})
}

func (c *LogosStorageClient) FetchManifestWithContext(ctx context.Context, cid string) (storage.Manifest, error) {
	return c.DownloadManifest(cid)
}

func (c *LogosStorageClient) TriggerDownloadWithContext(ctx context.Context, cid string) (storage.Manifest, error) {
	return c.node.Fetch(cid)
}

// UploadArchive is a convenience method for uploading archive data
func (c *LogosStorageClient) UploadArchive(encodedArchive []byte) (string, error) {
	return c.Upload(bytes.NewReader(encodedArchive), "archive-data.bin")
}

func (c *LogosStorageClient) DownloadManifest(cid string) (storage.Manifest, error) {
	return c.node.DownloadManifest(cid)
}

func (c *LogosStorageClient) PeerId() (string, error) {
	return c.node.PeerId()
}

func (c *LogosStorageClient) Debug() (storage.DebugInfo, error) {
	return c.node.Debug()
}

func (c *LogosStorageClient) Connect(peerId string, peerAddresses []string) error {
	return c.node.Connect(peerId, peerAddresses)
}
