package api

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	m_common "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/wakuv2"

	"github.com/stretchr/testify/suite"
)

type MessengerRawMessageResendTest struct {
	suite.Suite
	aliceBackend   *GethStatusBackend
	bobBackend     *GethStatusBackend
	aliceMessenger *protocol.Messenger
	bobMessenger   *protocol.Messenger
	// add exchangeBootNode to ensure alice and bob can find each other.
	// If relying on in the fleet, the test will likely be flaky
	exchangeBootNode *wakuv2.Waku
}

func TestMessengerRawMessageResendTestSuite(t *testing.T) {
	suite.Run(t, new(MessengerRawMessageResendTest))
}

func (s *MessengerRawMessageResendTest) SetupTest() {
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)

	exchangeNodeConfig := &wakuv2.Config{
		Port:                     0,
		EnableDiscV5:             true,
		EnablePeerExchangeServer: true,
		ClusterID:                16,
		UseShardAsDefaultTopic:   true,
		DefaultShardPubsubTopic:  shard.DefaultShardPubsubTopic(),
	}
	s.exchangeBootNode, err = wakuv2.New("", "", exchangeNodeConfig, logger.Named("pxServerNode"), nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(s.exchangeBootNode.Start())

	s.createAliceBobBackendAndLogin()
	community := s.createTestCommunity(s.aliceMessenger, protobuf.CommunityPermissions_MANUAL_ACCEPT)
	s.addMutualContact()
	advertiseCommunityToUserOldWay(&s.Suite, community, s.aliceMessenger, s.bobMessenger)
	requestBob := &requests.RequestToJoinCommunity{
		CommunityID: community.ID(),
	}
	joinOnRequestCommunity(&s.Suite, community, s.aliceMessenger, s.bobMessenger, requestBob)
}

func (s *MessengerRawMessageResendTest) TearDownTest() {
	// Initialize a map to keep track of the operation status.
	operationStatus := map[string]bool{
		"Alice Logout":   false,
		"Bob Logout":     false,
		"Boot Node Stop": false,
	}

	done := make(chan string, 3) // Buffered channel to receive the names of the completed operations
	errs := make(chan error, 3)  // Channel to receive errs from operations

	// Asynchronously perform operations and report completion or errs.
	go func() {
		err := s.aliceBackend.Logout()
		if err != nil {
			errs <- err
		}
		done <- "Alice Logout"
	}()

	go func() {
		err := s.bobBackend.Logout()
		if err != nil {
			errs <- err
		}
		done <- "Bob Logout"
	}()

	go func() {
		err := s.exchangeBootNode.Stop()
		if err != nil {
			errs <- err
		}
		done <- "Boot Node Stop"
	}()

	timeout := time.After(30 * time.Second)
	operationsCompleted := 0

	for operationsCompleted < 3 {
		select {
		case opName := <-done:
			s.T().Logf("%s completed successfully.", opName)
			operationStatus[opName] = true
			operationsCompleted++
		case err := <-errs:
			s.Require().NoError(err)
		case <-timeout:
			// If a timeout occurs, check which operations have not reported completion.
			s.T().Errorf("Timeout occurred, the following operations did not complete in time:")
			for opName, completed := range operationStatus {
				if !completed {
					s.T().Errorf("%s is still pending.", opName)
				}
			}
			s.T().FailNow()
		}
	}
}

func (s *MessengerRawMessageResendTest) createAliceBobBackendAndLogin() {
	pxServerNodeENR, err := s.exchangeBootNode.GetNodeENRString()
	s.Require().NoError(err)
	// we don't support multiple logger instances, so just share the log dir
	shareLogDir := filepath.Join(s.T().TempDir(), "logs")
	s.T().Logf("shareLogDir: %s", shareLogDir)
	s.createBackendAndLogin(&s.aliceBackend, &s.aliceMessenger, "alice66", pxServerNodeENR, shareLogDir)
	s.createBackendAndLogin(&s.bobBackend, &s.bobMessenger, "bob66", pxServerNodeENR, shareLogDir)

	aliceWaku := s.aliceBackend.StatusNode().WakuV2Service()
	bobWaku := s.bobBackend.StatusNode().WakuV2Service()
	// NOTE: default MaxInterval is 10s, which is too short for the test
	// TODO(frank) figure out why it takes so long for the peers to know each other
	err = tt.RetryWithBackOff(func() error {
		if len(aliceWaku.Peerstore().Addrs(bobWaku.PeerID())) > 0 {
			return nil
		}
		s.T().Logf("alice don't know bob's addresses")
		return errors.New("alice don't know bob's addresses")
	}, func(b *backoff.ExponentialBackOff) { b.MaxInterval = 20 * time.Second })
	s.Require().NoError(err)
	err = tt.RetryWithBackOff(func() error {
		if len(bobWaku.Peerstore().Addrs(aliceWaku.PeerID())) > 0 {
			return nil
		}
		s.T().Logf("bob don't know alice's addresses")
		return errors.New("bob don't know alice's addresses")
	}, func(b *backoff.ExponentialBackOff) { b.MaxInterval = 20 * time.Second })
	s.Require().NoError(err)
}

func (s *MessengerRawMessageResendTest) createBackendAndLogin(backend **GethStatusBackend, messenger **protocol.Messenger, displayName, pxServerNodeENR, shareLogDir string) {
	*backend = NewGethStatusBackend()
	rootDir := filepath.Join(s.T().TempDir())
	s.T().Logf("%s rootDir: %s", displayName, rootDir)
	createAccountRequest := s.setCreateAccountRequest(displayName, rootDir, shareLogDir)
	_, err := (*backend).CreateAccountAndLogin(createAccountRequest,
		params.WithDiscV5BootstrapNodes([]string{pxServerNodeENR}),
		// override fleet nodes
		params.WithWakuNodes([]string{}))
	s.Require().NoError(err)
	*messenger = (*backend).Messenger()
	s.Require().NotNil(messenger)
	_, err = (*messenger).Start()
	s.Require().NoError(err)
}

func (s *MessengerRawMessageResendTest) setCreateAccountRequest(displayName, backupDisabledDataDir, logFilePath string) *requests.CreateAccount {
	nameServer := "1.1.1.1"
	verifyENSContractAddress := "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"
	verifyTransactionChainID := int64(1)
	verifyURL := "https://eth-archival.rpc.grove.city/v1/3ef2018191814b7e1009b8d9"
	logLevel := "DEBUG"
	networkID := uint64(1)
	password := "qwerty"
	return &requests.CreateAccount{
		UpstreamConfig:           verifyURL,
		WakuV2Nameserver:         &nameServer,
		VerifyENSContractAddress: &verifyENSContractAddress,
		BackupDisabledDataDir:    backupDisabledDataDir,
		Password:                 password,
		DisplayName:              displayName,
		LogEnabled:               true,
		VerifyTransactionChainID: &verifyTransactionChainID,
		VerifyTransactionURL:     &verifyURL,
		VerifyENSURL:             &verifyURL,
		LogLevel:                 &logLevel,
		LogFilePath:              logFilePath,
		NetworkID:                &networkID,
		CustomizationColor:       string(m_common.CustomizationColorPrimary),
	}
}

// TestMessageSent tests if ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN is in state `sent` without resending
func (s *MessengerRawMessageResendTest) TestMessageSent() {
	ids, err := s.bobMessenger.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN)
	s.Require().NoError(err)
	s.Require().Len(ids, 1)

	err = tt.RetryWithBackOff(func() error {
		rawMessage, err := s.bobMessenger.RawMessageByID(ids[0])
		s.Require().NoError(err)
		s.Require().NotNil(rawMessage)
		if rawMessage.Sent {
			return nil
		}
		return errors.New("raw message should be sent finally")
	})
	s.Require().NoError(err)
}

// TestMessageResend tests if ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN is resent
func (s *MessengerRawMessageResendTest) TestMessageResend() {
	ids, err := s.bobMessenger.RawMessagesIDsByType(protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN)
	s.Require().NoError(err)
	s.Require().Len(ids, 1)
	rawMessage, err := s.bobMessenger.RawMessageByID(ids[0])
	s.Require().NoError(err)
	s.Require().NotNil(rawMessage)
	s.Require().NoError(s.bobMessenger.UpdateRawMessageSent(rawMessage.ID, false, 0))
	err = tt.RetryWithBackOff(func() error {
		rawMessage, err := s.bobMessenger.RawMessageByID(ids[0])
		s.Require().NoError(err)
		s.Require().NotNil(rawMessage)
		if !rawMessage.Sent {
			return errors.New("message ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN was not resent yet")
		}
		return nil
	})
	s.Require().NoError(err)

	waitOnMessengerResponse(&s.Suite, func(r *protocol.MessengerResponse) error {
		if len(r.RequestsToJoinCommunity()) > 0 {
			return nil
		}
		return errors.New("community request to join not received")
	}, s.aliceMessenger)
}

// To be removed in https://github.com/status-im/status-go/issues/4437
func advertiseCommunityToUserOldWay(s *suite.Suite, community *communities.Community, alice *protocol.Messenger, bob *protocol.Messenger) {
	chat := protocol.CreateOneToOneChat(bob.IdentityPublicKeyString(), bob.IdentityPublicKey(), bob.GetTransport())

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err := alice.SaveChat(chat)
	s.Require().NoError(err)
	_, err = alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Ensure community is received
	response, err := protocol.WaitOnMessengerResponse(
		bob,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"bob did not receive community request to join",
	)
	s.Require().NoError(err)
	communityInResponse := response.Communities()[0]
	s.Require().Equal(community.ID(), communityInResponse.ID())
}

func (s *MessengerRawMessageResendTest) addMutualContact() {
	bobPubkey := s.bobMessenger.IdentityPublicKeyCompressed()
	bobZQ3ID, err := utils.SerializePublicKey(bobPubkey)
	s.Require().NoError(err)
	mr, err := s.aliceMessenger.AddContact(context.Background(), &requests.AddContact{
		ID:          bobZQ3ID,
		DisplayName: "bob666",
	})
	s.Require().NoError(err)
	s.Require().Len(mr.Messages(), 2)

	var contactRequest *common.Message
	waitOnMessengerResponse(&s.Suite, func(r *protocol.MessengerResponse) error {
		for _, m := range r.Messages() {
			if m.GetContentType() == protobuf.ChatMessage_CONTACT_REQUEST {
				contactRequest = m
				return nil
			}
		}
		return errors.New("contact request not received")
	}, s.bobMessenger)

	mr, err = s.bobMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{
		ID: types.FromHex(contactRequest.ID),
	})
	s.Require().NoError(err)
	s.Require().Len(mr.Contacts, 1)

	waitOnMessengerResponse(&s.Suite, func(r *protocol.MessengerResponse) error {
		if len(r.Contacts) > 0 {
			return nil
		}
		return errors.New("contact accepted not received")
	}, s.aliceMessenger)
}

type MessageResponseValidator func(*protocol.MessengerResponse) error

func waitOnMessengerResponse(s *suite.Suite, fnWait MessageResponseValidator, user *protocol.Messenger) {
	_, err := protocol.WaitOnMessengerResponse(
		user,
		func(r *protocol.MessengerResponse) bool {
			err := fnWait(r)
			if err != nil {
				s.T().Logf("response error: %s", err.Error())
			}
			return err == nil
		},
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
}

func requestToJoinCommunity(s *suite.Suite, controlNode *protocol.Messenger, user *protocol.Messenger, request *requests.RequestToJoinCommunity) types.HexBytes {
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin := response.RequestsToJoinCommunity()[0]
	s.Require().Equal(requestToJoin.PublicKey, user.IdentityPublicKeyString())

	_, err = protocol.WaitOnMessengerResponse(
		controlNode,
		func(r *protocol.MessengerResponse) bool {
			if len(r.RequestsToJoinCommunity()) == 0 {
				return false
			}

			for _, resultRequest := range r.RequestsToJoinCommunity() {
				if resultRequest.PublicKey == user.IdentityPublicKeyString() {
					return true
				}
			}
			return false
		},
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)

	return requestToJoin.ID
}

func joinOnRequestCommunity(s *suite.Suite, community *communities.Community, controlNode *protocol.Messenger, user *protocol.Messenger, request *requests.RequestToJoinCommunity) {
	// Request to join the community
	requestToJoinID := requestToJoinCommunity(s, controlNode, user, request)

	// accept join request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoinID}
	response, err := controlNode.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	updatedCommunity := response.Communities()[0]
	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(user.IdentityPublicKey()))

	// receive request to join response
	_, err = protocol.WaitOnMessengerResponse(
		user,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(user.IdentityPublicKey())
		},
		"user did not receive request to join response",
	)
	s.Require().NoError(err)

	userCommunity, err := user.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(userCommunity.HasMember(user.IdentityPublicKey()))

	_, err = protocol.WaitOnMessengerResponse(
		controlNode,
		func(r *protocol.MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(user.IdentityPublicKey())
		},
		"control node did not receive request to join response",
	)
	s.Require().NoError(err)
}

func (s *MessengerRawMessageResendTest) createTestCommunity(controlNode *protocol.Messenger, membershipType protobuf.CommunityPermissions_Access) *communities.Community {
	description := &requests.CreateCommunity{
		Membership:                  membershipType,
		Name:                        "status",
		Color:                       "#ffffff",
		Description:                 "status community description",
		PinMessageAllMembersEnabled: false,
	}
	response, err := controlNode.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	return response.Communities()[0]
}
