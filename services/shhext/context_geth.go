// +build !nimbus

package shhext

import (
	"context"

	"github.com/status-im/status-go/db"
)

// NewContextFromService creates new context instance using Service fileds directly and Storage.
func NewContextFromService(ctx context.Context, service *Service, storage db.Storage) Context {
	return NewContext(ctx, service.w.GetCurrentTime, service.requestsRegistry, storage)
}
