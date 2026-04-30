package puzzleauth

import "errors"

// ErrAuthRotating is returned by Transport when 401, 403, or 429 persist after all
// auth invalidation/refresh/retry attempts (e.g. during JWT rotation races).
// The health manager treats it as non-critical so the provider is not marked down.
var ErrAuthRotating = errors.New("puzzle auth: HTTP rejected credentials after token refresh; retry on next call")
