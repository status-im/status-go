package adapters

import (
	"github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func FromWakuTransportStats(wakuStats wakutypes.StatsSummary) types.TransportStats {
	return types.TransportStats{
		UploadRate:   wakuStats.UploadRate,
		DownloadRate: wakuStats.DownloadRate,
	}
}
