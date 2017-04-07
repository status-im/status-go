package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the main Ethereum network.
var MainnetBootnodes = []string{
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the Ropsten test network.
var TestnetBootnodes = []string{
	"enode://a3287fce9c98a2cfe5753362cbd43b307abf24295802b237eb3c6526ab1be8b0a5348896a55aa0683e38f35d2ed33b5ee0af879c07fc87ac0f3e0455c8426806@123.207.28.44:30303",
	"enode://9b8f5d287ecc763226cac1344b3de2918599bdbd6df03cb564fe7b7580d40cf8446681c31809cacbba4a5955d01035db93dc175b84e26ed762bb100f337f3cb1@172.31.102.179:30303",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
}
