package discovery

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/stretchr/testify/require"
)

func newRegistry() *registry {
	return &registry{
		storage: map[string][]int{},
	}
}

type registry struct {
	mu      sync.Mutex
	storage map[string][]int
}

func (r *registry) Add(topic string, id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storage[topic] = append(r.storage[topic], id)
}

func (r *registry) Get(topic string) []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.storage[topic]
}

type fake struct {
	started  bool
	err      error
	id       int
	registry *registry
}

func (f *fake) Start() error {
	if f.err != nil {
		return f.err
	}
	f.started = true
	return nil
}

func (f *fake) Stop() error {
	f.started = false
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fake) Running() bool {
	return f.started
}

func (f *fake) Register(topic string, stop chan struct{}) error {
	if f.err != nil {
		return f.err
	}
	f.registry.Add(topic, f.id)
	return nil
}

func (f *fake) Discover(topic string, period <-chan time.Duration, found chan<- *discv5.Node, lookup chan<- bool) error {
	if f.err != nil {
		return f.err
	}
	for _, n := range f.registry.Get(topic) {
		found <- discv5.NewNode(discv5.NodeID{byte(n)}, nil, 0, 0)
	}
	return nil
}

type testErrorCase struct {
	desc   string
	errors []error
}

func errorCases() []testErrorCase {
	return []testErrorCase{
		{desc: "SingleError", errors: []error{nil, errors.New("test")}},
		{desc: "NoErrors", errors: []error{nil, nil}},
		{desc: "AllErrors", errors: []error{errors.New("test"), errors.New("test")}},
	}
}

func TestMuxerStart(t *testing.T) {
	for _, tc := range errorCases() {
		t.Run(tc.desc, func(t *testing.T) {
			discoveries := make([]Discovery, len(tc.errors))
			erred := false
			for i, err := range tc.errors {
				if err != nil {
					erred = true
				}
				discoveries[i] = &fake{err: err}
			}
			muxer := NewMultiplexer(discoveries)
			if erred {
				require.Error(t, muxer.Start())
			} else {
				require.NoError(t, muxer.Start())
			}
			for _, d := range discoveries {
				require.Equal(t, !erred, d.Running())
			}
		})
	}
}

func TestMuxerStop(t *testing.T) {
	for _, tc := range errorCases() {
		t.Run(tc.desc, func(t *testing.T) {
			discoveries := make([]Discovery, len(tc.errors))
			erred := false
			for i, err := range tc.errors {
				if err != nil {
					erred = true
				}
				discoveries[i] = &fake{started: true, err: err}
			}
			muxer := NewMultiplexer(discoveries)
			if erred {
				require.Error(t, muxer.Stop())
			} else {
				require.NoError(t, muxer.Stop())
			}
			for _, d := range discoveries {
				require.False(t, d.Running())
			}
		})
	}
}

func TestMuxerRunning(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		started []bool
	}{
		{desc: "FirstRunning", started: []bool{false, true}},
		{desc: "SecondRunning", started: []bool{true, false}},
		{desc: "AllRunning", started: []bool{true, true}},
		{desc: "NoRunning", started: []bool{false, false}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			discoveries := make([]Discovery, len(tc.started))
			allstarted := false
			for i, start := range tc.started {
				allstarted = start || allstarted
				discoveries[i] = &fake{started: start}
			}
			require.Equal(t, allstarted, NewMultiplexer(discoveries).Running())
		})
	}
}

func TestMuxerRegister(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		errors []error
		topics []string
	}{
		{"NoErrors", []error{nil, nil, nil}, []string{"a"}},
		{"MultipleTopics", []error{nil, nil, nil}, []string{"a", "b", "c"}},
		{"SingleError", []error{nil, errors.New("test"), nil}, []string{"a"}},
		{"AllErrors", []error{errors.New("test"), errors.New("test"), errors.New("test")}, []string{"a"}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			reg := newRegistry()
			discoveries := make([]Discovery, len(tc.errors))
			erred := 0
			for i := range discoveries {
				if tc.errors[i] != nil {
					erred++
				}
				discoveries[i] = &fake{id: i, err: tc.errors[i], registry: reg}
			}
			muxer := NewMultiplexer(discoveries)
			for _, topic := range tc.topics {
				if erred != 0 {
					require.Error(t, muxer.Register(topic, nil))
				} else {
					require.NoError(t, muxer.Register(topic, nil))
				}
				require.Equal(t, len(discoveries)-erred, len(reg.Get(topic)))
			}
		})
	}
}

func TestMuxerDiscovery(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		errors []error
		topics []string
		ids    [][]int
	}{
		{"EqualNoErrors", []error{nil, nil}, []string{"a"}, [][]int{{11, 22, 33}, {44, 55, 66}}},
		{"MultiTopicsSingleSource", []error{nil, nil}, []string{"a", "b"}, [][]int{{11, 22, 33}, {}}},
		{"SingleError", []error{nil, errors.New("test")}, []string{"a"}, [][]int{{11, 22, 33}, {44, 55, 66}}},
		{"AllErrors", []error{errors.New("test"), errors.New("test")}, []string{"a"}, [][]int{{11, 22, 33}, {44, 55, 66}}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			discoveries := make([]Discovery, len(tc.errors))
			erred := false
			expected := 0
			for i := range discoveries {
				if tc.errors[i] == nil {
					expected += len(tc.ids[i])
				} else {
					erred = true
				}
				reg := newRegistry()
				discoveries[i] = &fake{id: i, err: tc.errors[i], registry: reg}
				for _, topic := range tc.topics {
					for _, id := range tc.ids[i] {
						reg.Add(topic, id)
					}
				}
			}
			muxer := NewMultiplexer(discoveries)
			for _, topic := range tc.topics {
				found := make(chan *discv5.Node, expected)
				period := make(chan time.Duration)
				close(period)
				if erred {
					// TODO test period channel
					require.Error(t, muxer.Discover(topic, period, found, nil))
				} else {
					require.NoError(t, muxer.Discover(topic, period, found, nil))
				}
				close(found)
				count := 0
				for range found {
					count++
				}
				require.Equal(t, expected, count)
			}
		})
	}
}
