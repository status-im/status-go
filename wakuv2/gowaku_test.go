package wakuv2

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"

	"go.uber.org/mock/gomock"

	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/stretchr/testify/require"
)

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
