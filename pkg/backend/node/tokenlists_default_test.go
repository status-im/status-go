//go:build !tkl

package node

import "testing"

func TestStatusNodeStopAfterUnsupportedNimCatalogue(t *testing.T) {
	testStatusNodeStopAfterCatalogueSetupFailure(t, true)
}
