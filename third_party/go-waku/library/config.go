package library

import (
	"encoding/json"
	"errors"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// WakuConfig contains all the configuration settings exposed to users of mobile and c libraries
type WakuConfig struct {
	Host                   *string          `json:"host,omitempty"`
	Port                   *int             `json:"port,omitempty"`
	AdvertiseAddress       *string          `json:"advertiseAddr,omitempty"`
	NodeKey                *string          `json:"nodeKey,omitempty"`
	LogLevel               *string          `json:"logLevel,omitempty"`
	KeepAliveInterval      *int             `json:"keepAliveInterval,omitempty"`
	EnableRelay            *bool            `json:"relay"`
	RelayTopics            []string         `json:"relayTopics,omitempty"`
	GossipSubParams        *GossipSubParams `json:"gossipsubParams,omitempty"`
	MinPeersToPublish      *int             `json:"minPeersToPublish,omitempty"`
	DNSDiscoveryURLs       []string         `json:"dnsDiscoveryURLs,omitempty"`
	DNSDiscoveryNameServer string           `json:"dnsDiscoveryNameServer,omitempty"`
	EnableDiscV5           *bool            `json:"discV5,omitempty"`
	DiscV5BootstrapNodes   []string         `json:"discV5BootstrapNodes,omitempty"`
	DiscV5UDPPort          *uint            `json:"discV5UDPPort,omitempty"`
	EnableStore            *bool            `json:"store,omitempty"`
	DatabaseURL            *string          `json:"databaseURL,omitempty"`
	RetentionMaxMessages   *int             `json:"storeRetentionMaxMessages,omitempty"`
	RetentionTimeSeconds   *int             `json:"storeRetentionTimeSeconds,omitempty"`
	DNS4DomainName         string           `json:"dns4DomainName,omitempty"`
	Websockets             *WebsocketConfig `json:"websockets,omitempty"`
	ClusterID              int              `json:"clusterID"`
	Shards                 []uint16         `json:"shards"`
}

// WebsocketConfig contains all the settings required to setup websocket support in waku
type WebsocketConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	CertPath string `json:"certPath"`
	KeyPath  string `json:"keyPath"`
}

var defaultHost = "0.0.0.0"
var defaultPort = 60000
var defaultKeepAliveInterval = 20
var defaultEnableRelay = true
var defaultMinPeersToPublish = 0
var defaultEnableDiscV5 = false
var defaultDiscV5UDPPort = uint(9000)
var defaultLogLevel = "INFO"
var defaultEnableStore = false
var defaultDatabaseURL = "sqlite3://store.db"
var defaultRetentionMaxMessages = 10000
var defaultRetentionTimeSeconds = 30 * 24 * 60 * 60 // 30d

var defaultWSPort = 60001
var defaultWSSPort = 6443
var defaultWSHost = "0.0.0.0"

// GossipSubParams defines all the gossipsub specific parameters.
type GossipSubParams struct {
	// overlay parameters.

	// D sets the optimal degree for a GossipSub topic mesh. For example, if D == 6,
	// each peer will want to have about six peers in their mesh for each topic they're subscribed to.
	// D should be set somewhere between Dlo and Dhi.
	D *int `json:"d,omitempty"`

	// Dlo sets the lower bound on the number of peers we keep in a GossipSub topic mesh.
	// If we have fewer than Dlo peers, we will attempt to graft some more into the mesh at
	// the next heartbeat.
	Dlo *int `json:"d_low,omitempty"`

	// Dhi sets the upper bound on the number of peers we keep in a GossipSub topic mesh.
	// If we have more than Dhi peers, we will select some to prune from the mesh at the next heartbeat.
	Dhi *int `json:"d_high,omitempty"`

	// Dscore affects how peers are selected when pruning a mesh due to over subscription.
	// At least Dscore of the retained peers will be high-scoring, while the remainder are
	// chosen randomly.
	Dscore *int `json:"d_score,omitempty"`

	// Dout sets the quota for the number of outbound connections to maintain in a topic mesh.
	// When the mesh is pruned due to over subscription, we make sure that we have outbound connections
	// to at least Dout of the survivor peers. This prevents sybil attackers from overwhelming
	// our mesh with incoming connections.
	//
	// Dout must be set below Dlo, and must not exceed D / 2.
	Dout *int `json:"dOut,omitempty"`

	// gossip parameters

	// HistoryLength controls the size of the message cache used for gossip.
	// The message cache will remember messages for HistoryLength heartbeats.
	HistoryLength *int `json:"historyLength,omitempty"`

	// HistoryGossip controls how many cached message ids we will advertise in
	// IHAVE gossip messages. When asked for our seen message IDs, we will return
	// only those from the most recent HistoryGossip heartbeats. The slack between
	// HistoryGossip and HistoryLength allows us to avoid advertising messages
	// that will be expired by the time they're requested.
	//
	// HistoryGossip must be less than or equal to HistoryLength to
	// avoid a runtime panic.
	HistoryGossip *int `json:"historyGossip,omitempty"`

	// Dlazy affects how many peers we will emit gossip to at each heartbeat.
	// We will send gossip to at least Dlazy peers outside our mesh. The actual
	// number may be more, depending on GossipFactor and how many peers we're
	// connected to.
	Dlazy *int `json:"dLazy,omitempty"`

	// GossipFactor affects how many peers we will emit gossip to at each heartbeat.
	// We will send gossip to GossipFactor * (total number of non-mesh peers), or
	// Dlazy, whichever is greater.
	GossipFactor *float64 `json:"gossipFactor,omitempty"`

	// GossipRetransmission controls how many times we will allow a peer to request
	// the same message id through IWANT gossip before we start ignoring them. This is designed
	// to prevent peers from spamming us with requests and wasting our resources.
	GossipRetransmission *int `json:"gossipRetransmission,omitempty"`

	// heartbeat interval

	// HeartbeatInitialDelayMs is the short delay before the heartbeat timer begins
	// after the router is initialized.
	HeartbeatInitialDelayMs *int `json:"heartbeatInitialDelayMs,omitempty"`

	// HeartbeatIntervalSeconds controls the time between heartbeats.
	HeartbeatIntervalSeconds *int `json:"heartbeatIntervalSeconds,omitempty"`

	// SlowHeartbeatWarning is the duration threshold for heartbeat processing before emitting
	// a warning; this would be indicative of an overloaded peer.
	SlowHeartbeatWarning *float64 `json:"slowHeartbeatWarning,omitempty"`

	// FanoutTTLSeconds controls how long we keep track of the fanout state. If it's been
	// FanoutTTLSeconds since we've published to a topic that we're not subscribed to,
	// we'll delete the fanout map for that topic.
	FanoutTTLSeconds *int `json:"fanoutTTLSeconds,omitempty"`

	// PrunePeers controls the number of peers to include in prune Peer eXchange.
	// When we prune a peer that's eligible for PX (has a good score, etc), we will try to
	// send them signed peer records for up to PrunePeers other peers that we
	// know of.
	PrunePeers *int `json:"prunePeers,omitempty"`

	// PruneBackoffSeconds controls the backoff time for pruned peers. This is how long
	// a peer must wait before attempting to graft into our mesh again after being pruned.
	// When pruning a peer, we send them our value of PruneBackoff so they know
	// the minimum time to wait. Peers running older versions may not send a backoff time,
	// so if we receive a prune message without one, we will wait at least PruneBackoff
	// before attempting to re-graft.
	PruneBackoffSeconds *int `json:"pruneBackoffSeconds,omitempty"`

	// UnsubscribeBackoffSeconds controls the backoff time to use when unsuscribing
	// from a topic. A peer should not resubscribe to this topic before this
	// duration.
	UnsubscribeBackoffSeconds *int `json:"unsubscribeBackoffSeconds,omitempty"`

	// Connectors controls the number of active connection attempts for peers obtained through PX.
	Connectors *int `json:"connectors,omitempty"`

	// MaxPendingConnections sets the maximum number of pending connections for peers attempted through px.
	MaxPendingConnections *int `json:"maxPendingConnections,omitempty"`

	// ConnectionTimeoutSeconds controls the timeout for connection attempts.
	ConnectionTimeoutSeconds *int `json:"connectionTimeoutSeconds,omitempty"`

	// DirectConnectTicks is the number of heartbeat ticks for attempting to reconnect direct peers
	// that are not currently connected.
	DirectConnectTicks *uint64 `json:"directConnectTicks,omitempty"`

	// DirectConnectInitialDelaySeconds is the initial delay before opening connections to direct peers
	DirectConnectInitialDelaySeconds *int `json:"directConnectInitialDelaySeconds,omitempty"`

	// OpportunisticGraftTicks is the number of heartbeat ticks for attempting to improve the mesh
	// with opportunistic grafting. Every OpportunisticGraftTicks we will attempt to select some
	// high-scoring mesh peers to replace lower-scoring ones, if the median score of our mesh peers falls
	// below a threshold (see https://godoc.org/github.com/libp2p/go-libp2p-pubsub#PeerScoreThresholds).
	OpportunisticGraftTicks *uint64 `json:"opportunisticGraftTicks,omitempty"`

	// OpportunisticGraftPeers is the number of peers to opportunistically graft.
	OpportunisticGraftPeers *int `json:"opportunisticGraftPeers,omitempty"`

	// If a GRAFT comes before GraftFloodThresholdSeconds has elapsed since the last PRUNE,
	// then there is an extra score penalty applied to the peer through P7.
	GraftFloodThresholdSeconds *int `json:"graftFloodThresholdSeconds,omitempty"`

	// MaxIHaveLength is the maximum number of messages to include in an IHAVE message.
	// Also controls the maximum number of IHAVE ids we will accept and request with IWANT from a
	// peer within a heartbeat, to protect from IHAVE floods. You should adjust this value from the
	// default if your system is pushing more than 5000 messages in HistoryGossip heartbeats;
	// with the defaults this is 1666 messages/s.
	MaxIHaveLength *int `json:"maxIHaveLength,omitempty"`

	// MaxIHaveMessages is the maximum number of IHAVE messages to accept from a peer within a heartbeat.
	MaxIHaveMessages *int `json:"maxIHaveMessages,omitempty"`

	// Time to wait for a message requested through IWANT following an IHAVE advertisement.
	// If the message is not received within this window, a broken promise is declared and
	// the router may apply bahavioural penalties.
	IWantFollowupTimeSeconds *int `json:"iWantFollowupTimeSeconds,omitempty"`

	// configures when a previously seen message ID can be forgotten about
	SeenMessagesTTLSeconds *int `json:"seenMessagesTTLSeconds"`
}

func getConfig(configJSON string) (WakuConfig, error) {
	var config WakuConfig
	if configJSON != "" {
		err := json.Unmarshal([]byte(configJSON), &config)
		if err != nil {
			return WakuConfig{}, err
		}
	}

	if config.Host == nil {
		config.Host = &defaultHost
	}

	if config.EnableRelay == nil {
		config.EnableRelay = &defaultEnableRelay
	}

	if config.EnableDiscV5 == nil {
		config.EnableDiscV5 = &defaultEnableDiscV5
	}

	if config.Host == nil {
		config.Host = &defaultHost
	}

	if config.Port == nil {
		config.Port = &defaultPort
	}

	if config.DiscV5UDPPort == nil {
		config.DiscV5UDPPort = &defaultDiscV5UDPPort
	}

	if config.KeepAliveInterval == nil {
		config.KeepAliveInterval = &defaultKeepAliveInterval
	}

	if config.MinPeersToPublish == nil {
		config.MinPeersToPublish = &defaultMinPeersToPublish
	}

	if config.LogLevel == nil {
		config.LogLevel = &defaultLogLevel
	}

	if config.EnableStore == nil {
		config.EnableStore = &defaultEnableStore
	}

	if config.DatabaseURL == nil {
		config.DatabaseURL = &defaultDatabaseURL
	}

	if config.RetentionMaxMessages == nil {
		config.RetentionMaxMessages = &defaultRetentionMaxMessages
	}

	if config.RetentionTimeSeconds == nil {
		config.RetentionTimeSeconds = &defaultRetentionTimeSeconds
	}

	if config.Websockets == nil {
		config.Websockets = &WebsocketConfig{}
	}

	if config.Websockets.Host == "" {
		config.Websockets.Host = defaultWSHost
	}

	if config.Websockets.Port == nil {
		if config.Websockets.Secure {
			config.Websockets.Port = &defaultWSSPort
		} else {
			config.Websockets.Port = &defaultWSPort
		}
	}

	if config.Websockets.CertPath == "" && config.Websockets.Secure {
		return WakuConfig{}, errors.New("certPath is required")
	}

	if config.Websockets.KeyPath == "" && config.Websockets.Secure {
		return WakuConfig{}, errors.New("keyPath is required")
	}

	return config, nil
}

func getSeenTTL(cfg WakuConfig) time.Duration {
	if cfg.GossipSubParams == nil || *cfg.GossipSubParams.SeenMessagesTTLSeconds == 0 {
		return pubsub.TimeCacheDuration
	}

	return time.Duration(*cfg.GossipSubParams.SeenMessagesTTLSeconds)
}

func getGossipSubParams(cfg *GossipSubParams) pubsub.GossipSubParams {
	params := pubsub.DefaultGossipSubParams()

	if cfg.D != nil {
		params.D = *cfg.D
	}

	if cfg.Dlo != nil {
		params.Dlo = *cfg.Dlo
	}

	if cfg.Dhi != nil {
		params.Dhi = *cfg.Dhi
	}

	if cfg.Dscore != nil {
		params.Dscore = *cfg.Dscore
	}

	if cfg.Dout != nil {
		params.Dout = *cfg.Dout
	}

	if cfg.HistoryLength != nil {
		params.HistoryLength = *cfg.HistoryLength
	}

	if cfg.HistoryGossip != nil {
		params.HistoryGossip = *cfg.HistoryGossip
	}

	if cfg.Dlazy != nil {
		params.Dlazy = *cfg.Dlazy
	}

	if cfg.GossipFactor != nil {
		params.GossipFactor = *cfg.GossipFactor
	}

	if cfg.GossipRetransmission != nil {
		params.GossipRetransmission = *cfg.GossipRetransmission
	}

	if cfg.HeartbeatInitialDelayMs != nil {
		params.HeartbeatInitialDelay = time.Duration(*cfg.HeartbeatInitialDelayMs) * time.Millisecond
	}

	if cfg.HeartbeatIntervalSeconds != nil {
		params.HeartbeatInterval = time.Duration(*cfg.HeartbeatIntervalSeconds) * time.Second
	}

	if cfg.SlowHeartbeatWarning != nil {
		params.SlowHeartbeatWarning = *cfg.SlowHeartbeatWarning
	}

	if cfg.FanoutTTLSeconds != nil {
		params.FanoutTTL = time.Duration(*cfg.FanoutTTLSeconds) * time.Second
	}

	if cfg.PrunePeers != nil {
		params.PrunePeers = *cfg.PrunePeers
	}

	if cfg.PruneBackoffSeconds != nil {
		params.PruneBackoff = time.Duration(*cfg.PruneBackoffSeconds) * time.Second
	}

	if cfg.UnsubscribeBackoffSeconds != nil {
		params.UnsubscribeBackoff = time.Duration(*cfg.UnsubscribeBackoffSeconds) * time.Second
	}

	if cfg.Connectors != nil {
		params.Connectors = *cfg.Connectors
	}

	if cfg.MaxPendingConnections != nil {
		params.MaxPendingConnections = *cfg.MaxPendingConnections
	}

	if cfg.ConnectionTimeoutSeconds != nil {
		params.ConnectionTimeout = time.Duration(*cfg.ConnectionTimeoutSeconds) * time.Second
	}

	if cfg.DirectConnectTicks != nil {
		params.DirectConnectTicks = *cfg.DirectConnectTicks
	}

	if cfg.DirectConnectInitialDelaySeconds != nil {
		params.DirectConnectInitialDelay = time.Duration(*cfg.DirectConnectInitialDelaySeconds) * time.Second
	}

	if cfg.OpportunisticGraftTicks != nil {
		params.OpportunisticGraftTicks = *cfg.OpportunisticGraftTicks
	}

	if cfg.OpportunisticGraftPeers != nil {
		params.OpportunisticGraftPeers = *cfg.OpportunisticGraftPeers
	}

	if cfg.GraftFloodThresholdSeconds != nil {
		params.GraftFloodThreshold = time.Duration(*cfg.GraftFloodThresholdSeconds) * time.Second
	}

	if cfg.MaxIHaveLength != nil {
		params.MaxIHaveLength = *cfg.MaxIHaveLength
	}

	if cfg.MaxIHaveMessages != nil {
		params.MaxIHaveMessages = *cfg.MaxIHaveMessages
	}

	if cfg.IWantFollowupTimeSeconds != nil {
		params.IWantFollowupTime = time.Duration(*cfg.IWantFollowupTimeSeconds) * time.Second
	}

	return params
}
