package params

// Define available fleets.
const (
	FleetUndefined = ""
	FleetProd      = "eth.prod"
	FleetStaging   = "eth.staging"
	FleetTest      = "eth.test"
)

// Cluster defines a list of Ethereum nodes.
type Cluster struct {
	StaticNodes     []string `json:"staticnodes"`
	BootNodes       []string `json:"bootnodes"`
	MailServers     []string `json:"mailservers"` // list of trusted mail servers
	RendezvousNodes []string `json:"rendezvousnodes"`
}
