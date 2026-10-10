package messaging

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/waku"
)

func newGoWakuTestBackend() (testBackend, error) {
	return waku.New(nil, &waku.DefaultConfig, zap.NewNop(), &testTimeSource{})
}
