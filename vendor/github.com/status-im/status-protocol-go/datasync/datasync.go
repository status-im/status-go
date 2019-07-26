package datasync

import (
	datasyncnode "github.com/vacp2p/mvds/node"
)

type DataSync struct {
	*datasyncnode.Node
	*DataSyncNodeTransport
}
