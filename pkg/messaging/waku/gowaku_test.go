package waku

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"

	"go.uber.org/mock/gomock"

	"github.com/waku-org/go-waku/waku/v2/node"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/pausable"
)

func TestBootstrapRecoveryBackoff(t *testing.T) {
	var recovery bootstrapRecovery
	now := time.Now()
	require.False(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	delays := []time.Duration{
		15 * time.Second, 30 * time.Second, time.Minute,
		2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute,
	}
	for _, delay := range delays {
		now = now.Add(delay)
		require.False(t, recovery.due(now.Add(-time.Nanosecond), 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
		require.True(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
		require.False(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	}
	require.False(t, recovery.due(now, 1, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	require.False(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	require.True(t, recovery.due(now.Add(15*time.Second), 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
}

func TestBootstrapRecoveryBackoffCapsClockRollback(t *testing.T) {
	var recovery bootstrapRecovery
	now := time.Now().Round(0)
	require.False(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	require.True(t, recovery.due(now.Add(bootstrapRecoveryInitialBackoff), 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))

	now = now.Add(-time.Hour)
	require.False(t, recovery.due(now, 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
	require.True(t, recovery.due(now.Add(30*time.Second), 0, bootstrapRecoveryInitialBackoff, bootstrapRecoveryMaxBackoff))
}

func TestLightClientRecoversWithoutNetworkChange(t *testing.T) {
	for _, paused := range []bool{false, true} {
		t.Run(fmt.Sprintf("paused=%t", paused), func(t *testing.T) {
			bootstrap, err := node.New(
				node.WithHostAddress(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")}),
				node.WithClusterID(16),
				node.WithShards([]uint16{32}),
				node.WithPeerExchange(),
				node.WithLightPush(),
			)
			require.NoError(t, err)
			require.NoError(t, bootstrap.Start(context.Background()))
			t.Cleanup(bootstrap.Stop)

			tree, address := makeTestTree("recovery", []*enode.Node{bootstrap.ENR()}, nil)
			w, err := New(nil, &Config{
				Host:      "127.0.0.1",
				ClusterID: 16,
				Mode:      ModeEdge,
				WakuNodes: []string{address},
				Resolver:  mapResolver(tree.ToTXT("recovery")),
			}, nil, nil)
			require.NoError(t, err)
			w.monitoringConfig = connectionMonitoringConfig{
				interval:                  time.Second,
				pausedLightClientInterval: 2 * time.Second,
				recoveryInitialBackoff:    100 * time.Millisecond,
				recoveryMaxBackoff:        500 * time.Millisecond,
			}
			require.NoError(t, w.Start())
			t.Cleanup(func() { require.NoError(t, w.Stop()) })
			require.Eventually(t, func() bool {
				return w.lightClientServicePeerCount() > 0
			}, 10*time.Second, 50*time.Millisecond)
			if paused {
				require.NoError(t, w.Pause())
			}

			w.node.DisconnectAllPeers()
			require.Eventually(t, func() bool {
				return len(w.Peers()) == 0
			}, 5*time.Second, 50*time.Millisecond)
			// Preserve the peerstore metadata to exercise unchanged-ENR recovery.
			peerStore := w.node.Host().Peerstore().(wps.WakuPeerstore)
			require.NoError(t, peerStore.SetOrigin(bootstrap.Host().ID(), wps.DNSDiscovery))
			require.NoError(t, peerStore.SetENR(bootstrap.Host().ID(), bootstrap.ENR()))
			w.node.Host().Peerstore().AddAddrs(bootstrap.Host().ID(), bootstrap.ListenAddresses(), time.Hour)
			_, err = peerStore.ENR(bootstrap.Host().ID())
			require.NoError(t, err, "bootstrap ENR metadata should be present")
			require.NotEmpty(t, w.node.Host().Peerstore().Addrs(bootstrap.Host().ID()))

			require.Eventually(t, func() bool {
				return w.lightClientServicePeerCount() > 0
			}, 5*time.Second, 50*time.Millisecond)

			var concurrentDiscoveries sync.WaitGroup
			for i := 0; i < 10; i++ {
				concurrentDiscoveries.Add(1)
				go func() {
					defer concurrentDiscoveries.Done()
					w.discoverAndConnect(address)
				}()
			}
			require.NoError(t, w.Stop())
			concurrentDiscoveries.Wait()
		})
	}
}

type failingDNSResolver struct {
	called chan int
	calls  atomic.Int32
}

func (r *failingDNSResolver) LookupTXT(context.Context, string) ([]string, error) {
	call := int(r.calls.Add(1))
	select {
	case r.called <- call:
	default:
	}
	return nil, errors.New("DNS unavailable")
}

func TestGoingOnlineKicksDNSRetryLoop(t *testing.T) {
	resolver := &failingDNSResolver{called: make(chan int, 2)}
	w, err := New(nil, &Config{Mode: ModeEdge, Resolver: resolver}, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		w.cancel()
		w.wg.Wait()
		w.envelopeCache.Stop()
	})

	const address = "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@nodes.example.org"
	w.discoverAndConnect(address)
	select {
	case <-resolver.called:
	case <-time.After(5 * time.Second):
		t.Fatal("initial DNS discovery did not start")
	}

	w.discoverAndConnect(address)
	select {
	case call := <-resolver.called:
		require.Equal(t, 2, call)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("going-online event did not wake DNS discovery from backoff")
	}
}

type blockingDNSResolver struct {
	called chan struct{}
	calls  atomic.Int32
}

func (r *blockingDNSResolver) LookupTXT(ctx context.Context, _ string) ([]string, error) {
	r.calls.Add(1)
	select {
	case r.called <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestBootstrapRecoverySharesDNSRetryLoop(t *testing.T) {
	resolver := &blockingDNSResolver{called: make(chan struct{}, 1)}
	w, err := New(nil, &Config{Mode: ModeEdge, Resolver: resolver}, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		w.cancel()
		w.wg.Wait()
		w.envelopeCache.Stop()
	})
	for i := 0; i < 10; i++ {
		w.discoverAndConnect("enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@nodes.example.org")
	}
	select {
	case <-resolver.called:
	case <-time.After(5 * time.Second):
		t.Fatal("DNS discovery did not start")
	}
	w.cancel()
	w.wg.Wait()
	require.EqualValues(t, 1, resolver.calls.Load())
}

func TestNewRetrievesInitialDiscV5BootstrapNodes(t *testing.T) {
	resolver := &failingDNSResolver{called: make(chan int, 1)}
	cfg := &Config{
		DiscV5BootstrapNodes: []string{
			"enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@nodes.example.org",
		},
		Resolver: resolver,
	}

	waku, err := New(nil, cfg, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		waku.cancel()
		waku.envelopeCache.Stop()
		waku.wg.Wait()
	})

	select {
	case <-resolver.called:
	default:
		t.Fatal("expected initial DiscV5 bootstrap nodes to be retrieved through DNS")
	}
}

func TestWakuLifecycleState(t *testing.T) {
	// New() alone is safe to call with nil params; Start()/Stop() are not used
	// here because they initialise the SDS library (requires a full node
	// environment). The PauseBroadcaster state machine is validated directly.
	w, err := New(nil, nil, nil, nil)
	require.NoError(t, err)

	require.Equal(t, pausable.ServiceStateStopped, w.PausableState())

	w.MarkStarted()
	require.Equal(t, pausable.ServiceStateRunning, w.PausableState())

	w.MarkStopped()
	require.Equal(t, pausable.ServiceStateStopped, w.PausableState())
}

// Regression test for #7620: storeClient is only populated by Start, so a
// query racing/preceding it must return an error instead of a nil-pointer panic.
func TestStoreQueryBeforeStartReturnsError(t *testing.T) {
	w, err := New(nil, nil, nil, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		w.cancel()
		w.envelopeCache.Stop()
		w.wg.Wait()
	})

	err = w.StoreQuery(context.Background(), batch(), 10, nil, false)
	require.ErrorIs(t, err, ErrNoStorenodesReachable)
}

func TestHandlePeerAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mocks
	mockHandler := NewMockpeerAddressHandler(ctrl)

	t.Run("valid enrtree", func(t *testing.T) {
		addr := "enrtree:// " + gofakeit.LetterN(10)

		// Setup discover mock expectation.
		mockHandler.EXPECT().
			discoverAndConnect(addr).
			Times(1)

		// Setup connect expectation (not called for enrtree resolution).
		mockHandler.EXPECT().
			connect(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		// Call the tested function
		err := handlePeerAddress(addr, mockHandler)
		require.NoError(t, err)
	})

	t.Run("invalid multiaddr", func(t *testing.T) {
		// Use a multiaddr with no p2p peerID
		addr := fmt.Sprintf("/ip4/%s/tcp/%d/", gofakeit.IPv4Address(), gofakeit.Uint16())

		// Setup discover mock expectation (no call expected).
		mockHandler.EXPECT().
			discoverAndConnect(gomock.Any()).
			Times(0)

		// Setup connect mock expectation (no call expected).
		mockHandler.EXPECT().
			connect(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		// Call the tested function
		err := handlePeerAddress(addr, mockHandler)
		require.ErrorContains(t, err, "invalid peer multiaddress")
	})

	t.Run("valid multiaddr", func(t *testing.T) {
		// Generate peer ID
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		publicKey := utils.EcdsaPubKeyToSecp256k1PublicKey(&privateKey.PublicKey)
		peerID, err := peer.IDFromPublicKey(publicKey)
		require.NoError(t, err)

		// Generate multiaddr
		ip4Addr := gofakeit.IPv4Address()
		port := gofakeit.Number(1024, 65535)
		addr := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip4Addr, port, peerID.String())
		maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip4Addr, port))
		require.NoError(t, err)

		// Setup expectations
		expectedPeerInfo := peer.AddrInfo{
			ID:    peerID,
			Addrs: []multiaddr.Multiaddr{maddr},
		}
		mockHandler.EXPECT().
			discoverAndConnect(gomock.Any()).
			Times(0)
		mockHandler.EXPECT().
			connect(gomock.Eq(expectedPeerInfo), gomock.Nil(), gomock.Eq(wps.Static)).
			//connect(gomock.Eq(expectedPeerInfo), gomock.Nil(), gomock.Eq(wps.Static)).
			Times(1)

		// Call the tested function
		err = handlePeerAddress(addr, mockHandler)
		require.NoError(t, err)
	})

	t.Run("valid enr", func(t *testing.T) {
		const enr = "enr:-QEQuEBuiQgFlJNcv255042zwyl4pOBOivakX8N30Dr9vaaEU2q8-7N4GVY4Hk87iEKELjlIXTpE9Wj6EQq1lrBuc7ayAYJpZIJ2NIJpcISPxvrpim11bHRpYWRkcnO4YAAtNihib290LTAxLmRvLWFtczMuc3RhdHVzLnN0YWdpbmcuc3RhdHVzLmltBnZfAC82KGJvb3QtMDEuZG8tYW1zMy5zdGF0dXMuc3RhZ2luZy5zdGF0dXMuaW0GAbveA4Jyc40AEAUAAQAgAEAAgAEAiXNlY3AyNTZrMaEDq-yGgpuoUG6NKkbIDRmrMiT-bEVzFlpWLEK_rF3yKUaDdGNwgnZfg3VkcIIjKIV3YWt1Mg0"
		expectedAddrs := []string{
			"/ip4/143.198.250.233/tcp/30303/p2p/16Uiu2HAmQE7FXQc6iZHdBzYfw3qCSDa9dLc1wsBJKoP4aZvztq2d",
			"/dns4/boot-01.do-ams3.status.staging.status.im/tcp/30303/p2p/16Uiu2HAmQE7FXQc6iZHdBzYfw3qCSDa9dLc1wsBJKoP4aZvztq2d",
			"/dns4/boot-01.do-ams3.status.staging.status.im/tcp/443/wss/p2p/16Uiu2HAmQE7FXQc6iZHdBzYfw3qCSDa9dLc1wsBJKoP4aZvztq2d",
		}
		expectedPeerID, err := peer.Decode("16Uiu2HAmQE7FXQc6iZHdBzYfw3qCSDa9dLc1wsBJKoP4aZvztq2d")
		require.NoError(t, err)

		node, err := enode.Parse(enode.ValidSchemes, enr)
		require.NoError(t, err)

		//maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
		//require.NoError(t, err)

		// Setup discover mock expectation.
		mockHandler.EXPECT().
			discoverAndConnect(gomock.Any()).
			Times(0)

		// Setup connect expectation (not called due to ENR resolution failure).
		expectedPeerInfo := peer.AddrInfo{
			ID:    expectedPeerID,
			Addrs: []multiaddr.Multiaddr{},
		}
		for _, m := range expectedAddrs {
			maddr, err := multiaddr.NewMultiaddr(m)
			require.NoError(t, err)
			expectedPeerInfo.Addrs = append(expectedPeerInfo.Addrs, maddr)
		}
		mockHandler.EXPECT().
			connect(gomock.Eq(expectedPeerInfo), gomock.Eq(node), gomock.Eq(wps.Static)).
			Times(1)

		// Call the tested function
		err = handlePeerAddress(enr, mockHandler)
		require.NoError(t, err)
	})

	t.Run("unknown address format", func(t *testing.T) {
		addr := gofakeit.LetterN(10)

		// Setup discover mock expectation (no call expected).
		mockHandler.EXPECT().
			discoverAndConnect(gomock.Any()).
			Times(0)

		// Setup connect mock expectation (also no call expected).
		mockHandler.EXPECT().
			connect(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		// Call the tested function
		err := handlePeerAddress(addr, mockHandler)
		require.ErrorContains(t, err, "unknown format of waku")
	})
}
