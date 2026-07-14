//go:build mobile_pprof

package statusgo

import (
	"net/http"
	_ "net/http/pprof" // registers pprof handlers on http.DefaultServeMux

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	logutils "github.com/status-im/status-go/internal/logutils"
)

// When built with the `mobile_pprof` tag, expose Go's net/http/pprof endpoints
// on a loopback-only listener so the Android service process can be CPU-profiled
// on a device (e.g. via `adb forward tcp:6060 tcp:6060`).
func init() {
	go func() {
		defer common.LogOnPanic()
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			logutils.ZapLogger().Error("mobile pprof listener stopped", zap.Error(err))
		}
	}()
}
