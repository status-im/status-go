package logosstorage

import (
	"context"
	"io"

	"github.com/status-im/status-go/common"

	"go.uber.org/zap"
)

type CodexIndexDownloader struct {
	codexClient CodexClientInterface
	logger      *zap.Logger
}

func NewCodexIndexDownloader(codexClient CodexClientInterface, logger *zap.Logger) *CodexIndexDownloader {
	return &CodexIndexDownloader{
		codexClient: codexClient,
		logger:      logger,
	}
}

func (d *CodexIndexDownloader) DownloadIndexFileFromLocalNode(
	ctx context.Context,
	indexCid string,
	output io.Writer,
) error {
	defer common.LogOnPanic()

	d.logger.Debug("[CODEX][download_index_file_from_local_node] downloading codex index file from local node", zap.String("indexCid", indexCid))

	return d.codexClient.LocalDownloadWithContext(ctx, indexCid, output)
}

func (d *CodexIndexDownloader) DownloadIndexFileFromNetwork(
	ctx context.Context,
	indexCid string,
	output io.Writer,
) error {
	defer common.LogOnPanic()

	d.logger.Debug("[CODEX][download_index_file_from_network] downloading codex index file from network", zap.String("indexCid", indexCid))

	return d.codexClient.DownloadWithContext(ctx, indexCid, output)
}
