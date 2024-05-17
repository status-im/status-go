package params

// Define available fleets.
const (
	FleetUndefined     = ""
	FleetProd          = "eth.prod"
	FleetShardsStaging = "shards.staging"
	FleetShardsTest    = "shards.test"
	FleetStatusProd    = "status.prod"
	FleetStatusTest    = "status.test"
	FleetWakuSandbox   = "waku.sandbox"
	FleetWakuTest      = "waku.test"
)

// Cluster defines a list of Ethereum nodes.
type Cluster struct {
	StaticNodes     []string `json:"staticnodes"`
	BootNodes       []string `json:"bootnodes"`
	MailServers     []string `json:"mailservers"` // list of trusted mail servers
	RendezvousNodes []string `json:"rendezvousnodes"`
}

// DefaultWakuNodes is a list of "supported" fleets. This list is populated to clients UI settings.
var supportedFleets = map[string][]string{
	FleetShardsStaging: {"enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.shards.nodes.status.im"},
	FleetShardsTest:    {"enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.test.shards.nodes.status.im"},
	FleetWakuSandbox:   {"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im"},
	FleetWakuTest:      {"enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im"},
}

func DefaultWakuNodes(fleet string) []string {
	return supportedFleets[fleet]
}

func IsFleetSupported(fleet string) bool {
	_, ok := supportedFleets[fleet]
	return ok
}

func GetSupportedFleets() map[string][]string {
	return supportedFleets
}
