package node

import (
	"context"
	"net"
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestExternalAddressSelection(t *testing.T) {
	a1, _ := ma.NewMultiaddr("/ip4/192.168.0.106/tcp/60000/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                            // Valid
	a2, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                                // Valid but should not be prefered
	a3, _ := ma.NewMultiaddr("/ip4/192.168.1.20/tcp/19710/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                             // Valid
	a4, _ := ma.NewMultiaddr("/dns4/www.status.im/tcp/2012/ws/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                         // Invalid (it's useless)
	a5, _ := ma.NewMultiaddr("/dns4/www.status.im/tcp/443/wss/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                         // Valid
	a6, _ := ma.NewMultiaddr("/ip4/192.168.1.20/tcp/19710/wss/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                         // Invalid (local + wss)
	a7, _ := ma.NewMultiaddr("/ip4/192.168.1.20/tcp/19710/ws/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                          // Invalid  (it's useless)
	a8, _ := ma.NewMultiaddr("/dns4/store-01.gc-us-central1-a.status.prod.status.im/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")   // VALID
	a9, _ := ma.NewMultiaddr("/dns4/store-01.gc-us-central1-a.status.prod.status.im/tcp/443/wss/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg/p2p-circuit/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f") // VALID
	a10, _ := ma.NewMultiaddr("/dns4/node-01.gc-us-central1-a.waku.test.status.im/tcp/8000/wss/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG/p2p-circuit/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")  // VALID
	a11, _ := ma.NewMultiaddr("/dns4/node-01.gc-us-central1-a.waku.test.status.im/tcp/30303/p2p/16Uiu2HAmDCp8XJ9z1ev18zuv8NHekAsjNyezAvmMfFEJkiharitG/p2p-circuit/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")     // VALID
	a12, _ := ma.NewMultiaddr("/ip4/188.23.1.8/tcp/30303/p2p/16Uiu2HAmUVVrJo1KMw4QwUANYF7Ws4mfcRqf9xHaaGP87GbMuY2f")                                                                                                              // VALID

	addrs := []ma.Multiaddr{a1, a2, a3, a4, a5, a6, a7}

	w := &WakuNode{}
	extAddr, multiaddr, err := w.getENRAddresses(context.Background(), []ma.Multiaddr{a1, a2, a3, a4, a5, a6, a7})
	a4NoP2P, _ := decapsulateP2P(a4)
	require.NoError(t, err)
	require.Equal(t, extAddr.IP, net.IPv4(192, 168, 0, 106))
	require.Equal(t, extAddr.Port, 60000)
	require.Equal(t, multiaddr[0].String(), a4NoP2P.String())
	require.Len(t, multiaddr, 4)

	addrs = append(addrs, a8, a9, a10, a11, a12)
	extAddr, _, err = w.getENRAddresses(context.Background(), addrs)
	require.NoError(t, err)
	require.Equal(t, extAddr.IP, net.IPv4(188, 23, 1, 8))
	require.Equal(t, extAddr.Port, 30303)

	a8RelayNode, _ := decapsulateCircuitRelayAddr(context.Background(), a8)
	_, multiaddr, err = w.getENRAddresses(context.Background(), []ma.Multiaddr{a1, a8})
	require.NoError(t, err)
	require.Len(t, multiaddr, 1)
	require.Equal(t, multiaddr[0].String(), a8RelayNode.String()) // Should have included circuit-relay addr
}
