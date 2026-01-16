package networkhelper

import (
	communitytokendeployer "github.com/status-im/status-go/internal/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/params"
)

// WithCommunitiesSupported returns a copy of the given networks slice with CommunitiesSupported
// set based on CommunitiesSupportedOnChain(chainID).
func WithCommunitiesSupported(networks []params.Network) []params.Network {
	updated := DeepCopyNetworks(networks)
	for i := range updated {
		updated[i].CommunitiesSupported = communitytokendeployer.CommunitiesSupportedOnChain(updated[i].ChainID)
	}
	return updated
}

// ApplyCommunitiesSupported sets CommunitiesSupported in-place for each network.
func ApplyCommunitiesSupported(networks []*params.Network) {
	for _, n := range networks {
		if n == nil {
			continue
		}
		n.CommunitiesSupported = communitytokendeployer.CommunitiesSupportedOnChain(n.ChainID)
	}
}
