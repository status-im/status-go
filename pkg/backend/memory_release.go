package backend

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

// memoryReleaser returns idle heap to the OS at the two moments it matters on a phone: once the
// login burst (key derivation, bootstrap decode, first token build) has settled, and when the
// UI goes to background and the OS starts weighing the process. The background scavenger does the
// same work on its own, but over minutes; each call here costs one GC cycle. On Android it also
// purges the native allocator, which keeps freed pages resident while the process stays idle.
type memoryReleaser struct {
	mu              sync.Mutex
	timer           *time.Timer
	gen             uint64
	last            time.Time
	minGap          time.Duration
	release         func()
	freeOSMemory    func()
	purgeNativeHeap func()
	now             func() time.Time
	afterFn         func(time.Duration, func()) *time.Timer
	spawn           func(func())
	watcher         *walletevent.Watcher
	trailing        *time.Timer
	trailingGen     uint64
}

const (
	releaseAfterLoginDelay = 45 * time.Second
	releaseMinGap          = 30 * time.Second
	// Token-list consumers rebuild on the update; releasing mid-rebuild would only raise the heap goal.
	tokenListsReleaseDelay = 5 * time.Second
)

func newMemoryReleaser() *memoryReleaser {
	r := &memoryReleaser{
		minGap:          releaseMinGap,
		freeOSMemory:    debug.FreeOSMemory,
		purgeNativeHeap: purgeNativeHeap,
		now:             time.Now,
		afterFn:         time.AfterFunc,
		spawn: func(f func()) {
			go func() {
				defer panics.LogOnPanic()
				f()
			}()
		},
	}
	r.release = r.releaseHeaps
	return r
}

// releaseHeaps runs the GC first: finalizers it runs free native memory the purge then returns.
func (r *memoryReleaser) releaseHeaps() {
	r.freeOSMemory()
	r.purgeNativeHeap()
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
	r.recordReleaseLocked()
	return true
}

// recordReleaseLocked marks a release as run and drops a pending trailing release it covers, so
// a token-list update that lands after it can arm a fresh one.
func (r *memoryReleaser) recordReleaseLocked() {
	r.last = r.now()
	r.stopTrailingLocked()
}

func (r *memoryReleaser) stopTrailingLocked() {
	if r.trailing != nil {
		r.trailing.Stop()
		r.trailing = nil
	}
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
	r.recordReleaseLocked()
	r.mu.Unlock()
	r.spawn(r.release)
}

// watchTokenLists releases after each token-list refresh that changed the catalogue: the refresh
// can land after the post-login release, and its parse leaves megabytes of freed native heap.
// It replaces any previous watch.
func (r *memoryReleaser) watchTokenLists(feed *event.Feed) {
	w := walletevent.NewWatcher(feed, r.onWalletEvent)
	r.mu.Lock()
	old := r.watcher
	r.watcher = w
	w.Start()
	r.mu.Unlock()
	if old != nil {
		old.Stop()
	}
}

func (r *memoryReleaser) onWalletEvent(ev walletevent.Event) {
	if ev.Type == walletevent.EventTokenListsUpdated {
		r.scheduleTrailing()
	}
}

// scheduleTrailing arms one release after the update settles, or once the throttle gap ends, so
// an update right after another release is not dropped. A pending post-login or trailing release
// already covers the update.
func (r *memoryReleaser) scheduleTrailing() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil || r.trailing != nil {
		return
	}
	now := r.now()
	at := now.Add(tokenListsReleaseDelay)
	if gapEnd := r.last.Add(r.minGap); gapEnd.After(at) {
		at = gapEnd
	}
	r.trailingGen++
	gen := r.trailingGen
	r.trailing = r.afterFn(at.Sub(now), func() {
		defer panics.LogOnPanic()
		if r.claimTrailing(gen) {
			r.release()
		}
	})
}

// claimTrailing skips the release when another one ran after the update was seen, or when a
// post-login release is pending: releasing inside the login burst would only raise the heap goal.
func (r *memoryReleaser) claimTrailing(gen uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.trailing == nil || r.trailingGen != gen {
		return false
	}
	r.trailing = nil
	if r.timer != nil || r.now().Sub(r.last) < r.minGap {
		return false
	}
	r.last = r.now()
	return true
}

// stop cancels the token-list watch, then pending releases. The watch is stopped first and
// outside the lock: Stop waits for a queued callback, which takes the lock and could arm a release.
func (r *memoryReleaser) stop() {
	r.mu.Lock()
	w := r.watcher
	r.watcher = nil
	r.mu.Unlock()
	if w != nil {
		w.Stop()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.stopTrailingLocked()
}
