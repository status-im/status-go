package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cenkalti/backoff/v3"
	"github.com/waku-org/go-waku/waku/v2/protocol"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type StringGenerator func(maxLength int) (string, error)

// GetHostAddress returns the first listen address used by a host
func GetHostAddress(ha host.Host) multiaddr.Multiaddr {
	return ha.Addrs()[0]
}

// Returns a full multiaddr of host appended by peerID
func GetAddr(h host.Host) multiaddr.Multiaddr {
	id, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().String()))
	var selectedAddr multiaddr.Multiaddr
	//For now skipping circuit relay addresses as libp2p seems to be returning empty p2p-circuit addresses.
	for _, addr := range h.Network().ListenAddresses() {
		if strings.Contains(addr.String(), "p2p-circuit") {
			continue
		}
		selectedAddr = addr
		break
	}
	return selectedAddr.Encapsulate(id)
}

// FindFreePort returns an available port number
func FindFreePort(t *testing.T, host string, maxAttempts int) (int, error) {
	t.Helper()

	if host == "" {
		host = "localhost"
	}

	for i := 0; i < maxAttempts; i++ {
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("unable to resolve tcp addr: %v", err)
			continue
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			l.Close()
			t.Logf("unable to listen on addr %q: %v", addr, err)
			continue
		}

		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		return port, nil

	}

	return 0, fmt.Errorf("no free port found")
}

// FindFreePort returns an available port number
func FindFreeUDPPort(t *testing.T, host string, maxAttempts int) (int, error) {
	t.Helper()

	if host == "" {
		host = "localhost"
	}

	for i := 0; i < maxAttempts; i++ {
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("unable to resolve tcp addr: %v", err)
			continue
		}
		l, err := net.ListenUDP("udp", addr)
		if err != nil {
			l.Close()
			t.Logf("unable to listen on addr %q: %v", addr, err)
			continue
		}

		port := l.LocalAddr().(*net.UDPAddr).Port
		l.Close()
		return port, nil

	}

	return 0, fmt.Errorf("no free port found")
}

// MakeHost creates a Libp2p host with a random key on a specific port
func MakeHost(ctx context.Context, port int, randomness io.Reader) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}

	psWrapper := peerstore.NewWakuPeerstore(ps)
	if err != nil {
		return nil, err
	}

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		libp2p.Peerstore(psWrapper),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

// CreateWakuMessage creates a WakuMessage protobuffer with default values and a custom contenttopic and timestamp
func CreateWakuMessage(contentTopic string, timestamp *int64, optionalPayload ...string) *pb.WakuMessage {
	var payload []byte
	if len(optionalPayload) > 0 {
		payload = []byte(optionalPayload[0])
	} else {
		payload = []byte{1, 2, 3}
	}
	return &pb.WakuMessage{Payload: payload, ContentTopic: contentTopic, Timestamp: timestamp}
}

// RandomHex returns a random hex string of n bytes
func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewLocalnode(priv *ecdsa.PrivateKey, ipAddr *net.TCPAddr, udpPort int, wakuFlags wenr.WakuEnrBitfield, advertiseAddr *net.IP, log *zap.Logger) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}
	localnode := enode.NewLocalNode(db, priv)
	localnode.SetFallbackUDP(udpPort)
	localnode.Set(enr.WithEntry(wenr.WakuENRField, wakuFlags))
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localnode.SetStaticIP(ipAddr.IP)

	if udpPort > 0 && udpPort <= math.MaxUint16 {
		localnode.Set(enr.UDP(uint16(udpPort))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting udpPort", zap.Int("port", udpPort))
	}

	if ipAddr.Port > 0 && ipAddr.Port <= math.MaxUint16 {
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting tcpPort", zap.Int("port", ipAddr.Port))
	}

	if advertiseAddr != nil {
		localnode.SetStaticIP(*advertiseAddr)
	}

	return localnode, nil
}

func CreateHost(t *testing.T, opts ...config.Option) (host.Host, int, *ecdsa.PrivateKey) {
	privKey, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	sPrivKey := libp2pcrypto.PrivKey(utils.EcdsaPrivKeyToSecp256k1PrivKey(privKey))

	port, err := FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)

	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	require.NoError(t, err)

	opts = append(opts, libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(sPrivKey))

	host, err := libp2p.New(opts...)
	require.NoError(t, err)

	return host, port, privKey
}

func ExtractIP(addr multiaddr.Multiaddr) (*net.TCPAddr, error) {
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return nil, err
	}

	portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{
		IP:   net.ParseIP(ipStr),
		Port: port,
	}, nil
}

func RandomInt(min, max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomASCIIString(maxLength int) (string, error) {
	length, err := rand.Int(rand.Reader, big.NewInt(int64(maxLength)))
	if err != nil {
		return "", err
	}
	length.SetInt64(length.Int64() + 1)

	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length.Int64())
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[num.Int64()]
	}

	return string(result), nil
}

func GenerateRandomUTF8String(maxLength int) (string, error) {
	length, err := rand.Int(rand.Reader, big.NewInt(int64(maxLength)))
	if err != nil {
		return "", err
	}
	length.SetInt64(length.Int64() + 1)

	var (
		runes      []rune
		start, end int
	)

	// Define unicode range
	start = 0x0020 // Space character
	end = 0x007F   // Tilde (~)

	for i := 0; int64(i) < length.Int64(); i++ {
		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(end-start+1)))
		if err != nil {
			return "", err
		}
		char := rune(start + int(randNum.Int64()))
		if !utf8.ValidRune(char) {
			continue
		}
		runes = append(runes, char)
	}

	return string(runes), nil
}

func GenerateRandomJSONString(maxLength int) (string, error) {
	// With 5 key-value pairs
	m := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		key, err := GenerateRandomASCIIString(20)
		if err != nil {
			return "", err
		}
		value, err := GenerateRandomASCIIString(maxLength)
		if err != nil {
			return "", err
		}

		m[key] = value
	}

	// Marshal the map into a JSON string
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(m)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateRandomBase64String(maxLength int) (string, error) {
	bytes, err := RandomBytes(maxLength)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}

func GenerateRandomURLEncodedString(maxLength int) (string, error) {
	randomString, err := GenerateRandomASCIIString(maxLength)
	if err != nil {
		return "", err
	}

	// URL-encode the random string
	return url.QueryEscape(randomString), nil
}

func GenerateRandomSQLInsert(maxLength int) (string, error) {
	// Random table name
	tableName, err := GenerateRandomASCIIString(10)
	if err != nil {
		return "", err
	}

	// Random column names
	columnCount, err := RandomInt(3, 6)
	if err != nil {
		return "", err
	}
	columnNames := make([]string, columnCount)
	for i := 0; i < columnCount; i++ {
		columnName, err := GenerateRandomASCIIString(maxLength)
		if err != nil {
			return "", err
		}
		columnNames[i] = columnName
	}

	// Random values
	values := make([]string, columnCount)
	for i := 0; i < columnCount; i++ {
		value, err := GenerateRandomASCIIString(maxLength)
		if err != nil {
			return "", err
		}
		values[i] = "'" + value + "'"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(values, ", "))

	return query, nil
}

func WaitForMsg(t *testing.T, timeout time.Duration, wg *sync.WaitGroup, ch chan *protocol.Envelope) {
	wg.Add(1)
	log := utils.Logger()
	go func() {
		defer wg.Done()
		select {
		case env := <-ch:
			msg := env.Message()
			log.Info("Received ", zap.String("msg", msg.String()))
		case <-time.After(timeout):
			require.Fail(t, "Message timeout")
		}
	}()
	wg.Wait()
}

func WaitForTimeout(t *testing.T, ctx context.Context, timeout time.Duration, wg *sync.WaitGroup, ch chan *protocol.Envelope) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case _, ok := <-ch:
			require.False(t, ok, "should not retrieve message")
		case <-time.After(timeout):
			// All good
		case <-ctx.Done():
			require.Fail(t, "test exceeded allocated time")
		}
	}()

	wg.Wait()
}

type BackOffOption func(*backoff.ExponentialBackOff)

func RetryWithBackOff(o func() error, options ...BackOffOption) error {
	b := backoff.ExponentialBackOff{
		InitialInterval:     time.Millisecond * 100,
		RandomizationFactor: 0.1,
		Multiplier:          1,
		MaxInterval:         time.Second,
		MaxElapsedTime:      time.Second * 10,
		Clock:               backoff.SystemClock,
	}
	for _, option := range options {
		option(&b)
	}
	b.Reset()
	return backoff.Retry(o, &b)
}
