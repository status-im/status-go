package datasync

import (
	datasyncnode "github.com/vacp2p/mvds/node"
)

type DataSync struct {
	*datasyncnode.Node
	// DataSyncNodeTransport is the implemntation of the datasync transport interface
	*DataSyncNodeTransport
}
