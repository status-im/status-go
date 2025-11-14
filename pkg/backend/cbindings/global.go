package cbindings

import (
	"github.com/status-im/status-go/pkg/backend"
)

// globalBackendInstance is the global StatusBackend instance.
// It must only be used from C-bindings defined in this file.
// `api` package is deliberately kept as a separate package from `backend`
// to avoid accidental usage of globalBackendInstance outside C-bindings.
var globalBackendInstance *backend.StatusBackend
