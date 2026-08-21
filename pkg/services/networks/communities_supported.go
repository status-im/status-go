package networks

import (
	communitytokendeployer "github.com/status-im/status-go/internal/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/params"
)

// withCommunitiesSupported returns a copy of the given networks slice with CommunitiesSupported
// set based on CommunitiesSupportedOnChain(chainID).
func withCommunitiesSupported(networks []params.Network) []params.Network {
	updated := deepCopyNetworks(networks)
	for i := range updated {
		updated[i].CommunitiesSupported = communitytokendeployer.CommunitiesSupportedOnChain(updated[i].ChainID)
	}
	return updated
}

// applyCommunitiesSupported sets CommunitiesSupported in-place for each network.
func applyCommunitiesSupported(networks []*params.Network) {
	for _, n := range networks {
		if n == nil {
			continue
		}
		n.CommunitiesSupported = communitytokendeployer.CommunitiesSupportedOnChain(n.ChainID)
	}
}
