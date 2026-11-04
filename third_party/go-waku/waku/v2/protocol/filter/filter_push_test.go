package filter

import (
	"context"
	"strconv"
	"time"

	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestValidPayloadsASCII() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomASCIIString)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsUTF8() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomUTF8String)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsBase64() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomBase64String)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsJSON() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomJSONString)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsURLEncoded() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomURLEncodedString)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsSQL() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare data
	messages := s.prepareData(100, false, false, true, tests.GenerateRandomSQLInsert)

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestLargePayloadsUTF8() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 40*time.Second)

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	// Prepare basic data
	messages := s.prepareData(10, false, false, false, nil)

	// Generate large string
	for i := range messages {
		messages[i].Payload, _ = tests.GenerateRandomUTF8String(153600)
		s.Log.Info("Generated payload with ", zap.String("length", strconv.Itoa(len(messages[i].Payload))))
	}

	// All messages should be received
	s.waitForMessages(messages)

	_, err := s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestFuturePayloadEncryptionVersion() {

	// Subscribe
	s.subscribe(s.TestTopic, s.TestContentTopic, s.FullNodeHost.ID())

	message := tests.CreateWakuMessage(s.TestContentTopic, utils.GetUnixEpoch(), "test_payload")
	futureVersion := uint32(100)
	message.Version = &futureVersion

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.TestTopic))
	s.Require().NoError(err)

	// Should be received
	s.waitForMsg(nil)

	_, err = s.LightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}
