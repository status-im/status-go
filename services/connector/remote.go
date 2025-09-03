package connector

// dappAllowedRemoteMethods contains methods that should be routed to
// the external EthClient; the rest is considered as not allowed
var dappAllowedRemoteMethods = [...]string{
	"eth_protocolVersion",
	"eth_syncing",
	"eth_coinbase",
	"eth_mining",
	"eth_hashrate",
	"eth_gasPrice",
	"eth_maxPriorityFeePerGas",
	"eth_feeHistory",
	//"eth_accounts", // due to sub-accounts handling
	"eth_blockNumber",
	"eth_getBalance",
	"eth_getStorageAt",
	"eth_getTransactionCount",
	"eth_getBlockTransactionCountByHash",
	"eth_getBlockTransactionCountByNumber",
	"eth_getUncleCountByBlockHash",
	"eth_getUncleCountByBlockNumber",
	"eth_getCode",
	//"eth_sign", // only the local node has an injected account to sign the payload with
	//"eth_sendTransaction", // we handle this specially calling eth_estimateGas, signing it locally and sending eth_sendRawTransaction afterwards
	"eth_sendRawTransaction",
	"eth_call",
	"eth_estimateGas",
	"linea_estimateGas", // Status chain specific gas estimation and tx gas amount calculation method
	"eth_getBlockByHash",
	"eth_getBlockByNumber",
	"eth_getTransactionByHash",
	"eth_getTransactionByBlockHashAndIndex",
	"eth_getTransactionByBlockNumberAndIndex",
	"eth_getTransactionReceipt",
	"eth_getUncleByBlockHashAndIndex",
	"eth_getUncleByBlockNumberAndIndex",
	//"eth_getCompilers",    // goes to the local because there's no need to send it anywhere
	//"eth_compileLLL",      // goes to the local because there's no need to send it anywhere
	//"eth_compileSolidity", // goes to the local because there's no need to send it anywhere
	//"eth_compileSerpent",  // goes to the local because there's no need to send it anywhere

	"eth_getLogs",
	"eth_getWork",
	"eth_submitWork",
	"eth_submitHashrate",
	"net_version",
	"net_peerCount",
	"net_listening",
}

var dappAllowedRemoteMethodsSet = map[string]struct{}{}

func init() {
	for _, method := range dappAllowedRemoteMethods {
		dappAllowedRemoteMethodsSet[method] = struct{}{}
	}
}

func dappRemoteMethodIsAllowed(method string) bool {
	_, ok := dappAllowedRemoteMethodsSet[method]
	return ok
}
