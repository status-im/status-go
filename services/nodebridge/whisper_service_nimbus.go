// +build nimbus

package nodebridge

import (
	nimbussvc "github.com/status-im/status-go/services/nimbus"
)

// Make sure that WhisperService implements nimbussvc.Service interface.
var _ nimbussvc.Service = (*WhisperService)(nil)

func (w *WhisperService) StartService() error {
	return nil
}
