package params

// Define available fleets.
const (
	FleetUndefined = ""
	FleetBeta      = "eth.beta"
	FleetStaging   = "eth.staging"
)

// Cluster defines a list of Ethereum nodes.
type Cluster struct {
	StaticNodes     []string `json:"staticnodes"`
	BootNodes       []string `json:"bootnodes"`
	MailServers     []string `json:"mailservers"` // list of trusted mail servers
	RendezvousNodes []string `json:"rendezvousnodes"`
}
