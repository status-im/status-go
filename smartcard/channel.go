package smartcard

import "github.com/status-im/status-go/smartcard/apdu"

type Channel interface {
	Send(*apdu.Command) (*apdu.Response, error)
}
