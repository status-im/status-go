package rpc

// router implements logic for routing
// JSON-RPC requests either to Upstream or
// Local node.
type router struct {
	methods map[string]bool
}

// newRouter inits new router.
func newRouter() *router {
	r := &router{
		methods: make(map[string]bool),
	}

	for _, m := range localMethods {
		r.methods[m] = true
	}

	return r
}

// isLocal returns true if given method should be routed to
// the local node
func (r *router) isLocal(method string) bool {
	return r.methods[method]
}

// localMethods contains methods that should be routed to
// the local node; the rest is considered to be routed to
// the upstream node.
var localMethods = [...]string{
	//Whisper commands
	"shh_post",
	"shh_version",
	"shh_newIdentity",
	"shh_hasIdentity",
	"shh_newGroup",
	"shh_addToGroup",
	"shh_newFilter",
	"shh_uninstallFilter",
	"shh_getFilterChanges",
	"shh_getMessages",

	// DB commands
	"db_putString",
	"db_getString",
	"db_putHex",
	"db_getHex",

	// Other commands
	"net_version",
	"net_peerCount",
	"net_listening",

	// blockchain commands
	"eth_sign",
	"eth_accounts",
	"eth_getCompilers",
	"eth_compileLLL",
	"eth_compileSolidity",
	"eth_compileSerpent",
}
