package versioning

type ConnectionParamVersion int

const (
	ConnectionParamsV1 ConnectionParamVersion = iota + 1
)

type LocalPairingVersion int

const (
	LocalPairingV1 LocalPairingVersion = iota + 1
)

const (
	LatestConnectionParamVer = ConnectionParamsV1
	LatestLocalPairingVer    = LocalPairingV1
)
