package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/wakuv2"
	"github.com/status-im/status-go/wakuv2/common"
)

var (
	testContentTopic = "/waku/1/0x12345679/rfc26"
)

func createMockServer(t *testing.T, wg *sync.WaitGroup, expectedType TelemetryType, expectedCondition func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		if r.URL.EscapedPath() != "/record-metrics" {
			t.Errorf("Expected request to '/record-metrics', got '%s'", r.URL.EscapedPath())
		}

		// Check the request body is as expected
		var received []TelemetryRequest
		err := json.NewDecoder(r.Body).Decode(&received)
		if err != nil {
			t.Fatal(err)
		}

		if expectedCondition != nil {
			shouldSucceed, shouldFail := expectedCondition(received)
			if shouldFail {
				w.WriteHeader(http.StatusInternalServerError)
				t.Fail()
				return
			}
			if !shouldSucceed {
				w.WriteHeader(http.StatusOK)
				return
			}
		} else {
			if len(received) != 1 {
				t.Errorf("Unexpected data received: %+v", received)
			} else {
				if received[0].TelemetryType != expectedType {
					t.Errorf("Unexpected telemetry type: got %v, want %v", received[0].TelemetryType, expectedType)
				}
			}
		}
		// If the data is as expected, respond with success
		t.Log("Responding with success")
		responseBody := []map[string]interface{}{
			{"status": "created"},
		}
		body, err := json.Marshal(responseBody)
		if err != nil {
			t.Fatalf("Failed to marshal response body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(body)
		if err != nil {
			t.Fatalf("Failed to write response body: %v", err)
		}
		wg.Done()
	}))
}

func createClient(t *testing.T, mockServerURL string) *Client {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return NewClient(logger, mockServerURL, "testUID", "testNode", "1.0", WithSendPeriod(100*time.Millisecond), WithPeerID("16Uiu2HAkvWiyFsgRhuJEb9JfjYxEkoHLgnUQmr1N5mKWnYjxYRVm"))
}

type expectedCondition func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool)

func withMockServer(t *testing.T, expectedType TelemetryType, expectedCondition expectedCondition, testFunc func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup)) {
	var wg sync.WaitGroup
	wg.Add(1) // Expecting one request

	mockServer := createMockServer(t, &wg, expectedType, expectedCondition)
	defer mockServer.Close()

	client := createClient(t, mockServer.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testFunc(ctx, t, client, &wg)

	// Wait for the request to be received
	wg.Wait()
}

func sendEnvelope(ctx context.Context, client *Client) {
	client.PushSentEnvelope(ctx, wakuv2.SentEnvelope{
		Envelope: v2protocol.NewEnvelope(&pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}, 0, ""),
		PublishMethod: publish.LightPush,
	})
}

func TestClient_ProcessReceivedMessages(t *testing.T) {
	withMockServer(t, ReceivedMessagesMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Create a telemetry request to send
		data := ReceivedMessages{
			Filter: transport.Filter{
				ChatID:       "testChat",
				PubsubTopic:  "testTopic",
				ContentTopic: types.StringToTopic(testContentTopic),
			},
			SSHMessage: &types.Message{
				Hash:      []byte("hash"),
				Timestamp: uint32(time.Now().Unix()),
			},
			Messages: []*v1protocol.StatusMessage{
				{
					ApplicationLayer: v1protocol.ApplicationLayer{
						ID:   types.HexBytes("123"),
						Type: 1,
					},
				},
			},
		}

		// Send the telemetry request
		client.Start(ctx)
		client.PushReceivedMessages(ctx, data)
	})
}

func TestClient_ProcessSentEnvelope(t *testing.T) {
	withMockServer(t, SentEnvelopeMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Send the telemetry request
		client.Start(ctx)
		sendEnvelope(ctx, client)
	})
}

var (
	testENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.status.nodes.status.im"
)

func TestTelemetryUponPublishError(t *testing.T) {
	withMockServer(t, ErrorSendingEnvelopeMetric, nil, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		enrTreeAddress := testENRBootstrap
		envEnrTreeAddress := os.Getenv("ENRTREE_ADDRESS")
		if envEnrTreeAddress != "" {
			enrTreeAddress = envEnrTreeAddress
		}

		wakuConfig := &wakuv2.Config{}
		wakuConfig.Port = 0
		wakuConfig.EnablePeerExchangeClient = true
		wakuConfig.LightClient = true
		wakuConfig.EnableDiscV5 = false
		wakuConfig.DiscV5BootstrapNodes = []string{enrTreeAddress}
		wakuConfig.DiscoveryLimit = 20
		wakuConfig.ClusterID = 16
		wakuConfig.WakuNodes = []string{enrTreeAddress}
		wakuConfig.TelemetryServerURL = client.serverURL
		wakuConfig.TelemetrySendPeriodMs = 500
		w, err := wakuv2.New(nil, "", wakuConfig, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		client.Start(ctx)
		w.SetStatusTelemetryClient(client)

		// Setting this forces the publish function to fail when sending a message
		w.SkipPublishToTopic(true)

		err = w.Start()
		require.NoError(t, err)

		msg := &pb.WakuMessage{
			Payload:      []byte{1, 2, 3, 4, 5},
			ContentTopic: testContentTopic,
			Version:      proto.Uint32(0),
			Timestamp:    proto.Int64(time.Now().Unix()),
		}

		// This should result in a single request sent by the telemetry client
		_, err = w.Send(wakuConfig.DefaultShardPubsubTopic, msg, nil)
		require.NoError(t, err)
	})
}

func TestRetryCache(t *testing.T) {
	counter := 0
	var wg sync.WaitGroup
	wg.Add(2)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}
		if r.URL.EscapedPath() != "/record-metrics" {
			t.Errorf("Expected request to '/record-metrics', got '%s'", r.URL.EscapedPath())
		}

		// Check the request body is as expected
		var received []TelemetryRequest
		err := json.NewDecoder(r.Body).Decode(&received)
		if err != nil {
			t.Fatal(err)
		}

		// Fail for the first request to make telemetry cache grow
		if counter < 1 {
			counter++
			w.WriteHeader(http.StatusInternalServerError)
			wg.Done()
		} else {
			t.Log("Counter reached, responding with success")
			if len(received) == 4 {
				w.WriteHeader(http.StatusCreated)
				responseBody := []map[string]interface{}{
					{"status": "created"},
				}
				body, err := json.Marshal(responseBody)
				if err != nil {
					t.Fatalf("Failed to marshal response body: %v", err)
				}
				w.WriteHeader(http.StatusCreated)
				_, err = w.Write(body)
				if err != nil {
					t.Fatalf("Failed to write response body: %v", err)
				}
				wg.Done()
			} else {
				t.Fatalf("Expected 4 metrics, got %d", len(received)-1)
			}
		}
	}))
	defer mockServer.Close()

	ctx := context.Background()

	client := createClient(t, mockServer.URL)
	client.Start(ctx)

	for i := 0; i < 3; i++ {
		sendEnvelope(ctx, client)
	}

	time.Sleep(110 * time.Millisecond)

	require.Equal(t, 3, len(client.telemetryRetryCache))

	sendEnvelope(ctx, client)

	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, 0, len(client.telemetryRetryCache))
}

func TestRetryCacheCleanup(t *testing.T) {
	ctx := context.Background()

	client := createClient(t, "")

	for i := 0; i < 6000; i++ {
		go sendEnvelope(ctx, client)
		telemetryRequest := <-client.telemetryCh
		client.telemetryCache = append(client.telemetryCache, telemetryRequest)
	}

	err := client.pushTelemetryRequest(client.telemetryCache)
	// For this test case an error when pushing to the server is fine
	require.Error(t, err)

	client.telemetryCache = nil
	require.Equal(t, 6000, len(client.telemetryRetryCache))

	go sendEnvelope(ctx, client)
	telemetryRequest := <-client.telemetryCh
	client.telemetryCache = append(client.telemetryCache, telemetryRequest)

	err = client.pushTelemetryRequest(client.telemetryCache)
	require.Error(t, err)

	telemetryRequests := make([]TelemetryRequest, len(client.telemetryCache))
	copy(telemetryRequests, client.telemetryCache)
	client.telemetryCache = nil

	err = client.pushTelemetryRequest(telemetryRequests)
	require.Error(t, err)

	require.Equal(t, 5001, len(client.telemetryRetryCache))
}

func setDefaultConfig(config *wakuv2.Config, lightMode bool) {
	config.ClusterID = 16

	if lightMode {
		config.EnablePeerExchangeClient = true
		config.LightClient = true
		config.EnableDiscV5 = false
	} else {
		config.EnableDiscV5 = true
		config.EnablePeerExchangeServer = true
		config.LightClient = false
		config.EnablePeerExchangeClient = false
	}
}

var testStoreENRBootstrap = "enrtree://AI4W5N5IFEUIHF5LESUAOSMV6TKWF2MB6GU2YK7PU4TYUGUNOCEPW@store.staging.shards.nodes.status.im"

func TestPeerCount(t *testing.T) {
	// t.Skip("flaky test")

	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		found := slices.ContainsFunc(received, func(req TelemetryRequest) bool {
			t.Log(req)
			return req.TelemetryType == PeerCountMetric
		})
		return found, false
	}
	withMockServer(t, PeerCountMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		config := &wakuv2.Config{}
		setDefaultConfig(config, false)
		config.DiscV5BootstrapNodes = []string{testStoreENRBootstrap}
		config.DiscoveryLimit = 20
		config.TelemetryServerURL = client.serverURL
		config.TelemetrySendPeriodMs = 1500
		config.TelemetryPeerCountSendPeriod = 1500
		w, err := wakuv2.New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		w.SetStatusTelemetryClient(client)
		client.Start(ctx)

		require.NoError(t, w.Start())

		err = tt.RetryWithBackOff(func() error {
			if len(w.Peers()) == 0 {
				return errors.New("no peers discovered")
			}
			return nil
		})

		require.NoError(t, err)

		require.NotEqual(t, 0, len(w.Peers()))
	})
}

func TestPeerId(t *testing.T) {
	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		var data map[string]interface{}

		err := json.Unmarshal(*received[0].TelemetryData, &data)
		if err != nil {
			return false, true
		}

		_, ok := data["peerId"]
		require.True(t, ok)
		return ok, false
	}
	withMockServer(t, SentEnvelopeMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		// Send the telemetry request
		client.Start(ctx)
		sendEnvelope(ctx, client)

	})

}

func TestPeerCountByShard(t *testing.T) {
	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		found := slices.ContainsFunc(received, func(req TelemetryRequest) bool {
			return req.TelemetryType == PeerCountByShardMetric
		})
		return found, false
	}
	withMockServer(t, PeerCountByShardMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		config := &wakuv2.Config{}
		setDefaultConfig(config, false)
		config.DiscV5BootstrapNodes = []string{testStoreENRBootstrap}
		config.DiscoveryLimit = 20
		config.TelemetryServerURL = client.serverURL
		config.TelemetryPeerCountSendPeriod = 1500
		config.TelemetrySendPeriodMs = 1500
		w, err := wakuv2.New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		w.SetStatusTelemetryClient(client)
		client.Start(ctx)

		require.NoError(t, w.Start())

		err = tt.RetryWithBackOff(func() error {
			if len(w.Peers()) == 0 {
				return errors.New("no peers discovered")
			}
			return nil
		})

		require.NoError(t, err)

		require.NotEqual(t, 0, len(w.Peers()))
	})
}

func TestPeerCountByOrigin(t *testing.T) {
	expectedCondition := func(received []TelemetryRequest) (shouldSucceed bool, shouldFail bool) {
		found := slices.ContainsFunc(received, func(req TelemetryRequest) bool {
			return req.TelemetryType == PeerCountByOriginMetric
		})
		return found, false
	}
	withMockServer(t, PeerCountByOriginMetric, expectedCondition, func(ctx context.Context, t *testing.T, client *Client, wg *sync.WaitGroup) {
		config := &wakuv2.Config{}
		setDefaultConfig(config, false)
		config.DiscV5BootstrapNodes = []string{testStoreENRBootstrap}
		config.DiscoveryLimit = 20
		config.TelemetryServerURL = client.serverURL
		config.TelemetryPeerCountSendPeriod = 1500
		config.TelemetrySendPeriodMs = 1500
		w, err := wakuv2.New(nil, "shards.staging", config, nil, nil, nil, nil, nil)
		require.NoError(t, err)

		w.SetStatusTelemetryClient(client)
		client.Start(ctx)

		require.NoError(t, w.Start())

		err = tt.RetryWithBackOff(func() error {
			if len(w.Peers()) == 0 {
				return errors.New("no peers discovered")
			}
			return nil
		})

		require.NoError(t, err)

		require.NotEqual(t, 0, len(w.Peers()))
	})
}

type testCase struct {
	name           string
	input          interface{}
	expectedType   TelemetryType
	expectedFields map[string]interface{}
}

func runTestCase(t *testing.T, tc testCase) {
	ctx := context.Background()
	client := createClient(t, "")

	go client.processAndPushTelemetry(ctx, tc.input)

	telemetryRequest := <-client.telemetryCh

	require.Equal(t, tc.expectedType, telemetryRequest.TelemetryType, "Unexpected telemetry type")

	var telemetryData map[string]interface{}
	err := json.Unmarshal(*telemetryRequest.TelemetryData, &telemetryData)
	require.NoError(t, err, "Failed to unmarshal telemetry data")

	for key, value := range tc.expectedFields {
		require.Equal(t, value, telemetryData[key], "Unexpected value for %s", key)
	}

	require.Contains(t, telemetryData, "nodeName", "Missing nodeName in telemetry data")
	require.Contains(t, telemetryData, "peerId", "Missing peerId in telemetry data")
	require.Contains(t, telemetryData, "statusVersion", "Missing statusVersion in telemetry data")
	require.Contains(t, telemetryData, "deviceType", "Missing deviceType in telemetry data")
	require.Contains(t, telemetryData, "timestamp", "Missing timestamp in telemetry data")

	// Simulate pushing the telemetry request
	client.telemetryCache = append(client.telemetryCache, telemetryRequest)

	err = client.pushTelemetryRequest(client.telemetryCache)
	// For this test case, we expect an error when pushing to the server
	require.Error(t, err)

	// Verify that the request is now in the retry cache
	require.Equal(t, 1, len(client.telemetryRetryCache), "Expected one item in telemetry retry cache")
}

func TestProcessMessageDeliveryConfirmed(t *testing.T) {
	tc := testCase{
		name: "MessageDeliveryConfirmed",
		input: MessageDeliveryConfirmed{
			MessageHash: "0x1234567890abcdef",
		},
		expectedType: MessageDeliveryConfirmedMetric,
		expectedFields: map[string]interface{}{
			"messageHash": "0x1234567890abcdef",
		},
	}
	runTestCase(t, tc)
}

func TestProcessMissedRelevantMessage(t *testing.T) {
	now := time.Now()
	message := common.NewReceivedMessage(
		v2protocol.NewEnvelope(
			&pb.WakuMessage{
				Payload:      []byte{1, 2, 3, 4, 5},
				ContentTopic: testContentTopic,
				Version:      proto.Uint32(0),
				Timestamp:    proto.Int64(now.Unix()),
			}, 0, ""),
		common.MissingMessageType,
	)
	tc := testCase{
		name: "MissedRelevantMessage",
		input: MissedRelevantMessage{
			ReceivedMessage: message,
		},
		expectedType: MissedRelevantMessageMetric,
		expectedFields: map[string]interface{}{
			"messageHash":  message.Envelope.Hash().String(),
			"pubsubTopic":  "",
			"contentTopic": "0x12345679",
		},
	}
	runTestCase(t, tc)
}

func TestProcessMissedMessage(t *testing.T) {
	now := time.Now()
	message := common.NewReceivedMessage(
		v2protocol.NewEnvelope(
			&pb.WakuMessage{
				Payload:      []byte{1, 2, 3, 4, 5},
				ContentTopic: testContentTopic,
				Version:      proto.Uint32(0),
				Timestamp:    proto.Int64(now.Unix()),
			}, 0, ""),
		common.MissingMessageType,
	)
	tc := testCase{
		name: "MissedMessage",
		input: MissedMessage{
			Envelope: message.Envelope,
		},
		expectedType: MissedMessageMetric,
		expectedFields: map[string]interface{}{
			"messageHash":  message.Envelope.Hash().String(),
			"pubsubTopic":  "",
			"contentTopic": message.Envelope.Message().ContentTopic,
		},
	}
	runTestCase(t, tc)
}

func TestProcessDialFailure(t *testing.T) {
	tc := testCase{
		name: "DialFailure",
		input: DialFailure{
			ErrorType: common.ErrorUnknown,
			ErrorMsg:  "test error message",
			Protocols: "test-protocols",
		},
		expectedType: DialFailureMetric,
		expectedFields: map[string]interface{}{
			"errorType": float64(common.ErrorUnknown),
			"errorMsg":  "test error message",
			"protocols": "test-protocols",
		},
	}
	runTestCase(t, tc)
}

func TestProcessSentMessageTotal(t *testing.T) {
	tc := testCase{
		name: "SentMessageTotal",
		input: SentMessageTotal{
			Size: uint32(1234),
		},
		expectedType: SentMessageTotalMetric,
		expectedFields: map[string]interface{}{
			"size": float64(1234),
		},
	}
	runTestCase(t, tc)
}
