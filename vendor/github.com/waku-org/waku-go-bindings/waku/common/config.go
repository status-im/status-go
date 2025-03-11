package common

type WakuConfig struct {
	Host                        string           `json:"host,omitempty"`
	Nodekey                     string           `json:"nodekey,omitempty"`
	Relay                       bool             `json:"relay"`
	Store                       bool             `json:"store,omitempty"`
	LegacyStore                 bool             `json:"legacyStore"`
	Storenode                   string           `json:"storenode,omitempty"`
	StoreMessageRetentionPolicy string           `json:"storeMessageRetentionPolicy,omitempty"`
	StoreMessageDbUrl           string           `json:"storeMessageDbUrl,omitempty"`
	StoreMessageDbVacuum        bool             `json:"storeMessageDbVacuum,omitempty"`
	StoreMaxNumDbConnections    int              `json:"storeMaxNumDbConnections,omitempty"`
	StoreResume                 bool             `json:"storeResume,omitempty"`
	Filter                      bool             `json:"filter,omitempty"`
	Filternode                  string           `json:"filternode,omitempty"`
	FilterSubscriptionTimeout   int64            `json:"filterSubscriptionTimeout,omitempty"`
	FilterMaxPeersToServe       uint32           `json:"filterMaxPeersToServe,omitempty"`
	FilterMaxCriteria           uint32           `json:"filterMaxCriteria,omitempty"`
	Lightpush                   bool             `json:"lightpush,omitempty"`
	LightpushNode               string           `json:"lightpushnode,omitempty"`
	LogLevel                    string           `json:"logLevel,omitempty"`
	DnsDiscovery                bool             `json:"dnsDiscovery,omitempty"`
	DnsDiscoveryUrl             string           `json:"dnsDiscoveryUrl,omitempty"`
	MaxMessageSize              string           `json:"maxMessageSize,omitempty"`
	Staticnodes                 []string         `json:"staticnodes,omitempty"`
	Discv5BootstrapNodes        []string         `json:"discv5BootstrapNodes,omitempty"`
	Discv5Discovery             bool             `json:"discv5Discovery,omitempty"`
	Discv5UdpPort               int              `json:"discv5UdpPort,omitempty"`
	ClusterID                   uint16           `json:"clusterId,omitempty"`
	Shards                      []uint16         `json:"shards,omitempty"`
	PeerExchange                bool             `json:"peerExchange,omitempty"`
	PeerExchangeNode            string           `json:"peerExchangeNode,omitempty"`
	TcpPort                     int              `json:"tcpPort,omitempty"`
	RateLimits                  RateLimitsConfig `json:"rateLimits,omitempty"`
}
