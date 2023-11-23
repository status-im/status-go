package protocol

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/suite"

	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
	waku2 "github.com/status-im/status-go/wakuv2"
)

const testENRBootstrap = "enrtree://AL65EKLJAUXKKPG43HVTML5EFFWEZ7L4LOKTLZCLJASG4DSESQZEC@prod.status.nodes.status.im"

// WaitOnMessengerResponse Wait until the condition is true or the timeout is reached.
func WaitOnMessengerResponse(m *Messenger, condition func(*MessengerResponse) bool, errorMessage string) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	err := tt.RetryWithBackOff(func() error {
		var err error
		r, err := m.RetrieveAll()
		if err != nil {
			panic(err)
		}

		if err := response.Merge(r); err != nil {
			panic(err)
		}

		if err == nil && !condition(response) {
			err = errors.New(errorMessage)
		}
		return err
	})
	return response, err
}

type MessengerSignalsHandlerMock struct {
	MessengerSignalsHandler

	responseChan chan *MessengerResponse
}

func (m *MessengerSignalsHandlerMock) MessengerResponse(response *MessengerResponse) {
	// Non-blocking send
	select {
	case m.responseChan <- response:
	default:
	}
}

func (m *MessengerSignalsHandlerMock) MessageDelivered(chatID string, messageID string) {}

func WaitOnSignaledMessengerResponse(m *Messenger, condition func(*MessengerResponse) bool, errorMessage string) (*MessengerResponse, error) {
	interval := 500 * time.Millisecond
	timeoutChan := time.After(10 * time.Second)

	if m.config.messengerSignalsHandler != nil {
		return nil, errors.New("messengerSignalsHandler already provided/mocked")
	}

	responseChan := make(chan *MessengerResponse, 1)
	m.config.messengerSignalsHandler = &MessengerSignalsHandlerMock{
		responseChan: responseChan,
	}

	defer func() {
		m.config.messengerSignalsHandler = nil
	}()

	for {
		_, err := m.RetrieveAll()
		if err != nil {
			return nil, err
		}

		select {
		case r := <-responseChan:
			if condition(r) {
				return r, nil
			}
			return nil, errors.New(errorMessage)

		case <-timeoutChan:
			return nil, errors.New("timed out: " + errorMessage)

		default: // No immediate response, rest & loop back to retrieve again
			time.Sleep(interval)
		}
	}
}

func FindFirstByContentType(messages []*common.Message, contentType protobuf.ChatMessage_ContentType) *common.Message {
	for _, message := range messages {
		if message.ContentType == contentType {
			return message
		}
	}
	return nil
}

func PairDevices(s *suite.Suite, device1, device2 *Messenger) {
	// Send pairing data
	response, err := device1.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Len(response.Chats(), 1)
	s.False(response.Chats()[0].Active)

	i, ok := device1.allInstallations.Load(device1.installationID)
	s.Require().True(ok)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		device2,
		func(r *MessengerResponse) bool {
			for _, installation := range r.Installations {
				if installation.ID == device1.installationID {
					return installation.InstallationMetadata != nil &&
						i.InstallationMetadata.Name == installation.InstallationMetadata.Name &&
						i.InstallationMetadata.DeviceType == installation.InstallationMetadata.DeviceType
				}
			}
			return false

		},
		"installation not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Ensure installation is enabled
	err = device2.EnableInstallation(device1.installationID)
	s.Require().NoError(err)
}

func SetSettingsAndWaitForChange(s *suite.Suite, messenger *Messenger, timeout time.Duration,
	actionCallback func(), eventCallback func(*SelfContactChangeEvent) bool) {

	allEventsReceived := false
	channel := messenger.SubscribeToSelfContactChanges()
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for !allEventsReceived {
			select {
			case event := <-channel:
				allEventsReceived = eventCallback(event)
			case <-time.After(timeout):
				return
			}
		}
	}()

	actionCallback()

	wg.Wait()

	s.Require().True(allEventsReceived)
}

func SetIdentityImagesAndWaitForChange(s *suite.Suite, messenger *Messenger, timeout time.Duration, actionCallback func()) {
	channel := messenger.SubscribeToSelfContactChanges()
	ok := false
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case event := <-channel:
			if event.ImagesChanged {
				ok = true
			}
		case <-time.After(timeout):
			return
		}
	}()

	actionCallback()

	wg.Wait()

	s.Require().True(ok)
}

func WaitForAvailableStoreNode(s *suite.Suite, m *Messenger, timeout time.Duration) {
	finish := make(chan struct{})
	cancel := make(chan struct{})

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer func() {
			wg.Done()
		}()
		for !m.isActiveMailserverAvailable() {
			select {
			case <-m.SubscribeMailserverAvailable():
			case <-cancel:
				return
			}
		}
	}()

	go func() {
		defer func() {
			close(finish)
		}()
		wg.Wait()
	}()

	select {
	case <-finish:
	case <-time.After(timeout):
		close(cancel)
	}

	s.Require().True(m.isActiveMailserverAvailable())
}

func NewWakuV2(s *suite.Suite, logger *zap.Logger, useLocalWaku bool, enableStore bool) *waku2.Waku {
	wakuConfig := &waku2.Config{
		DefaultShardPubsubTopic: relay.DefaultWakuTopic, // shard.DefaultShardPubsubTopic(),
	}

	var onPeerStats func(connStatus types.ConnStatus)
	var connStatusChan chan struct{}
	var db *sql.DB

	if !useLocalWaku {
		enrTreeAddress := testENRBootstrap
		envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
		if envEnrTreeAddress != "" {
			enrTreeAddress = envEnrTreeAddress
		}

		wakuConfig.EnableDiscV5 = true
		wakuConfig.DiscV5BootstrapNodes = []string{enrTreeAddress}
		wakuConfig.DiscoveryLimit = 20
		wakuConfig.WakuNodes = []string{enrTreeAddress}

		connStatusChan = make(chan struct{})
		terminator := sync.Once{}
		onPeerStats = func(connStatus types.ConnStatus) {
			if connStatus.IsOnline {
				terminator.Do(func() {
					connStatusChan <- struct{}{}
				})
			}
		}
	}

	if enableStore {
		var err error
		db, err = helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
		s.Require().NoError(err)

		wakuConfig.EnableStore = true
		wakuConfig.StoreCapacity = 200
		wakuConfig.StoreSeconds = 200
	}

	wakuNode, err := waku2.New("", "", wakuConfig, logger, db, nil, nil, onPeerStats)
	s.Require().NoError(err)
	s.Require().NoError(wakuNode.Start())

	if !useLocalWaku {
		select {
		case <-time.After(30 * time.Second):
			s.Require().Fail("timeout elapsed")
		case <-connStatusChan:
			// proceed, peers found
			close(connStatusChan)
		}
	}

	return wakuNode
}

func CreateWakuV2Network(s *suite.Suite, parentLogger *zap.Logger, nodeNames []string) []types.Waku {
	nodes := make([]*waku2.Waku, len(nodeNames))
	for i, name := range nodeNames {
		logger := parentLogger.With(zap.String("name", name+"-waku"))
		wakuNode := NewWakuV2(s, logger, true, false)
		nodes[i] = wakuNode
	}

	// Setup local network graph
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			addrs := nodes[j].ListenAddresses()
			s.Require().Greater(len(addrs), 0)
			_, err := nodes[i].AddRelayPeer(addrs[0])
			s.Require().NoError(err)
			err = nodes[i].DialPeer(addrs[0])
			s.Require().NoError(err)
		}
	}
	wrappers := make([]types.Waku, len(nodes))
	for i, n := range nodes {
		wrappers[i] = gethbridge.NewGethWakuV2Wrapper(n)
	}
	return wrappers
}
