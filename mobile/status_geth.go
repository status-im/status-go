package statusgo

import (
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
)

var statusBackend = api.NewGethStatusBackend(logutils.ZapLogger())
