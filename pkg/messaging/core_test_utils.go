package messaging

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"testing"
	"time"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// testBackend is the messaging backend a TestMessagingEnvironment shares
// between its cores. newTestBackend picks go-waku or, with the logos_delivery
// build tag, logos-delivery.
type testBackend interface {
	types.Waku
	transport.MessagingAPI
	// SkipPublishToTopic makes sends fail while value is true.
	SkipPublishToTopic(value bool)
}

type TestMessagingEnvironment struct {
	// To enable communication between multiple messaging core instances in tests,
	// share a single Waku instance across them.
	waku *testWakuWrapper
}

func NewTestMessagingEnvironment() (*TestMessagingEnvironment, error) {
	backend, err := newTestBackend()
	if err != nil {
		return nil, err
	}
	waku := &testWakuWrapper{testBackend: backend}
	waku.watchSends()

	return &TestMessagingEnvironment{
		waku: waku,
	}, nil
}

func (f *TestMessagingEnvironment) Setup(t *testing.T) error {
	err := f.waku.testBackend.Start()
	if err != nil {
		return err
	}

	t.Cleanup(func() {
		err = f.waku.testBackend.Stop()
		if err != nil {
			t.Error(err)
		}
	})

	return nil
}

func (f *TestMessagingEnvironment) NewTestCore(params CoreParams, options ...Options) (*Core, error) {
	return newCore(f.waku, params, newConfig(options...))
}

func (f *TestMessagingEnvironment) SubscribePostEvents() chan *PostMessageSubscription {
	return f.waku.SubscribePostEvents()
}

// SetProcessMailserverBatchHook installs a hook that intercepts every store
// (mailserver) batch request issued through this environment. Passing nil
// restores the default behavior of forwarding to the underlying waku.
func (f *TestMessagingEnvironment) SetProcessMailserverBatchHook(hook func(ctx context.Context, batch types.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error) {
	f.waku.processMailserverBatchHook = hook
}

func (f *TestMessagingEnvironment) SetConnectionState(state types.ConnectionState) {
	f.waku.connectionStateMu.Lock()
	defer f.waku.connectionStateMu.Unlock()
	f.waku.connectionStateOverride = &state
}

func (f *TestMessagingEnvironment) SimulateOffline() func() {
	f.waku.SkipPublishToTopic(true)
	return func() {
		f.waku.SkipPublishToTopic(false)
	}
}

// Wraps the backend to provide ability to subscribe to post events. It embeds
// testBackend (not just types.Waku) so that the messaging API methods that live
// on the backend — Send / Subscribe / Unsubscribe / envelope events, i.e.
// transport.MessagingAPI — are promoted too.
type testWakuWrapper struct {
	testBackend
	postSubscriptionsMu     sync.Mutex
	postSubscriptions       []chan *PostMessageSubscription
	posts                   postOrder
	connectionStateMu       sync.RWMutex
	connectionStateOverride *types.ConnectionState

	// processMailserverBatchHook, when set, intercepts every store (mailserver)
	// batch request instead of forwarding it to the underlying waku. Used by
	// tests to observe store queries without a real store node.
	processMailserverBatchHook func(ctx context.Context, batch types.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error
}

func (tw *testWakuWrapper) ConnectionState() types.ConnectionState {
	tw.connectionStateMu.RLock()
	defer tw.connectionStateMu.RUnlock()
	if tw.connectionStateOverride != nil {
		return *tw.connectionStateOverride
	}
	return tw.testBackend.ConnectionState()
}

// Send overrides the embedded Waku's Send to record the order of sends for
// SubscribePostEvents.
func (tw *testWakuWrapper) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) (types.RequestID, error) {
	id, err := tw.testBackend.Send(ctx, pubsubTopic, contentTopic, payload, ephemeral, priority)
	if err != nil {
		return "", err
	}
	tw.publishPosts(tw.posts.sent(id))
	return id, nil
}

// watchSends publishes the wire hash of each send to the post subscribers once
// its outcome arrives, in the order of the sends. An expired send is still
// delivered locally, so it keeps its place when its event has the hash.
func (tw *testWakuWrapper) watchSends() {
	events := make(chan types.EnvelopeEvent, 100)
	tw.testBackend.SubscribeEnvelopeEvents(events)
	go func() {
		defer panics.LogOnPanic()
		for event := range events {
			if event.Event != types.EventEnvelopeSent && event.Event != types.EventEnvelopeExpired {
				continue
			}
			var hash []byte
			if event.Hash != (cryptotypes.Hash{}) {
				hash = event.Hash.Bytes()
			}
			tw.publishPosts(tw.posts.resolved(event.RequestID, hash))
		}
	}()
}

func (tw *testWakuWrapper) publishPosts(hashes [][]byte) {
	tw.postSubscriptionsMu.Lock()
	defer tw.postSubscriptionsMu.Unlock()
	for _, hash := range hashes {
		for _, s := range tw.postSubscriptions {
			select {
			case s <- &PostMessageSubscription{ID: hash}:
			default:
				// subscription channel full
			}
		}
	}
}

// postOrder holds back the outcome of a send until the earlier sends have one.
type postOrder struct {
	mu       sync.Mutex
	pending  []types.RequestID
	outcomes map[types.RequestID][]byte
}

// sent records a send and returns the wire hashes now in order: its outcome
// may arrive before Send returns.
func (o *postOrder) sent(id types.RequestID) [][]byte {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pending = append(o.pending, id)
	return o.flush()
}

// resolved records the wire hash of a send, nil when unknown, and returns the
// wire hashes now in order.
func (o *postOrder) resolved(id types.RequestID, hash []byte) [][]byte {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.outcomes == nil {
		o.outcomes = make(map[types.RequestID][]byte)
	}
	o.outcomes[id] = hash
	return o.flush()
}

// flush returns the wire hashes of the sends at the head of the order whose
// outcome is known. Not thread-safe: the caller holds mu.
func (o *postOrder) flush() [][]byte {
	var hashes [][]byte
	for len(o.pending) > 0 {
		hash, ok := o.outcomes[o.pending[0]]
		if !ok {
			break
		}
		delete(o.outcomes, o.pending[0])
		o.pending = o.pending[1:]
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func (tw *testWakuWrapper) StoreQuery(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	if tw.processMailserverBatchHook != nil {
		return tw.processMailserverBatchHook(ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
	}
	return tw.testBackend.StoreQuery(ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (tw *testWakuWrapper) SubscribePostEvents() chan *PostMessageSubscription {
	subscription := make(chan *PostMessageSubscription, 100)
	tw.postSubscriptionsMu.Lock()
	defer tw.postSubscriptionsMu.Unlock()
	tw.postSubscriptions = append(tw.postSubscriptions, subscription)
	return subscription
}

func (tw *testWakuWrapper) Start() error {
	// No-op, as one waku instance is shared across multiple cores in tests
	// and it must be started only once.
	return nil
}

func (tw *testWakuWrapper) Stop() error {
	// No-op, as one waku instance is shared across multiple cores in tests
	// and it must be stopped only once.
	return nil
}

type PostMessageSubscription struct {
	ID  []byte
	Msg *types.NewMessage
}

type testTimeSource struct {
}

func (ts *testTimeSource) Now() time.Time {
	return time.Now()
}

type TestUtils struct {
	API *API
}

func (t TestUtils) GetAllHRKeysCount(groupID []byte) (int, error) {
	keys, err := t.API.core.stack.Encryption.GetAllHRKeys(groupID)
	if err != nil {
		return 0, err
	}
	if keys == nil {
		return 0, nil
	}
	return len(keys.Keys), nil
}

func (t TestUtils) GetKeysForGroupCount(groupID []byte) (int, error) {
	keys, err := t.API.core.stack.Encryption.GetKeysForGroup(groupID)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

func (t TestUtils) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, theirAPI *API, theirIdentityKey *ecdsa.PrivateKey) error {
	theirBundle, err := theirAPI.core.stack.Encryption.GetBundle(theirIdentityKey)
	if err != nil {
		return err
	}

	_, err = t.API.core.stack.Encryption.ProcessPublicBundle(myIdentityKey, theirBundle)
	return err
}
