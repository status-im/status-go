package routes

type Route []*Path

func (r Route) Copy() Route {
	newRoute := make(Route, len(r))
	for i, path := range r {
		newRoute[i] = path.Copy()
	}
	return newRoute
}

// GetFirstPathChains returns the chain IDs (from and to) of the first path in the route
// If certain tx fails or succeeds status--go will send from/to chains along with other tx details to client to let
// thc client know about failed/successful tx. But if an error occurs before the first path, during the preparation
// of the route, the client will not have the chain IDs to know where the tx was supposed to be sent. This function
// is used to get the chain IDs of the first path in the route to send them to the client in case of an error.
func (r Route) GetFirstPathChains() (uint64, uint64) {
	if len(r) == 0 {
		return 0, 0
	}

	return r[0].FromChain.ChainID, r[0].ToChain.ChainID
}
