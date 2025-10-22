package communities

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/codex-storage/codex-go-bindings/codex"
)

// CodexClient handles basic upload/download operations with Codex storage
type CodexClient struct {
	config codex.Config
	node   *codex.CodexNode
}

// NewCodexClient creates a new Codex client
func NewCodexClient(config codex.Config) (CodexClient, error) {
	node, err := codex.New(config)
	if err != nil {
		return CodexClient{}, err
	}

	return CodexClient{
		config: config,
		node:   node,
	}, nil
}

func (c CodexClient) Start() error {
	log.Println("Starting Codex node...")
	return c.node.Start()
}

func (c CodexClient) Stop() error {
	return c.node.Stop()
}

func (c CodexClient) Destroy() error {
	return c.node.Destroy()
}

func (c *CodexClient) UpdateLogLevel(logLevel string) error {
	return c.node.UpdateLogLevel(logLevel)
}

// Upload uploads data from a reader to Codex and returns the CID
func (c *CodexClient) Upload(data io.Reader, filename string) (string, error) {
	options := codex.UploadOptions{
		Filepath: filename,
	}
	return c.node.UploadReader(options, data)
}

// Download downloads data from Codex by CID and writes it to the provided writer
func (c *CodexClient) Download(cid string, output io.Writer) error {
	return c.DownloadWithContext(context.Background(), cid, output)
}

func (c *CodexClient) TriggerDownload(cid string) (codex.Manifest, error) {
	return c.TriggerDownloadWithContext(context.Background(), cid)
}

// HasCid checks if the given CID exists in Codex storage
// TODO: When the PR is merge https://github.com/codex-storage/nim-codex/pull/1331
// add the HasCid method to the codex-go-bindings and improve this implementation.
func (c *CodexClient) HasCid(cid string) (bool, error) {
	err := c.LocalDownload(cid, io.Discard)
	return err == nil, nil
}

func (c *CodexClient) RemoveCid(cid string) error {
	return c.node.Delete(cid)
}

// DownloadWithContext downloads data from Codex by CID with cancellation support
func (c *CodexClient) DownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	options := codex.DownloadStreamOptions{
		Writer: output,
	}
	return c.node.DownloadStream(cid, options)
}

func (c *CodexClient) LocalDownload(cid string, output io.Writer) error {
	return c.node.DownloadStream(cid, codex.DownloadStreamOptions{Writer: output})
}

func (c *CodexClient) LocalDownloadWithContext(ctx context.Context, cid string, output io.Writer) error {
	return c.LocalDownload(cid, output)
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
