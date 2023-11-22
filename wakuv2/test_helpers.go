package wakuv2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func PeerExchangeConfig() *Config {
	config := &Config{}
	config.PeerExchange = true
	return config
}

func StoreConfig() Config {
	panic("not implemented")
}

// CreateLocalWakuNetwork sets up a local Waku network for testing.
func CreateLocalNetwork(t *testing.T, configs []*Config) []*Waku {
	nodes := make([]*Waku, len(configs))

	// start nodes
	for i, config := range configs {
		node, err := New("", "", config, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, node.Start())

		nodes[i] = node
	}

	// connect nodes
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			addrs := nodes[j].ListenAddresses()
			require.Greater(t, len(addrs), 0)
			_, err := nodes[i].AddRelayPeer(addrs[0])
			require.NoError(t, err)
			err = nodes[i].DialPeer(addrs[0])
			require.NoError(t, err)
		}
	}

	return nodes
}

// DropLocalWakuNetwork clears the resources that local Waku network uses.
func DropLocalWakuNetwork(t *testing.T, nodes []*Waku) {
	for _, node := range nodes {
		require.NoError(t, node.Stop())
	}
}
