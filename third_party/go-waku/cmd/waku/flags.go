package main

import (
	"time"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"github.com/waku-org/go-waku/waku/cliutils"
	"github.com/waku-org/go-waku/waku/v2/node"
)

var (
	TcpPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "tcp-port",
		Aliases:     []string{"port", "p"},
		Value:       60000,
		Usage:       "Libp2p TCP listening port (0 for random)",
		Destination: &options.Port,
		EnvVars:     []string{"WAKUNODE2_TCP_PORT"},
	})
	Address = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "address",
		Aliases:     []string{"host", "listen-address"},
		Value:       "0.0.0.0",
		Usage:       "Listening address",
		Destination: &options.Address,
		EnvVars:     []string{"WAKUNODE2_ADDRESS"},
	})
	MaxPeerConnections = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "max-connections",
		Value:       50,
		Usage:       "Maximum allowed number of libp2p connections.",
		Destination: &options.MaxPeerConnections,
		EnvVars:     []string{"WAKUNODE2_MAX_CONNECTIONS"},
	})
	PeerStoreCapacity = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "peer-store-capacity",
		Usage:       "Maximum stored peers in the peerstore.",
		Destination: &options.PeerStoreCapacity,
		EnvVars:     []string{"WAKUNODE2_PEERSTORE_CAPACITY"},
	})
	WebsocketSupport = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "websocket-support",
		Aliases:     []string{"ws"},
		Usage:       "Enable websockets support",
		Destination: &options.Websocket.Enable,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_SUPPORT"},
	})
	WebsocketPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "websocket-port",
		Aliases:     []string{"ws-port"},
		Value:       60001,
		Usage:       "Libp2p TCP listening port for websocket connection (0 for random)",
		Destination: &options.Websocket.WSPort,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_PORT"},
	})
	WebsocketSecurePort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "websocket-secure-port",
		Aliases:     []string{"wss-port"},
		Value:       6443,
		Usage:       "Libp2p TCP listening port for secure websocket connection (0 for random, binding to 443 requires root access)",
		Destination: &options.Websocket.WSSPort,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_SECURE_PORT"},
	})
	WebsocketAddress = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "websocket-address",
		Aliases:     []string{"ws-address"},
		Value:       "0.0.0.0",
		Usage:       "Listening address for websocket connections",
		Destination: &options.Websocket.Address,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_ADDRESS"},
	})
	WebsocketSecureSupport = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "websocket-secure-support",
		Aliases:     []string{"wss"},
		Usage:       "Enable secure websockets support",
		Destination: &options.Websocket.Secure,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_SECURE_SUPPORT"},
	})
	WebsocketSecureKeyPath = altsrc.NewPathFlag(&cli.PathFlag{
		Name:        "websocket-secure-key-path",
		Aliases:     []string{"wss-key"},
		Value:       "/path/to/key.txt",
		Usage:       "Secure websocket key path",
		Destination: &options.Websocket.KeyPath,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_SECURE_KEY_PATH"},
	})
	WebsocketSecureCertPath = altsrc.NewPathFlag(&cli.PathFlag{
		Name:        "websocket-secure-cert-path",
		Aliases:     []string{"wss-cert"},
		Value:       "/path/to/cert.txt",
		Usage:       "Secure websocket certificate path",
		Destination: &options.Websocket.CertPath,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_SECURE_CERT_PATH"},
	})
	DNS4DomainName = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "dns4-domain-name",
		Value:       "",
		Usage:       "The domain name resolving to the node's public IPv4 address",
		Destination: &options.DNS4DomainName,
		EnvVars:     []string{"WAKUNODE2_WEBSOCKET_DNS4_DOMAIN_NAME"},
	})
	NodeKey = cliutils.NewGenericFlagSingleValue(&cli.GenericFlag{
		Name:  "nodekey",
		Usage: "P2P node private key as hex.",
		Value: &cliutils.PrivateKeyValue{
			Value: &options.NodeKey,
		},
		EnvVars: []string{"WAKUNODE2_NODEKEY", "GOWAKU-NODEKEY"},
	})
	KeyFile = altsrc.NewPathFlag(&cli.PathFlag{
		Name:        "key-file",
		Value:       "./nodekey",
		Usage:       "Path to a file containing the private key for the P2P node",
		Destination: &options.KeyFile,
		EnvVars:     []string{"WAKUNODE2_KEY_FILE"},
	})
	KeyPassword = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "key-password",
		Value:       "secret",
		Usage:       "Password used for the private key file",
		Destination: &options.KeyPasswd,
		EnvVars:     []string{"WAKUNODE2_KEY_PASSWORD"},
	})
	ClusterID = altsrc.NewUintFlag(&cli.UintFlag{
		Name:        "cluster-id",
		Value:       0,
		Usage:       "Cluster id that the node is running in. Node in a different cluster id is disconnected.",
		Destination: &options.ClusterID,
		EnvVars:     []string{"WAKUNODE2_CLUSTER_ID"},
	})
	StaticNode = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "staticnode",
		Usage: "Multiaddr of peer to directly connect with. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.StaticNodes,
		},
		EnvVars: []string{"WAKUNODE2_STATICNODE"},
	})
	KeepAlive = altsrc.NewDurationFlag(&cli.DurationFlag{
		Name:        "keep-alive",
		Value:       5 * time.Minute,
		Usage:       "Interval of time for pinging peers to keep the connection alive.",
		Destination: &options.KeepAlive,
		EnvVars:     []string{"WAKUNODE2_KEEP_ALIVE"},
	})
	PersistPeers = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "persist-peers",
		Usage:       "Enable peer persistence",
		Destination: &options.PersistPeers,
		Value:       false,
		EnvVars:     []string{"WAKUNODE2_PERSIST_PEERS"},
	})
	NAT = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "nat", // This was added so js-waku test don't fail
		Usage:       "TODO: Not implemented yet. Specify method to use for determining public address: any, none ('any' will attempt upnp/pmp)",
		Value:       "any",
		Destination: &options.NAT, // TODO: accept none,any,upnp,extaddr
		EnvVars:     []string{"WAKUNODE2_NAT"},
	})
	IPAddress = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "ext-ip", // This was added so js-waku test don't fail
		Usage:       "Set external IP address",
		Value:       "",
		Destination: &options.ExtIP,
		EnvVars:     []string{"WAKUNODE2_EXT_IP"},
	})
	ExtMultiaddresses = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "ext-multiaddr",
		Usage: "External address to advertise to other nodes. Overrides --address and --ws-address flags. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.AdvertiseAddresses,
		},
		EnvVars: []string{"WAKUNODE2_EXT_MULTIADDR"},
	})
	ShowAddresses = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "show-addresses",
		Usage:       "Display listening addresses according to current configuration",
		Destination: &options.ShowAddresses,
	})
	CircuitRelay = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "circuit-relay",
		Usage:       "Enable circuit relay service",
		Value:       true,
		Destination: &options.CircuitRelay,
		EnvVars:     []string{"WAKUNODE2_CIRCUIT_RELAY"},
	})
	ForceReachability = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "force-reachability",
		Usage:       "Force the node reachability. WARNING: This flag is created for testing circuit relay and is not meant to be used in production. Use 'public' or 'private'",
		Value:       "",
		Hidden:      true,
		Destination: &options.ForceReachability,
		EnvVars:     []string{"WAKUNODE2_REACHABILITY"},
	})
	ResourceScalingMemoryPercent = altsrc.NewFloat64Flag(&cli.Float64Flag{
		Name:        "resource-scaling-memory-percentage",
		Usage:       "Determines the percentage of total accessible memory that wil be dedicated to go-waku. A dedicated node with a lot of RAM could allocate 25% or more memory to go-waku",
		Value:       25,
		Destination: &options.ResourceScalingMemoryPercent,
		EnvVars:     []string{"WAKUNODE2_RESOURCE_MEMORY_PERCENTAGE"},
	})
	ResourceScalingFDPercent = altsrc.NewFloat64Flag(&cli.Float64Flag{
		Name:        "resource-scaling-fd-percentage",
		Usage:       "Determines the percentage of total file descriptors that wil be dedicated to go-waku.",
		Value:       50,
		Destination: &options.ResourceScalingFDPercent,
		EnvVars:     []string{"WAKUNODE2_RESOURCE_FD_PERCENTAGE"},
	})
	LogLevel = cliutils.NewGenericFlagSingleValue(&cli.GenericFlag{
		Name:    "log-level",
		Aliases: []string{"l"},
		Value: &cliutils.ChoiceValue{
			Choices: []string{"DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"},
			Value:   &options.LogLevel,
		},
		Usage:   "Define the logging level (allowed values: DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL)",
		EnvVars: []string{"WAKUNODE2_LOG_LEVEL"},
	})
	LogEncoding = cliutils.NewGenericFlagSingleValue(&cli.GenericFlag{
		Name:  "log-encoding",
		Usage: "Define the encoding used for the logs (allowed values: console, nocolor, json)",
		Value: &cliutils.ChoiceValue{
			Choices: []string{"console", "nocolor", "json"},
			Value:   &options.LogEncoding,
		},
		EnvVars: []string{"WAKUNODE2_LOG_ENCODING"},
	})
	LogOutput = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "log-output",
		Value:       "stdout",
		Usage:       "specifies where logging output should be written  (stdout, file, file:./filename.log)",
		Destination: &options.LogOutput,
		EnvVars:     []string{"WAKUNODE2_LOG_OUTPUT"},
	})
	AgentString = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "agent-string",
		Value:       node.UserAgent,
		Usage:       "client id to advertise",
		Destination: &options.UserAgent,
		EnvVars:     []string{"WAKUNODE2_AGENT_STRING"},
	})
	IPColocationLimit = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "ip-colocation-limit",
		Value:       node.DefaultMaxConnectionsPerIP,
		Usage:       "max number of allowed peers from the same IP. Set it to 0 to remove the limitation.",
		Destination: &options.IPColocationLimit,
		EnvVars:     []string{"WAKUNODE2_IP_COLOCATION_LIMIT"},
	})
	Relay = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "relay",
		Value:       true,
		Usage:       "Enable relay protocol",
		Destination: &options.Relay.Enable,
		EnvVars:     []string{"WAKUNODE2_RELAY"},
	})
	Topics = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:        "topic",
		Usage:       "Default topic to subscribe to. Argument may be repeated. Deprecated! Please use pubsub-topic and/or content-topic instead.",
		Destination: &options.Relay.Topics,
		EnvVars:     []string{"WAKUNODE2_TOPICS"},
	})
	PubSubTopics = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:        "pubsub-topic",
		Usage:       "Default pubsub topic to subscribe to. Argument may be repeated.",
		Destination: &options.Relay.PubSubTopics,
		EnvVars:     []string{"WAKUNODE2_PUBSUB_TOPICS"},
	})
	ContentTopics = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:        "content-topic",
		Usage:       "Default content topic to subscribe to. Argument may be repeated.",
		Destination: &options.Relay.ContentTopics,
		EnvVars:     []string{"WAKUNODE2_CONTENT_TOPICS"},
	})
	ProtectedTopics = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:    "protected-topic",
		Usage:   "Topics and its public key to be used for message validation, topic:pubkey. Argument may be repeated.",
		EnvVars: []string{"WAKUNODE2_PROTECTED_TOPIC"},
		Value: &cliutils.ProtectedTopicSlice{
			Values: &options.Relay.ProtectedTopics,
		},
	})
	RelayPeerExchange = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "relay-peer-exchange",
		Value:       false,
		Usage:       "Enable GossipSub Peer Exchange",
		Destination: &options.Relay.PeerExchange,
		EnvVars:     []string{"WAKUNODE2_RELAY_PEER_EXCHANGE"},
	})
	MinRelayPeersToPublish = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "min-relay-peers-to-publish",
		Value:       1,
		Usage:       "Minimum number of peers to publish to Relay",
		Destination: &options.Relay.MinRelayPeersToPublish,
		EnvVars:     []string{"WAKUNODE2_MIN_RELAY_PEERS_TO_PUBLISH"},
	})
	MaxRelayMsgSize = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "max-msg-size",
		Value:       "150KB",
		Usage:       "Maximum message size. Supported formats are B, KiB, KB, MiB. If no suffix, default is bytes",
		Destination: &options.Relay.MaxMsgSize,
		EnvVars:     []string{"WAKUNODE2_MAX_RELAY_MSG_SIZE"},
	})
	StoreNodeFlag = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "storenode",
		Usage: "Multiaddr of a peer that supports store protocol. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.Store.Nodes,
		},
		EnvVars: []string{"WAKUNODE2_STORENODE"},
	})
	StoreFlag = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "store",
		Usage:       "Enable store protocol to persist messages",
		Destination: &options.Store.Enable,
		EnvVars:     []string{"WAKUNODE2_STORE"},
	})
	StoreMessageRetentionTime = altsrc.NewDurationFlag(&cli.DurationFlag{
		Name:        "store-message-retention-time",
		Value:       time.Hour * 24 * 2,
		Usage:       "maximum number of seconds before a message is removed from the store. Set to 0 to disable it",
		Destination: &options.Store.RetentionTime,
		EnvVars:     []string{"WAKUNODE2_STORE_MESSAGE_RETENTION_TIME"},
	})
	StoreMessageRetentionCapacity = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "store-message-retention-capacity",
		Value:       0,
		Usage:       "maximum number of messages to store. Set to 0 to disable it",
		Destination: &options.Store.RetentionMaxMessages,
		EnvVars:     []string{"WAKUNODE2_STORE_MESSAGE_RETENTION_CAPACITY"},
	})
	StoreMessageDBURL = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "store-message-db-url",
		Usage:       "The database connection URL for persistent storage.",
		Value:       "sqlite3://store.db",
		Destination: &options.Store.DatabaseURL,
		EnvVars:     []string{"WAKUNODE2_STORE_MESSAGE_DB_URL"},
	})
	StoreMessageDBMigration = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "store-message-db-migration",
		Usage:       "Enable database migration at start.",
		Destination: &options.Store.Migration,
		Value:       true,
		EnvVars:     []string{"WAKUNODE2_STORE_MESSAGE_DB_MIGRATION"},
	})
	FilterFlag = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "filter",
		Usage:       "Enable filter protocol",
		Destination: &options.Filter.Enable,
		EnvVars:     []string{"WAKUNODE2_FILTER"},
	})
	FilterNode = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "filternode",
		Usage: "Multiaddr of a peer that supports filter protocol. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.Filter.Nodes,
		},
		EnvVars: []string{"WAKUNODE2_FILTERNODE"},
	})
	FilterTimeout = altsrc.NewDurationFlag(&cli.DurationFlag{
		Name:        "filter-timeout",
		Value:       14400 * time.Second,
		Usage:       "Timeout for filter node in seconds",
		Destination: &options.Filter.Timeout,
		EnvVars:     []string{"WAKUNODE2_FILTER_TIMEOUT"},
	})
	LightPush = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "lightpush",
		Usage:       "Enable lightpush protocol",
		Destination: &options.LightPush.Enable,
		EnvVars:     []string{"WAKUNODE2_LIGHTPUSH"},
	})
	LightPushNode = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "lightpushnode",
		Usage: "Multiaddr of a peer that supports lightpush protocol. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.LightPush.Nodes,
		},
		EnvVars: []string{"WAKUNODE2_LIGHTPUSHNODE"},
	})
	Discv5Discovery = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "discv5-discovery",
		Usage:       "Enable discovering nodes via Node Discovery v5",
		Destination: &options.DiscV5.Enable,
		EnvVars:     []string{"WAKUNODE2_DISCV5_DISCOVERY"},
	})
	Discv5BootstrapNode = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:        "discv5-bootstrap-node",
		Usage:       "Text-encoded ENR for bootstrap node. Used when connecting to the network. Option may be repeated",
		Destination: &options.DiscV5.Nodes,
		EnvVars:     []string{"WAKUNODE2_DISCV5_BOOTSTRAP_NODE"},
	})
	Discv5UDPPort = altsrc.NewUintFlag(&cli.UintFlag{
		Name:        "discv5-udp-port",
		Value:       9000,
		Usage:       "Listening UDP port for Node Discovery v5.",
		Destination: &options.DiscV5.Port,
		EnvVars:     []string{"WAKUNODE2_DISCV5_UDP_PORT"},
	})
	Discv5ENRAutoUpdate = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "discv5-enr-auto-update",
		Usage:       "Discovery can automatically update its ENR with the IP address as seen by other nodes it communicates with.",
		Destination: &options.DiscV5.AutoUpdate,
		EnvVars:     []string{"WAKUNODE2_DISCV5_ENR_AUTO_UPDATE"},
	})
	Rendezvous = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "rendezvous",
		Usage:       "Enable rendezvous protocol for peer discovery",
		Destination: &options.Rendezvous.Enable,
		EnvVars:     []string{"WAKUNODE2_RENDEZVOUS"},
	})
	RendezvousNode = cliutils.NewGenericFlagMultiValue(&cli.GenericFlag{
		Name:  "rendezvous-node",
		Usage: "Multiaddr of a waku2 rendezvous node. Option may be repeated",
		Value: &cliutils.MultiaddrSlice{
			Values: &options.Rendezvous.Nodes,
		},
		EnvVars: []string{"WAKUNODE2_RENDEZVOUSNODE"},
	})
	RendezvousServer = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "rendezvous-server",
		Usage:       "Enable rendezvous protocol so other peers can use this node for discovery",
		Destination: &options.Rendezvous.Enable,
		EnvVars:     []string{"WAKUNODE2_RENDEZVOUS_SERVER"},
	})
	PeerExchange = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "peer-exchange",
		Usage:       "Enable waku peer exchange protocol (responder side)",
		Destination: &options.PeerExchange.Enable,
		EnvVars:     []string{"WAKUNODE2_PEER_EXCHANGE"},
	})
	PeerExchangeNode = cliutils.NewGenericFlagSingleValue(&cli.GenericFlag{
		Name:  "peer-exchange-node",
		Usage: "Peer multiaddr to send peer exchange requests to. (enables peer exchange protocol requester side)",
		Value: &cliutils.MultiaddrValue{
			Value: &options.PeerExchange.Node,
		},
		EnvVars: []string{"WAKUNODE2_PEER_EXCHANGE_NODE"},
	})
	DNSDiscovery = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "dns-discovery",
		Usage:       "Enable DNS discovery",
		Destination: &options.DNSDiscovery.Enable,
		EnvVars:     []string{"WAKUNODE2_DNS_DISCOVERY"},
	})
	DNSDiscoveryUrl = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:        "dns-discovery-url",
		Usage:       "URL for DNS node list in format 'enrtree://<key>@<fqdn>'. Option may be repeated",
		Destination: &options.DNSDiscovery.URLs,
		EnvVars:     []string{"WAKUNODE2_DNS_DISCOVERY_URL"},
	})
	DNSDiscoveryNameServer = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "dns-discovery-name-server",
		Aliases:     []string{"dns-discovery-nameserver"},
		Usage:       "DNS nameserver IP to query (empty to use system's default)",
		Destination: &options.DNSDiscovery.Nameserver,
		EnvVars:     []string{"WAKUNODE2_DNS_DISCOVERY_NAME_SERVER"},
	})
	MetricsServer = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "metrics-server",
		Aliases:     []string{"metrics"},
		Usage:       "Enable the metrics server",
		Destination: &options.Metrics.Enable,
		EnvVars:     []string{"WAKUNODE2_METRICS_SERVER"},
	})
	MetricsServerAddress = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "metrics-server-address",
		Aliases:     []string{"metrics-address"},
		Value:       "127.0.0.1",
		Usage:       "Listening address of the metrics server",
		Destination: &options.Metrics.Address,
		EnvVars:     []string{"WAKUNODE2_METRICS_SERVER_ADDRESS"},
	})
	MetricsServerPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "metrics-server-port",
		Aliases:     []string{"metrics-port"},
		Value:       8008,
		Usage:       "Listening HTTP port of the metrics server",
		Destination: &options.Metrics.Port,
		EnvVars:     []string{"WAKUNODE2_METRICS_SERVER_PORT"},
	})
	RESTFlag = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "rest",
		Usage:       "Enable Waku REST HTTP server",
		Destination: &options.RESTServer.Enable,
		EnvVars:     []string{"WAKUNODE2_REST"},
	})
	RESTAddress = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "rest-address",
		Value:       "127.0.0.1",
		Usage:       "Listening address of the REST HTTP server",
		Destination: &options.RESTServer.Address,
		EnvVars:     []string{"WAKUNODE2_REST_ADDRESS"},
	})
	RESTPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "rest-port",
		Value:       8645,
		Usage:       "Listening port of the REST HTTP server",
		Destination: &options.RESTServer.Port,
		EnvVars:     []string{"WAKUNODE2_REST_PORT"},
	})
	RESTRelayCacheCapacity = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "rest-relay-cache-capacity",
		Value:       1000,
		Usage:       "Capacity of the Relay REST API message cache",
		Destination: &options.RESTServer.RelayCacheCapacity,
		EnvVars:     []string{"WAKUNODE2_REST_RELAY_CACHE_CAPACITY"},
	})
	RESTFilterCacheCapacity = altsrc.NewIntFlag(&cli.IntFlag{
		Name:        "rest-filter-cache-capacity",
		Value:       30,
		Usage:       "Capacity of the Filter REST API message cache",
		Destination: &options.RESTServer.FilterCacheCapacity,
		EnvVars:     []string{"WAKUNODE2_REST_FILTER_CACHE_CAPACITY"},
	})
	RESTAdmin = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "rest-admin",
		Value:       false,
		Usage:       "Enable access to REST HTTP Admin API",
		Destination: &options.RESTServer.Admin,
		EnvVars:     []string{"WAKUNODE2_REST_ADMIN"},
	})
	PProf = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "pprof",
		Usage:       "provides runtime profiling data at /debug/pprof in both REST and RPC servers if they're enabled",
		Destination: &options.PProf,
		EnvVars:     []string{"WAKUNODE2_PPROF"},
	})
)
