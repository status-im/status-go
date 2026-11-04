package noise

import (
	"bytes"
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func createRelayNode(t *testing.T) (host.Host, *relay.WakuRelay) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	b := relay.NewBroadcaster(1024)
	require.NoError(t, b.Start(context.Background()))
	relay := relay.NewWakuRelay(b, 0, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())
	relay.SetHost(host)
	err = relay.Start(context.Background())
	require.NoError(t, err)

	return host, relay
}

func TestPairingObj1Success(t *testing.T) {
	host1, relay1 := createRelayNode(t)
	host2, relay2 := createRelayNode(t)

	defer host1.Close()
	defer host2.Close()
	defer relay1.Stop()
	defer relay2.Stop()

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err := host1.Peerstore().AddProtocols(host2.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)
	_, err = host1.Network().DialPeer(context.Background(), host2.ID())
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Wait for relay to form mesh

	bobStaticKey, _ := n.DH25519.GenerateKeypair()
	bobEphemeralKey, _ := n.DH25519.GenerateKeypair()

	bobMessenger, err := NewWakuRelayMessenger(context.Background(), relay1, nil, timesource.NewDefaultClock())
	require.NoError(t, err)

	bobPairingObj, err := NewPairing(bobStaticKey, bobEphemeralKey, WithDefaultResponderParameters(), bobMessenger, utils.Logger())
	require.NoError(t, err)

	authCodeCheckCh := make(chan string, 2)

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Check that authcodes match
		authcode1 := <-authCodeCheckCh
		authcode2 := <-authCodeCheckCh
		require.Equal(t, authcode1, authcode2)
	}()

	// Execute in separate go routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := bobPairingObj.Execute(ctx)
		require.NoError(t, err)
	}()

	// Confirmation is done by manually
	go func() {
		authCode := <-bobPairingObj.AuthCode()
		authCodeCheckCh <- authCode
		err := bobPairingObj.ConfirmAuthCode(true)
		require.NoError(t, err)
	}()

	aliceStaticKey, _ := n.DH25519.GenerateKeypair()
	aliceEphemeralKey, _ := n.DH25519.GenerateKeypair()

	aliceMessenger, err := NewWakuRelayMessenger(context.Background(), relay2, nil, timesource.NewDefaultClock())
	require.NoError(t, err)

	qrString, qrMessageNameTag := bobPairingObj.PairingInfo()
	alicePairingObj, err := NewPairing(aliceStaticKey, aliceEphemeralKey, WithInitiatorParameters(qrString, qrMessageNameTag), aliceMessenger, utils.Logger())
	require.NoError(t, err)

	// Execute in separate go routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := alicePairingObj.Execute(ctx)
		require.NoError(t, err)
	}()

	// Alice waits for authcode and confirms it
	wg.Add(1)
	go func() {
		defer wg.Done()
		authCode := <-alicePairingObj.AuthCode()
		authCodeCheckCh <- authCode
		err := alicePairingObj.ConfirmAuthCode(true)
		require.NoError(t, err)
	}()

	wg.Wait()

	// We test read/write of random messages exchanged between Alice and Bob
	// Note that we exchange more than the number of messages contained in the nametag buffer to test if they are filled correctly as the communication proceeds
	// We assume messages are sent via one of waku protocols
	for i := 0; i < 10*n.MessageNametagBufferSize; i++ {
		// Alice writes to Bob
		message := generateRandomBytes(t, 32)
		msg, err := alicePairingObj.Encrypt(message)
		require.NoError(t, err)

		readMessage, err := bobPairingObj.Decrypt(msg)
		require.NoError(t, err)
		require.True(t, bytes.Equal(message, readMessage))

		// Bob writes to Alice
		message = generateRandomBytes(t, 32)
		msg, err = alicePairingObj.Encrypt(message)
		require.NoError(t, err)

		readMessage, err = bobPairingObj.Decrypt(msg)
		require.NoError(t, err)
		require.True(t, bytes.Equal(message, readMessage))
	}

}

func TestPairingObj1ShouldTimeout(t *testing.T) {
	host1, relay1 := createRelayNode(t)
	host2, relay2 := createRelayNode(t)

	defer host1.Close()
	defer host2.Close()
	defer relay1.Stop()
	defer relay2.Stop()

	host1.Peerstore().AddAddr(host2.ID(), tests.GetHostAddress(host2), peerstore.PermanentAddrTTL)
	err := host1.Peerstore().AddProtocols(host2.ID(), relay.WakuRelayID_v200)
	require.NoError(t, err)
	_, err = host1.Network().DialPeer(context.Background(), host2.ID())
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // Wait for relay to form mesh

	bobStaticKey, _ := n.DH25519.GenerateKeypair()
	bobEphemeralKey, _ := n.DH25519.GenerateKeypair()

	bobMessenger, err := NewWakuRelayMessenger(context.Background(), relay1, nil, timesource.NewDefaultClock())
	require.NoError(t, err)

	bobPairingObj, err := NewPairing(bobStaticKey, bobEphemeralKey, WithDefaultResponderParameters(), bobMessenger, utils.Logger())
	require.NoError(t, err)

	aliceStaticKey, _ := n.DH25519.GenerateKeypair()
	aliceEphemeralKey, _ := n.DH25519.GenerateKeypair()

	aliceMessenger, err := NewWakuRelayMessenger(context.Background(), relay2, nil, timesource.NewDefaultClock())
	require.NoError(t, err)

	qrString, qrMessageNameTag := bobPairingObj.PairingInfo()
	alicePairingObj, err := NewPairing(aliceStaticKey, aliceEphemeralKey, WithInitiatorParameters(qrString, qrMessageNameTag), aliceMessenger, utils.Logger())
	require.NoError(t, err)

	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := bobPairingObj.Execute(ctx)
		require.ErrorIs(t, err, ErrPairingTimeout)
	}()

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := alicePairingObj.Execute(ctx)
		require.ErrorIs(t, err, ErrPairingTimeout)
	}()

	wg.Wait()
}
