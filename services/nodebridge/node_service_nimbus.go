// +build nimbus

package nodebridge

import (
	nimbussvc "github.com/status-im/status-go/services/nimbus"
)

// Make sure that NodeService implements nimbussvc.Service interface.
var _ nimbussvc.Service = (*NodeService)(nil)

func (w *NodeService) StartService() error {
	return nil
}
