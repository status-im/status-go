package statusgo

import (
	"github.com/status-im/status-go/v10/api"
	"github.com/status-im/status-go/v10/logutils"
)

var statusBackend = api.NewGethStatusBackend(logutils.ZapLogger())
