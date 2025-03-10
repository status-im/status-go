package params

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// Define available fleets.
const (
	FleetUndefined     = ""
	FleetProd          = "eth.prod"
	FleetStatusStaging = "status.staging"
	FleetStatusProd    = "status.prod"
	FleetWakuSandbox   = "waku.sandbox"
	FleetWakuTest      = "waku.test"
)

type FleetName = string
type NodeType = string
type FleetsMap = map[FleetName]map[NodeType][]string

type FleetInfo struct {
	Name                 FleetName `json:"name"`
	WakuNodes            []string  `json:"wakuNodes"`
	DiscV5BootstrapNodes []string  `json:"discV5BootstrapNodes"`
	ClusterID            uint16    `json:"clusterID"`
}
type FleetsMapV2 map[FleetName]FleetInfo

const (
	WakuNodes            NodeType = "WakuNodes"
	DiscV5BootstrapNodes NodeType = "DiscV5BootstrapNodes"
)

// DefaultWakuNodes is a list of "supported" fleets. This list is populated to clients UI settings.
var supportedFleets = FleetsMapV2{
	FleetStatusStaging: {
		WakuNodes: []string{
			"enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.status.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@boot.staging.status.nodes.status.im",
			"enr:-QEQuEBuiQgFlJNcv255042zwyl4pOBOivakX8N30Dr9vaaEU2q8-7N4GVY4Hk87iEKELjlIXTpE9Wj6EQq1lrBuc7ayAYJpZIJ2NIJpcISPxvrpim11bHRpYWRkcnO4YAAtNihib290LTAxLmRvLWFtczMuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfAC82KGJvb3QtMDEuZG8tYW1zMy5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDq-yGgpuoUG6NKkbIDRmrMiT-bEVzFlpWLEK_rF3yKUaDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEiuED2UusuHo1d6WN2-tHjtj0T0gdnsOh7aRZnFF6OEYLDbyxOtQo2_4dFUHhc9xm5SHNrWJJq8X7FRsxc4VCMGjjbAYJpZIJ2NIJpcIRoxQVgim11bHRpYWRkcnO4cgA2NjFib290LTAxLmdjLXVzLWNlbnRyYWwxLWEuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfADg2MWJvb3QtMDEuZ2MtdXMtY2VudHJhbDEtYS5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDNAvlGjekD1YV4WpmjwArGAH2g9kHFJnMRfgUhcIkoA2DdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEiuECJPv2vL00Jp5sTEMAFyW7qXkK2cFgphlU_G8-FJuJqoW_D5aWIy3ylGdv2K8DkiG7PWgng4Ql_VI7Qc2RhBdwfAYJpZIJ2NIJpcIQvTKi6im11bHRpYWRkcnO4cgA2NjFib290LTAxLmFjLWNuLWhvbmdrb25nLWMuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfADg2MWJvb3QtMDEuYWMtY24taG9uZ2tvbmctYy5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDkbgV7oqPNmFtX5FzSPi9WH8kkmrPB1R3n9xRXge91M-DdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
		},
		ClusterID: 16,
	},
	FleetStatusProd: {
		WakuNodes: []string{
			"enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
			"enr:-QEKuED9AJm2HGgrRpVaJY2nj68ao_QiPeUT43sK-aRM7sMJ6R4G11OSDOwnvVacgN1sTw-K7soC5dzHDFZgZkHU0u-XAYJpZIJ2NIJpcISnYxMvim11bHRpYWRkcnO4WgAqNiVib290LTAxLmRvLWFtczMuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfACw2JWJvb3QtMDEuZG8tYW1zMy5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEC3rRtFQSgc24uWewzXaxTY8hDAHB8sgnxr9k8Rjb5GeSDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEcuED7ww5vo2rKc1pyBp7fubBUH-8STHEZHo7InjVjLblEVyDGkjdTI9VdqmYQOn95vuQH-Htku17WSTzEufx-Wg4mAYJpZIJ2NIJpcIQihw1Xim11bHRpYWRkcnO4bAAzNi5ib290LTAxLmdjLXVzLWNlbnRyYWwxLWEuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfADU2LmJvb3QtMDEuZ2MtdXMtY2VudHJhbDEtYS5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaECxjqgDQ0WyRSOilYU32DA5k_XNlDis3m1VdXkK9xM6kODdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
			"enr:-QEcuEAoShWGyN66wwusE3Ri8hXBaIkoHZHybUB8cCPv5v3ypEf9OCg4cfslJxZFANl90s-jmMOugLUyBx4EfOBNJ6_VAYJpZIJ2NIJpcIQI2hdMim11bHRpYWRkcnO4bAAzNi5ib290LTAxLmFjLWNuLWhvbmdrb25nLWMuc3RhdHVzLnByb2Quc3RhdHVzLmltBnZfADU2LmJvb3QtMDEuYWMtY24taG9uZ2tvbmctYy5zdGF0dXMucHJvZC5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDP7CbRk-YKJwOFFM4Z9ney0GPc7WPJaCwGkpNRyla7mCDdGNwgnZfg3VkcIIjKIV3YWt1Mg0",
		},
		ClusterID: 16,
	},
	FleetWakuSandbox: {
		WakuNodes: []string{
			"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im",
		},
	},
	FleetWakuTest: {
		WakuNodes: []string{
			"enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im",
		},
		DiscV5BootstrapNodes: []string{
			"enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im",
		},
	},
}

var DefaultPushNotificationServers = []string{
	"401ba5eda402678dc78a0a40fd0795f4ea8b1e34972c4d15cf33ac01292341c89f0cbc637fa9f7a3ffe0b9dfe90e9cdae7a14925500ab01b6a91c67bae42a97a",
	"181141b1d111908aaf05f4788e6778ec07073a1d4e1ce43c73815c40ee4e7345a1cbf5a90a45f601bf3763f12be63b01624ba1f36eeb9572455e7034b8f9f2c4",
	"5ffc34d5ffda180d94cd3974d9ed2bb082ede68f342babdbe801ceffb7da902087d43f9aa961c7b85029358874c08ef04ecad9f1d95a1f0e448cbdd5d04350c7",
}

func init() {
	if fleetsOverride := os.Getenv(envFleetsFilePath); fleetsOverride != "" {
		supportedFleets = loadFleetsFromFile(fleetsOverride)
	}
}

func loadFleetsFromFile(filepath string) FleetsMapV2 {
	// Read the JSON file to populate the supportedFleets map
	file, err := os.Open(filepath)
	if err != nil {
		err = errors.Wrap(err, "failed to open fleets json file")
		panic(err)
	}

	defer file.Close()

	var overrideFleets FleetsMapV2
	decoder := json.NewDecoder(file)

	err = decoder.Decode(&overrideFleets)
	if err != nil {
		err = errors.Wrap(err, "failed to decode fleets json file")
		panic(err)
	}

	return overrideFleets
}

func DefaultWakuNodes(fleet string) []string {
	return supportedFleets[fleet].WakuNodes
}

func DefaultDiscV5Nodes(fleet string) []string {
	return supportedFleets[fleet].DiscV5BootstrapNodes
}

func DefaultClusterID(fleet string) uint16 {
	return supportedFleets[fleet].ClusterID
}

func IsFleetSupported(fleet string) bool {
	_, ok := supportedFleets[fleet]
	return ok
}

func GetSupportedFleets() FleetsMap {
	var oldFleetMap = make(FleetsMap)

	for fleet, nodes := range supportedFleets {
		oldFleetMap[fleet] = map[string][]string{
			WakuNodes:            nodes.WakuNodes,
			DiscV5BootstrapNodes: nodes.DiscV5BootstrapNodes,
		}
	}

	return oldFleetMap
}
