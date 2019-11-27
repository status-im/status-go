// +build nimbus

package accounts

import (
	nimbussvc "github.com/status-im/status-go/services/nimbus"
)

// Make sure that Service implements nimbussvc.Service interface.
var _ nimbussvc.Service = (*Service)(nil)

// Start a service.
func (s *Service) StartService() error {
	return nil
}
