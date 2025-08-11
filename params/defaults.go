package params

const (
	ArchivesRelativePath        = "data/archivedata"
	TorrentTorrentsRelativePath = "data/torrents"

	// SendTransactionMethodName https://docs.walletconnect.com/advanced/rpc-reference/ethereum-rpc#eth_sendtransaction
	SendTransactionMethodName = "eth_sendTransaction"

	BalanceMethodName = "eth_getBalance"

	// AccountsMethodName defines the name for listing the currently signed accounts.
	AccountsMethodName = "eth_accounts"

	// PersonalSignMethodName https://docs.walletconnect.com/advanced/rpc-reference/ethereum-rpc#personal_sign
	PersonalSignMethodName = "personal_sign"

	// PersonalRecoverMethodName defines the name for `personal.recover` API.
	PersonalRecoverMethodName = "personal_ecRecover"

	// MainnetEthereumNetworkURL is URL where the upstream ethereum network is loaded to
	// allow us avoid syncing node.
	MainnetEthereumNetworkURL = "https://mainnet.infura.io/nKmXgiFgc2KqtoQ8BCGJ"

	// SepoliaEthereumNetworkURL is an open RPC endpoint to Sepolia network
	SepoliaEthereumNetworkURL = "https://sepolia.etherscan.io/"

	// MainNetworkID is id of the main network
	MainNetworkID = 1

	// SepoliaNetworkID is id of sepolia test network
	SepoliaNetworkID = 11155111

	// StatusChainNetworkID is id of a test network (private chain)
	StatusChainNetworkID = 777

	// LESDiscoveryIdentifier is a prefix for topic used for LES peers discovery.
	LESDiscoveryIdentifier = "LES2@"

	// IpfsGatewayURL is the Gateway URL to use for IPFS
	IpfsGatewayURL = "https://ipfs.status.im/"
)
