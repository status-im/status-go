package requests

import (
	"strings"

	"github.com/multiformats/go-multiaddr"
)

type SaveNewWakuNode struct {
	NodeAddress string `json:"nodeAddress"`
}

func (r *SaveNewWakuNode) Validate() error {
	if strings.HasPrefix(r.NodeAddress, "enrtree://") {
		return nil
	}

	// It is a normal multiaddress
	_, err := multiaddr.NewMultiaddr(r.NodeAddress)
	if err != nil {
		return err
	}

	return nil
}
