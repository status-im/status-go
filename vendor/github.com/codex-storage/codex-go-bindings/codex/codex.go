package codex

/*
   #include "bridge.h"
   #include <stdlib.h>

   void libcodexNimMain(void);

   static void codex_host_init_once(void){
       static int done;
       if (!__atomic_exchange_n(&done, 1, __ATOMIC_SEQ_CST)) libcodexNimMain();
   }

   // resp must be set != NULL in case interest on retrieving data from the callback
   void callback(int ret, char* msg, size_t len, void* resp);

   static void* cGoCodexNew(const char* configJson, void* resp) {
       void* ret = codex_new(configJson, (CodexCallback) callback, resp);
       return ret;
   }

   static int cGoCodexStart(void* codexCtx, void* resp) {
       return codex_start(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexStop(void* codexCtx, void* resp) {
       return codex_stop(codexCtx, (CodexCallback) callback, resp);
   }

	static int cGoCodexClose(void* codexCtx, void* resp) {
		return codex_close(codexCtx, (CodexCallback) callback, resp);
	}

   static int cGoCodexDestroy(void* codexCtx, void* resp) {
       return codex_destroy(codexCtx, (CodexCallback) callback, resp);
   }

    static int cGoCodexVersion(void* codexCtx, void* resp) {
       return codex_version(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexRevision(void* codexCtx, void* resp) {
       return codex_revision(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexRepo(void* codexCtx, void* resp) {
       return codex_repo(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexSpr(void* codexCtx, void* resp) {
       return codex_spr(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexPeerId(void* codexCtx, void* resp) {
       return codex_peer_id(codexCtx, (CodexCallback) callback, resp);
   }
*/
import "C"
import (
	"encoding/json"
	"unsafe"
)

type LogLevel string

const (
	TRACE  LogLevel = "trace"
	DEBUG  LogLevel = "debug"
	INFO   LogLevel = "info"
	NOTICE LogLevel = "notice"
	WARN   LogLevel = "warn"
	ERROR  LogLevel = "error"
	FATAL  LogLevel = "fatal"
)

type LogFormat string

const (
	LogFormatAuto     LogFormat = "auto"
	LogFormatColors   LogFormat = "colors"
	LogFormatNoColors LogFormat = "nocolors"
	LogFormatJSON     LogFormat = "json"
)

type RepoKind string

const (
	FS      RepoKind = "fs"
	SQLite  RepoKind = "sqlite"
	LevelDb RepoKind = "leveldb"
)

type Config struct {
	// Default: INFO
	LogLevel string `json:"log-level,omitempty"`

	// Specifies what kind of logs should be written to stdout
	// Default: auto
	LogFormat LogFormat `json:"log-format,omitempty"`

	// Enable the metrics server
	// Default: false
	MetricsEnabled bool `json:"metrics,omitempty"`

	// Listening address of the metrics server
	// Default: 127.0.0.1
	MetricsAddress string `json:"metrics-address,omitempty"`

	// Listening HTTP port of the metrics server
	// Default: 8008
	MetricsPort int `json:"metrics-port,omitempty"`

	// The directory where codex will store configuration and data
	// Default:
	// $HOME\AppData\Roaming\Codex on Windows
	// $HOME/Library/Application Support/Codex on macOS
	// $HOME/.cache/codex on Linux
	DataDir string `json:"data-dir,omitempty"`

	// Multi Addresses to listen on
	// Default: ["/ip4/0.0.0.0/tcp/0"]
	ListenAddrs []string `json:"listen-addrs,omitempty"`

	// Specify method to use for determining public address.
	// Must be one of: any, none, upnp, pmp, extip:<IP>
	// Default: any
	Nat string `json:"nat,omitempty"`

	// Discovery (UDP) port
	// Default: 8090
	DiscoveryPort int `json:"disc-port,omitempty"`

	// Source of network (secp256k1) private key file path or name
	// Default: "key"
	NetPrivKeyFile string `json:"net-privkey,omitempty"`

	// Specifies one or more bootstrap nodes to use when connecting to the network.
	BootstrapNodes []string `json:"bootstrap-node,omitempty"`

	// The maximum number of peers to connect to.
	// Default: 160
	MaxPeers int `json:"max-peers,omitempty"`

	// Number of worker threads (\"0\" = use as many threads as there are CPU cores available)
	// Default: 0
	NumThreads int `json:"num-threads,omitempty"`

	// Node agent string which is used as identifier in network
	// Default: "Codex"
	AgentString string `json:"agent-string,omitempty"`

	// Backend for main repo store (fs, sqlite, leveldb)
	// Default: fs
	RepoKind RepoKind `json:"repo-kind,omitempty"`

	// The size of the total storage quota dedicated to the node
	// Default: 20 GiBs
	StorageQuota int `json:"storage-quota,omitempty"`

	// Default block timeout in seconds - 0 disables the ttl
	// Default: 30 days
	BlockTtl int `json:"block-ttl,omitempty"`

	// Time interval in seconds - determines frequency of block
	// maintenance cycle: how often blocks are checked for expiration and cleanup
	// Default: 10 minutes
	BlockMaintenanceInterval int `json:"block-mi,omitempty"`

	// Number of blocks to check every maintenance cycle
	// Default: 1000
	BlockMaintenanceNumberOfBlocks int `json:"block-mn,omitempty"`

	// Number of times to retry fetching a block before giving up
	// Default: 3000
	BlockRetries int `json:"block-retries,omitempty"`

	// The size of the block cache, 0 disables the cache -
	// might help on slow hardrives
	// Default: 0
	CacheSize int `json:"cache-size,omitempty"`

	// Default: "" (no log file)
	LogFile string `json:"log-file,omitempty"`
}

type CodexNode struct {
	ctx unsafe.Pointer
}

type ChunkSize int

func (c ChunkSize) valOrDefault() int {
	if c == 0 {
		return defaultBlockSize
	}

	return int(c)
}

func (c ChunkSize) toSizeT() C.size_t {
	return C.size_t(c.valOrDefault())
}

// New creates a new Codex node with the provided configuration.
// The node is not started automatically; you need to call CodexStart
// to start it.
// It returns a Codex node that can be used to interact
// with the Codex network.
func New(config Config) (*CodexNode, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	cJsonConfig := C.CString(string(jsonConfig))
	defer C.free(unsafe.Pointer(cJsonConfig))

	ctx := C.cGoCodexNew(cJsonConfig, bridge.resp)

	if _, err := bridge.wait(); err != nil {
		return nil, bridge.err
	}

	return &CodexNode{ctx: ctx}, bridge.err
}

// Start starts the Codex node.
func (node CodexNode) Start() error {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexStart(node.ctx, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexStart")
	}

	_, err := bridge.wait()
	return err
}

// StartAsync is the asynchronous version of Start.
func (node CodexNode) StartAsync(onDone func(error)) {
	go func() {
		err := node.Start()
		onDone(err)
	}()
}

// Stop stops the Codex node.
func (node CodexNode) Stop() error {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexStop(node.ctx, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexStop")
	}

	_, err := bridge.wait()
	return err
}

// Destroy destroys the Codex node, freeing all resources.
// The node must be stopped before calling this method.
func (node CodexNode) Destroy() error {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexClose(node.ctx, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexClose")
	}

	_, err := bridge.wait()
	if err != nil {
		return err
	}

	if C.cGoCodexDestroy(node.ctx, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexDestroy")
	}

	// We don't wait for the bridge here.
	// The destroy function does not call the worker thread,
	// it destroys the context directly and return the return
	// value synchronously.

	return nil
}

// Version returns the version of the Codex node.
func (node CodexNode) Version() (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexVersion(node.ctx, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexVersion")
	}

	return bridge.wait()
}

func (node CodexNode) Revision() (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexRevision(node.ctx, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexRevision")
	}

	return bridge.wait()
}

// Repo returns the path of the data dir folder.
func (node CodexNode) Repo() (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexRepo(node.ctx, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexRepo")
	}

	return bridge.wait()
}

func (node CodexNode) Spr() (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexSpr(node.ctx, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexSpr")
	}

	return bridge.wait()
}

func (node CodexNode) PeerId() (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexPeerId(node.ctx, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexPeerId")
	}

	return bridge.wait()
}
