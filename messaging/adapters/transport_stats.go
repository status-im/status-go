package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func FromWakuTransportStats(wakuStats wakutypes.StatsSummary) types.TransportStats {
	return types.TransportStats{
		UploadRate:   wakuStats.UploadRate,
		DownloadRate: wakuStats.DownloadRate,
	}
}
