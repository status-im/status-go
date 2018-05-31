package node

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"
)

func TestInboundConnections(t *testing.T) {
	type testCase struct {
		name    string
		max     int
		percent int
	}
	for _, tc := range []testCase{
		{
			name:    "All",
			max:     4,
			percent: 100,
		},
		{
			name:    "Half",
			max:     4,
			percent: 50,
		},
		{
			name:    "FloorDivision",
			max:     4,
			percent: 70,
		},
		{
			name:    "None",
			max:     4,
			percent: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, _ := crypto.GenerateKey()
			main := p2p.Server{
				Config: p2p.Config{
					NoDiscovery:    true,
					PrivateKey:     key,
					InboundPercent: tc.percent,
					MaxPeers:       tc.max,
					ListenAddr:     "127.0.0.1:0",
				},
			}
			require.NoError(t, main.Start())
			defer main.Stop()

			rst := make(chan *p2p.PeerEvent, main.Config.MaxPeers)
			sub := main.SubscribeEvents(rst)
			defer sub.Unsubscribe()

			peers := make([]*p2p.Server, 0, main.Config.MaxPeers)
			for i := 0; i < main.Config.MaxPeers; i++ {
				key, _ := crypto.GenerateKey()
				peer := &p2p.Server{
					Config: p2p.Config{
						NoDiscovery: true,
						PrivateKey:  key,
						MaxPeers:    1,
						ListenAddr:  "127.0.0.1:0",
					},
				}
				require.NoError(t, peer.Start())
				peers = append(peers, peer)
				peer.AddPeer(main.Self())
			}
			defer func() {
				for i := range peers {
					peers[i].Stop()
				}
			}()

			expected := (main.Config.MaxPeers * main.Config.InboundPercent) / 100
			connections := 0
			for {
				select {
				case ev := <-rst:
					if ev.Type == p2p.PeerEventTypeAdd {
						connections++
					}
				case <-time.After(500 * time.Millisecond):
					require.Equal(t, expected, connections)
					return
				}
			}
		})
	}
}
