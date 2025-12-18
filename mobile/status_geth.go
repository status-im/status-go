package statusgo

import (
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/backend"
)

var statusBackend = backend.NewStatusBackend(logutils.ZapLogger())
