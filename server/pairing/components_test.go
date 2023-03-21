package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/server"
	"github.com/stretchr/testify/require"
)

type TestPairingServerComponents struct {
	EphemeralPK  *ecdsa.PrivateKey
	EphemeralAES []byte
	OutboundIP   net.IP
	Cert         tls.Certificate
	SS           *SenderServer
	RS           *ReceiverServer
}

func (tpsc *TestPairingServerComponents) SetupPairingServerComponents(t *testing.T) {
	var err error

	// Get 4 key components for tls.cert generation
	// 1) Ephemeral private key
	tpsc.EphemeralPK, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// 2) AES encryption key
	tpsc.EphemeralAES, err = common.MakeECDHSharedKey(tpsc.EphemeralPK, &tpsc.EphemeralPK.PublicKey)
	require.NoError(t, err)

	// 3) Device outbound IP address
	tpsc.OutboundIP, err = server.GetOutboundIP()
	require.NoError(t, err)

	// Generate tls.Certificate and Server
	tpsc.Cert, _, err = GenerateCertFromKey(tpsc.EphemeralPK, time.Now(), tpsc.OutboundIP.String())
	require.NoError(t, err)

	sc := &ServerConfig{
		PK:       &tpsc.EphemeralPK.PublicKey,
		EK:       tpsc.EphemeralAES,
		Cert:     &tpsc.Cert,
		Hostname: tpsc.OutboundIP.String(),
	}

	tpsc.SS, err = NewSenderServer(nil, &SenderServerConfig{ServerConfig: sc, SenderConfig: &SenderConfig{}})
	require.NoError(t, err)
	tpsc.RS, err = NewReceiverServer(nil, &ReceiverServerConfig{ServerConfig: sc, ReceiverConfig: &ReceiverConfig{}})
	require.NoError(t, err)
}

type MockPayloadReceiver struct {
	encryptor *PayloadEncryptor
}

func NewMockPayloadReceiver(aesKey []byte) *MockPayloadReceiver {
	return &MockPayloadReceiver{NewPayloadEncryptor(aesKey)}
}

func (m *MockPayloadReceiver) Receive(data []byte) error {
	return m.encryptor.decrypt(data)
}

func (m *MockPayloadReceiver) Received() []byte {
	return m.encryptor.getDecrypted()
}

func (m *MockPayloadReceiver) LockPayload() {}

type MockPayloadMounter struct {
	encryptor *PayloadEncryptor
}

func NewMockPayloadMounter(aesKey []byte) *MockPayloadMounter {
	return &MockPayloadMounter{NewPayloadEncryptor(aesKey)}
}

func (m *MockPayloadMounter) Mount() error {
	// Make a random payload
	data := make([]byte, 32)
	_, err := rand.Read(data)
	if err != nil {
		return err
	}

	return m.encryptor.encrypt(data)
}

func (m *MockPayloadMounter) ToSend() []byte {
	return m.encryptor.getEncrypted()
}

func (m *MockPayloadMounter) LockPayload() {
	m.encryptor.lockPayload()
}
