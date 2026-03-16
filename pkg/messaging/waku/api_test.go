// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package wakuv2

import (
	"maps"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/messaging/waku/common"
	types2 "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestRunPausedPollingLoopSkipsWhenPausedAndResumes(t *testing.T) {
	lifecycleCh := make(chan bool, 2)
	tickerCh := make(chan time.Time, 2)
	stopCh := make(chan error, 1)

	var ticks atomic.Int32
	stopped := make(chan struct{}, 1)

	go runPausedPollingLoop(
		true,
		lifecycleCh,
		tickerCh,
		stopCh,
		func() {
			ticks.Add(1)
		},
		func() {
			stopped <- struct{}{}
		},
	)

	tickerCh <- time.Now()
	time.Sleep(50 * time.Millisecond)
	if ticks.Load() != 0 {
		t.Fatalf("expected no ticks while paused, got %d", ticks.Load())
	}

	lifecycleCh <- false
	require.Eventually(t, func() bool {
		select {
		case tickerCh <- time.Now():
		default:
		}
		return ticks.Load() >= 1
	}, time.Second, 20*time.Millisecond)

	stopCh <- nil
	require.Eventually(t, func() bool {
		return len(stopped) == 1
	}, time.Second, 20*time.Millisecond)
}

func TestRunPausedPollingLoopStopsWhenLifecycleChannelCloses(t *testing.T) {
	lifecycleCh := make(chan bool)
	tickerCh := make(chan time.Time, 1)
	stopCh := make(chan error, 1)

	stopped := make(chan struct{}, 1)
	go runPausedPollingLoop(
		false,
		lifecycleCh,
		tickerCh,
		stopCh,
		func() {},
		func() { stopped <- struct{}{} },
	)

	close(lifecycleCh)
	require.Eventually(t, func() bool {
		return len(stopped) == 1
	}, time.Second, 20*time.Millisecond)
}

func TestRunPausedPollingLoopStopsOnStopSignal(t *testing.T) {
	lifecycleCh := make(chan bool, 1)
	tickerCh := make(chan time.Time, 1)
	stopCh := make(chan error, 1)

	var ticks atomic.Int32
	stopped := make(chan struct{}, 1)
	go runPausedPollingLoop(
		false,
		lifecycleCh,
		tickerCh,
		stopCh,
		func() { ticks.Add(1) },
		func() { stopped <- struct{}{} },
	)

	tickerCh <- time.Now()
	require.Eventually(t, func() bool {
		return ticks.Load() == 1
	}, time.Second, 20*time.Millisecond)

	stopCh <- nil
	require.Eventually(t, func() bool {
		return len(stopped) == 1
	}, time.Second, 20*time.Millisecond)
}

func TestMultipleTopicCopyInNewMessageFilter(t *testing.T) {
	w, err := New(nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Error creating WakuV2 client: %v", err)
	}

	keyID, err := w.GenerateSymKey()
	if err != nil {
		t.Fatalf("Error generating symmetric key: %v", err)
	}
	api := PublicWakuAPI{
		w:        w,
		lastUsed: make(map[string]time.Time),
	}

	t1 := common.TopicType([4]byte{0xde, 0xea, 0xbe, 0xef})
	t2 := common.TopicType([4]byte{0xca, 0xfe, 0xde, 0xca})

	crit := types2.Criteria{
		SymKeyID: keyID,
		Topics:   []types2.TopicType{types2.TopicType(t1), types2.TopicType(t2)},
	}

	_, err = api.NewMessageFilter(crit)
	if err != nil {
		t.Fatalf("Error creating the filter: %v", err)
	}

	found := false
	candidates := w.filters.GetWatchersByTopic(DefaultShardPubsubTopic(), t1)
	for _, f := range candidates {
		if maps.Equal(f.ContentTopics, common.NewTopicSet([]common.TopicType{t1, t2})) {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Could not find filter with both topics")
	}
}
