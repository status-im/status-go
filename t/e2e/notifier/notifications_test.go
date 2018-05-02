package whisper

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	sdk "github.com/status-im/status-go-sdk"
	"github.com/status-im/status-go/geth/api"
	notifier "github.com/status-im/status-go/notifier"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}

type NotifierTestSuite struct {
	e2e.BackendTestSuite
}

const discoveryTopic = "notifier"

func (s *NotifierTestSuite) TestPushNotificationServerSubscriptionProcess() {
	//
	// Arrange (1st step, broadcast PN server availability and have clients listen to broadcast)
	//
	dataRootDir, err := ioutil.TempDir("", "test-notif-")
	s.Require().NoError(err)

	var pnServerBackend *api.StatusBackend
	if pnServerBackend = notifier.NewStatusBackend(filepath.Join(dataRootDir, "pnserver"), "", uint64(GetNetworkID())); pnServerBackend == nil {
		s.FailNow("Couldn't setup the node")
	}
	defer func() {
		_ = pnServerBackend.StopNode()
		_ = os.RemoveAll(dataRootDir)
	}()
	pnServerNode := pnServerBackend.StatusNode()

	m, err := notifier.NewMessenger(newRPCClient(pnServerBackend), notificationProviderMock{}, discoveryTopic, 5*time.Second)
	if err != nil {
		s.FailNow("Error while creating the PN server:", err.Error())
	}

	pnServerAccount, err := pnServerBackend.AccountManager().SelectedAccount()
	s.Require().NoError(err)
	pnServerPubKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSAPub(&pnServerAccount.AccountKey.PrivateKey.PublicKey))

	aliceBackend, stopAliceNode := s.startBackend("alice", dataRootDir)
	defer stopAliceNode()

	// Add peer so both peers can see each other
	aliceGethNode := aliceBackend.StatusNode().GethNode()
	aliceEnode := aliceGethNode.Server().NodeInfo().Enode
	err = pnServerNode.AddPeer(aliceEnode)
	s.Require().NoError(err)

	aliceClient := sdk.New(newRPCClient(aliceBackend))
	_, _, _, err = aliceClient.SignupAndLogin("")
	s.Require().NoError(err)

	pubChannel, err := aliceClient.JoinPublicChannel(discoveryTopic)
	s.Require().NoError(err)
	defer pubChannel.Close()
	msgArrivedCh := make(chan *sdk.Msg)

	//
	// Act (1st step, broadcast PN server availability and have clients listen to broadcast)
	//
	if err := m.BroadcastAvailability(); err != nil {
		s.FailNowf("Could not broadcast PN server availability: %s", err.Error())
	}

	_, err = pubChannel.Subscribe(func(msg *sdk.Msg) { msgArrivedCh <- msg })
	s.Require().NoError(err)

	//
	// Assert (1st step, broadcast PN server availability and have clients listen to broadcast)
	//

	// Ensure that we have two different nodes
	s.Require().NotEqual(pnServerNode.GethNode().Server().NodeInfo().Enode, aliceEnode)

	// Ensure that 'alice' node receives 'new contact key' message
	select {
	case msg := <-msgArrivedCh:
		s.Require().Equal(sdk.PNBroadcastAvailabilityType, msg.Type, "Type of the message received from PN server is not 'PNBroadcastAvailabilityType'")
		s.Require().Equal(discoveryTopic, msg.ChannelName, "Message should arrive on the expected topic")

		s.Require().IsType(&sdk.PNBroadcastAvailabilityMsg{}, msg.Properties)
		bcastMsg := msg.Properties.(*sdk.PNBroadcastAvailabilityMsg)
		s.Require().Equal(pnServerPubKeyHex, bcastMsg.Pubkey, "PN server broadcast should contain pubkey of AK1 (see diagram)")

		// TODO (pombeirp): Assert that PN server broadcast contains expected occupancy value (0%)
	case <-time.After(15 * time.Second):
		s.FailNow("Timeout waiting for message to arrive from PN server to 'alice' node")
	}

	//
	// Arrange (2nd step, have client reply to PN server availability broadcast)
	//

	//
	// Act (2nd step, have client reply to PN server availability broadcast)
	//

	// TODO (pombeirp): Have PN server listen for registration requests
	// TODO (pombeirp): Have aliceMessenger send anonymous message to PN server using SK1, containing a device registration token
	// go m.ManageRegistrations()

	//
	// Assert (2nd step, have client reply to PN server availability broadcast)
	//

	// TODO (pombeirp): Assert that PN server registered 'alice' correctly

	//
	// Arrange (3rd step, have client listen for PN server registration response)
	//

	//
	// Act (3rd step, have client listen for PN server registration response)
	//

	//
	// Assert (3rd step, have client listen for PN server registration response)
	//
}

// Start status node.
func (s *NotifierTestSuite) startBackend(name string, parentDir string) (*api.StatusBackend, func()) {
	datadir := filepath.Join(parentDir, name)
	backend := api.NewStatusBackend()
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	nodeConfig.DataDir = datadir
	nodeConfig.StatusServiceEnabled = true
	s.Require().NoError(err)
	s.Require().False(backend.IsNodeRunning())

	nodeConfig.WhisperConfig.LightClient = true

	s.Require().NoError(backend.StartNode(nodeConfig))
	s.Require().True(backend.IsNodeRunning())

	return backend, func() {
		s.True(backend.IsNodeRunning())
		s.NoError(backend.StopNode())
		s.False(backend.IsNodeRunning())
		err = os.RemoveAll(datadir)
		s.Require().NoError(err)
	}
}

type notificationProviderMock struct {
}

// Send : Sends a push notification to given devices
func (n notificationProviderMock) Send(tokens []string, message string) error {
	return nil
}

type rpcClient struct {
	b *api.StatusBackend
}

func newRPCClient(b *api.StatusBackend) sdk.RPCClient {
	return &rpcClient{b: b}
}

func (c rpcClient) Call(request interface{}) (response interface{}, err error) {
	response = c.b.CallPrivateRPC(request.(string))
	return
}
