package statusgo

import (
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/backend"
)

var statusBackend = backend.NewStatusBackend(logutils.ZapLogger())

// runAsync runs an accepted login. Tests replace it to observe a login mid-flight.
var runAsync = backend.RunAsync
