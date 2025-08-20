package enr

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func MustDecode(enrStr string) *enode.Node {
	node, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil || node == nil {
		panic("could not decode enr: " + enrStr)
	}
	return node
}
