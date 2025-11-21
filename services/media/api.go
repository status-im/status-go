package media

import (
	"github.com/status-im/status-go/server"
)

type API struct {
}

func (a *API) TLSCertificate() (string, error) {
	// TODO: Don't use a global certificate
	return server.PublicMediaTLSCert()
}
