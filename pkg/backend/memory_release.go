package backend

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/status-im/status-go/internal/panics"
)

// memoryReleaser returns idle heap to the OS at the two moments it matters on a phone: once the
// login burst (key derivation, bootstrap decode, first token build) has settled, and when the
// UI goes to background and the OS starts weighing the process. The background scavenger does the
// same work on its own, but over minutes; each call here costs one GC cycle.
type memoryReleaser struct {
	mu      sync.Mutex
	timer   *time.Timer
	gen     uint64
	last    time.Time
	minGap  time.Duration
	release func()
	now     func() time.Time
	afterFn func(time.Duration, func()) *time.Timer
	spawn   func(func())
}

const (
	releaseAfterLoginDelay = 45 * time.Second
	releaseMinGap          = 30 * time.Second
)

func newMemoryReleaser() *memoryReleaser {
	return &memoryReleaser{
		minGap:  releaseMinGap,
		release: debug.FreeOSMemory,
		now:     time.Now,
		afterFn: time.AfterFunc,
		spawn: func(f func()) {
			go func() {
				defer panics.LogOnPanic()
				f()
			}()
		},
	}
}

// scheduleAfterLogin arms a one-shot release, replacing any pending one. It is never throttled:
// it is the release that lands after the burst, and a pause release that happened to run during
// the burst must not cancel it.
func (r *memoryReleaser) scheduleAfterLogin(delay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil {
		r.timer.Stop()
	}
	r.gen++
	gen := r.gen
	r.timer = r.afterFn(delay, func() {
		defer panics.LogOnPanic()
		if r.claimFired(gen) {
			r.release()
		}
	})
}

// claimFired clears the fired timer and records the release under one lock, so a concurrent
// releaseNow sees either the pending timer or the fresh release time. The generation is captured
// by value when the timer is armed, so a callback that fires after its timer was replaced or
// stopped claims nothing.
func (r *memoryReleaser) claimFired(gen uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer == nil || r.gen != gen {
		return false
	}
	r.timer = nil
	r.last = r.now()
	return true
}

// releaseNow releases unless the post-login release is still pending or a release ran within
// minGap. Releasing while the burst is still live would only raise the heap goal for the cycles
// that follow. The release itself runs off the caller's goroutine: on iOS PauseServices is a
// synchronous call on the GUI thread inside the backgrounding window.
func (r *memoryReleaser) releaseNow() {
	r.mu.Lock()
	if r.timer != nil || r.now().Sub(r.last) < r.minGap {
		r.mu.Unlock()
		return
	}
	r.last = r.now()
	r.mu.Unlock()
	r.spawn(r.release)
}

// stop cancels a pending post-login release.
func (r *memoryReleaser) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
}
