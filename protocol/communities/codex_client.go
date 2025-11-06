package communities

import (
	"bytes"
	"context"
	"io"

	"github.com/codex-storage/codex-go-bindings/codex"

	"github.com/status-im/status-go/params"
)

// CodexClient handles basic upload/download operations with Codex storage
type CodexClient struct {
	config    codex.Config
	node      *codex.CodexNode
	enabled   bool
	started   bool
	stopped   bool
	destroyed bool
}

// NewCodexClient creates a new Codex client
func NewCodexClient(config params.CodexConfig) (CodexClientInterface, error) {
	node, err := codex.New(config.CodexNodeConfig)
	if err != nil {
		return nil, err
	}

	return &CodexClient{
		config:  config.CodexNodeConfig,
		node:    node,
		enabled: config.Enabled,
	}, nil
}

func (c *CodexClient) Start() error {
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

func (c *CodexClient) Stop() error {
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

func (c *CodexClient) Destroy() error {
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

func (c *CodexClient) UpdateLogLevel(logLevel string) error {
	return c.node.UpdateLogLevel(logLevel)
}

// Upload uploads data from a reader to Codex and returns the CID
func (c *CodexClient) Upload(data io.Reader, filename string) (string, error) {
	options := codex.UploadOptions{
		Filepath: filename,
	}
	return c.node.UploadReader(context.Background(), options, data)
}

// Download downloads data from Codex by CID and writes it to the provided writer
func (c *CodexClient) Download(cid string, output io.Writer) error {
	return c.DownloadWithContext(context.Background(), cid, output)
}

func (c *CodexClient) TriggerDownload(cid string) (codex.Manifest, error) {
	return c.TriggerDownloadWithContext(context.Background(), cid)
}

// HasCid checks if the given CID exists in Codex storage
func (c *CodexClient) HasCid(cid string) (bool, error) {
	return c.node.Exists(cid)
}

func (c *CodexClient) RemoveCid(cid string) error {
	return c.node.Delete(cid)
}

// DownloadWithContext downloads data from Codex by CID with cancellation support
func (c *CodexClient) DownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	options := codex.DownloadStreamOptions{
		Writer: output,
	}
	return c.node.DownloadStream(ctx, cid, options)
}

func (c *CodexClient) LocalDownload(cid string, output io.Writer) error {
	return c.LocalDownloadWithContext(context.Background(), cid, output)
}

func (c *CodexClient) LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	return c.node.DownloadStream(ctx, cid, codex.DownloadStreamOptions{
		Writer: output,
		Local:  true,
	})
}

func (c *CodexClient) FetchManifestWithContext(ctx context.Context, cid string) (codex.Manifest, error) {
	return c.DownloadManifest(cid)
}

func (c *CodexClient) TriggerDownloadWithContext(ctx context.Context, cid string) (codex.Manifest, error) {
	return c.node.Fetch(cid)
}

// UploadArchive is a convenience method for uploading archive data
func (c *CodexClient) UploadArchive(encodedArchive []byte) (string, error) {
	return c.Upload(bytes.NewReader(encodedArchive), "archive-data.bin")
}

func (c *CodexClient) DownloadManifest(cid string) (codex.Manifest, error) {
	return c.node.DownloadManifest(cid)
}

func (c *CodexClient) PeerId() (string, error) {
	return c.node.PeerId()
}

func (c *CodexClient) Debug() (codex.DebugInfo, error) {
	return c.node.Debug()
}

func (c *CodexClient) Connect(peerId string, peerAddresses []string) error {
	return c.node.Connect(peerId, peerAddresses)
}
