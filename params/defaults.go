package params

const (
	ArchivesRelativePath        = "data/archivedata"
	TorrentTorrentsRelativePath = "data/torrents"

	// MainnetEthereumNetworkURL is URL where the upstream ethereum network is loaded to
	// allow us avoid syncing node.
	MainnetEthereumNetworkURL = "https://mainnet.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	// MainNetworkID is id of the main network
	MainNetworkID = 1

	// SepoliaNetworkID is id of sepolia test network
	SepoliaNetworkID = 11155111

	// StatusChainNetworkID is id of a test network (private chain)
	StatusChainNetworkID = 777

	// IpfsGatewayURL is the Gateway URL to use for IPFS
	IpfsGatewayURL = "https://ipfs.status.im/"

	// Number of times to retry fetching a block on Codex before giving up
	BlockRetries = 50
)
