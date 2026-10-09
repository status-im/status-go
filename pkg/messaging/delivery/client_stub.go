//go:build !logos_delivery

package delivery

import "errors"

// Available reports whether this build links liblogosdelivery.
const Available = false

// ErrNotAvailable is returned by NewClient in builds without the
// logos_delivery tag.
var ErrNotAvailable = errors.New("delivery: built without the logos_delivery tag")

// NewClient fails: this build does not link liblogosdelivery.
func NewClient(Config) (Client, error) {
	return nil, ErrNotAvailable
}
