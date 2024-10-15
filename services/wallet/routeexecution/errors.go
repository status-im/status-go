package routeexecution

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `W` for the error code stands for Wallet
var (
	ErrCannotResolveRouteId = &errors.ErrorResponse{Code: errors.ErrorCode("W-001"), Details: "cannot resolve route id"}
)
