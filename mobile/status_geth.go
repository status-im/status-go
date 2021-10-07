package statusgo

import (
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/services/wakuext"
)

var statusBackend = api.NewGethStatusBackend()

func GetExtAPI() *wakuext.PublicAPI {
	// TODO: this should be replaced by a function that determines which WakuVersion is active
	// and returns the proper ext.PublicAPI object
	return statusBackend.StatusNode().WakuExtService().APIs()[0].Service.(*wakuext.PublicAPI)
}
