// +build nimbus

package status

import (
	nimbussvc "github.com/status-im/status-go/services/nimbus"
)

// Make sure that Service implements nimbussvc.Service interface.
var _ nimbussvc.Service = (*Service)(nil)

// StartService is run when a service is started.
// It does nothing in this case but is required by `nimbussvc.Service` interface.
func (s *Service) StartService() error {
	return nil
}
