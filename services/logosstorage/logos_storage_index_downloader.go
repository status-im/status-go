//go:build use_logos_storage

package logosstorage

import (
	"context"
	"io"

	"github.com/status-im/status-go/common"

	"go.uber.org/zap"
)

type LogosStorageIndexDownloader struct {
	logosStorageClient LogosStorageClientInterface
	logger             *zap.Logger
}

func NewLogosStorageIndexDownloader(logosStorageClient LogosStorageClientInterface, logger *zap.Logger) *LogosStorageIndexDownloader {
	return &LogosStorageIndexDownloader{
		logosStorageClient: logosStorageClient,
		logger:             logger,
	}
}

func (d *LogosStorageIndexDownloader) DownloadIndexFileFromLocalNode(
	ctx context.Context,
	indexCid string,
	output io.Writer,
) error {
	defer common.LogOnPanic()

	d.logger.Debug("[LOGOS_STORAGE][download_index_file_from_local_node] downloading LogosStorage index file from local node", zap.String("indexCid", indexCid))

	return d.logosStorageClient.LocalDownloadWithContext(ctx, indexCid, output)
}

func (d *LogosStorageIndexDownloader) DownloadIndexFileFromNetwork(
	ctx context.Context,
	indexCid string,
	output io.Writer,
) error {
	defer common.LogOnPanic()

	d.logger.Debug("[LogosStorage][download_index_file_from_network] downloading LogosStorage index file from network", zap.String("indexCid", indexCid))

	return d.logosStorageClient.DownloadWithContext(ctx, indexCid, output)
}
