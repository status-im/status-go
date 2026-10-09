//go:build !logos_delivery

package messaging

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/waku"
)

func newTestBackend() (testBackend, error) {
	return waku.New(nil, &waku.DefaultConfig, zap.NewNop(), &testTimeSource{})
}
