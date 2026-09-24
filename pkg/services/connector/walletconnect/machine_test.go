package walletconnect

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testInitialBackoff = time.Second
	testMaxBackoff     = time.Minute
)

var errTestDial = errors.New("dial failed")

// Machines in every state of RELAY_SPEC.md.

func idle() machine { return newMachine(testInitialBackoff, testMaxBackoff) }

func dialingFirst() machine {
	m := idle()
	m.state, m.attempt, m.first, m.connectWaiting = stateDialing, 1, true, true
	return m
}

func dialingRedial(attempt int, delay time.Duration, parked ...callID) machine {
	m := idle()
	m.state, m.attempt, m.delay, m.parked, m.everConnected = stateDialing, attempt, delay, parked, true
	return m
}

func connected(c connID) machine {
	m := idle()
	m.state, m.conn, m.everConnected = stateConnected, c, true
	return m
}

func backoff(attempt int, delay time.Duration, parked ...callID) machine {
	m := idle()
	m.state, m.attempt, m.delay, m.parked, m.everConnected = stateBackoff, attempt, delay, parked, true
	return m
}

func closed(everConnected bool) machine {
	m := idle()
	m.state, m.everConnected = stateClosed, everConnected
	return m
}

func connectWaiting(m machine) machine {
	m.connectWaiting = true
	return m
}

// Events.

func evConnect() relayEvent              { return relayEvent{kind: eventConnect} }
func evDialOK(c connID) relayEvent       { return relayEvent{kind: eventDialOK, conn: c} }
func evDialFailed(e error) relayEvent    { return relayEvent{kind: eventDialFailed, err: e} }
func evConnLost(c connID) relayEvent     { return relayEvent{kind: eventConnLost, conn: c} }
func evBackoffElapsed() relayEvent       { return relayEvent{kind: eventBackoffElapsed} }
func evClose() relayEvent                { return relayEvent{kind: eventClose} }
func evCall(id callID) relayEvent        { return relayEvent{kind: eventCall, call: id} }
func evWaitExpired(id callID) relayEvent { return relayEvent{kind: eventWaitExpired, call: id} }

// Commands.

func startDial() relayCommand                   { return relayCommand{kind: cmdStartDial} }
func abortDial() relayCommand                   { return relayCommand{kind: cmdAbortDial} }
func closeConn(c connID) relayCommand           { return relayCommand{kind: cmdCloseConn, conn: c} }
func startBackoff(d time.Duration) relayCommand { return relayCommand{kind: cmdStartBackoff, delay: d} }
func stopBackoff() relayCommand                 { return relayCommand{kind: cmdStopBackoff} }
func startHeartbeat(c connID) relayCommand      { return relayCommand{kind: cmdStartHeartbeat, conn: c} }
func startReader(c connID) relayCommand         { return relayCommand{kind: cmdStartReader, conn: c} }
func notifyReconnected() relayCommand           { return relayCommand{kind: cmdNotifyReconnected} }
func replyConnect(e error) relayCommand         { return relayCommand{kind: cmdReplyConnect, err: e} }
func serve(id callID, c connID) relayCommand    { return relayCommand{kind: cmdServe, call: id, conn: c} }
func park(id callID) relayCommand               { return relayCommand{kind: cmdPark, call: id} }
func fail(id callID, e error) relayCommand      { return relayCommand{kind: cmdFail, call: id, err: e} }

type transition struct {
	rule    string // T-<State>-<Event>
	variant string
	from    machine
	event   relayEvent
	to      machine
	cmds    []relayCommand
}

// transitions has one row per non-blank cell of the transition tables.
var transitions = []transition{
	// Idle
	{rule: "T-Idle-Connect", from: idle(), event: evConnect(),
		to: dialingFirst(), cmds: []relayCommand{startDial()}},
	{rule: "T-Idle-Close", from: idle(), event: evClose(),
		to: closed(false)},
	{rule: "T-Idle-Call", from: idle(), event: evCall(7),
		to: idle(), cmds: []relayCommand{fail(7, errNotConnected)}},

	// Dialing (first)
	{rule: "T-DialingFirst-Connect", from: dialingFirst(), event: evConnect(),
		to: dialingFirst()},
	{rule: "T-DialingFirst-DialOK", from: dialingFirst(), event: evDialOK(1),
		to:   connected(1),
		cmds: []relayCommand{startHeartbeat(1), startReader(1), replyConnect(nil)}},
	{rule: "T-DialingFirst-DialFailed", from: dialingFirst(), event: evDialFailed(errTestDial),
		to: idle(), cmds: []relayCommand{replyConnect(errTestDial)}},
	{rule: "T-DialingFirst-Close", from: dialingFirst(), event: evClose(),
		to: closed(false), cmds: []relayCommand{abortDial(), replyConnect(ErrRelayClosed)}},
	{rule: "T-DialingFirst-Call", from: dialingFirst(), event: evCall(7),
		to: dialingFirst(), cmds: []relayCommand{fail(7, errNotConnected)}},

	// Dialing (redial, attempt n)
	{rule: "T-DialingRedial-Connect", from: dialingRedial(3, 4*time.Second, 7), event: evConnect(),
		to: connectWaiting(dialingRedial(3, 4*time.Second, 7))},
	{rule: "T-DialingRedial-DialOK", from: dialingRedial(3, 4*time.Second, 7, 8), event: evDialOK(2),
		to:   connected(2),
		cmds: []relayCommand{startHeartbeat(2), startReader(2), serve(7, 2), serve(8, 2), notifyReconnected()}},
	{rule: "T-DialingRedial-DialOK", variant: "Connect waiting",
		from: connectWaiting(dialingRedial(3, 4*time.Second, 7)), event: evDialOK(2),
		to:   connected(2),
		cmds: []relayCommand{startHeartbeat(2), startReader(2), serve(7, 2), notifyReconnected(), replyConnect(nil)}},
	{rule: "T-DialingRedial-DialFailed", from: dialingRedial(3, 4*time.Second, 7), event: evDialFailed(errTestDial),
		to: backoff(3, 8*time.Second, 7), cmds: []relayCommand{startBackoff(8 * time.Second)}},
	{rule: "T-DialingRedial-DialFailed", variant: "Connect waiting",
		from: connectWaiting(dialingRedial(3, 4*time.Second)), event: evDialFailed(errTestDial),
		to: backoff(3, 8*time.Second), cmds: []relayCommand{startBackoff(8 * time.Second), replyConnect(errTestDial)}},
	{rule: "T-DialingRedial-DialFailed", variant: "capped at the maximum",
		from: dialingRedial(7, 40*time.Second), event: evDialFailed(errTestDial),
		to: backoff(7, testMaxBackoff), cmds: []relayCommand{startBackoff(testMaxBackoff)}},
	{rule: "T-DialingRedial-Close", from: dialingRedial(3, 4*time.Second, 7, 8), event: evClose(),
		to:   closed(true),
		cmds: []relayCommand{abortDial(), fail(7, errShuttingDown), fail(8, errShuttingDown)}},
	{rule: "T-DialingRedial-Close", variant: "Connect waiting",
		from: connectWaiting(dialingRedial(3, 4*time.Second, 7)), event: evClose(),
		to:   closed(true),
		cmds: []relayCommand{abortDial(), fail(7, errShuttingDown), replyConnect(ErrRelayClosed)}},
	{rule: "T-DialingRedial-Call", from: dialingRedial(3, 4*time.Second, 7), event: evCall(8),
		to: dialingRedial(3, 4*time.Second, 7, 8), cmds: []relayCommand{park(8)}},
	{rule: "T-DialingRedial-WaitExpired", from: dialingRedial(3, 4*time.Second, 7, 8), event: evWaitExpired(7),
		to: dialingRedial(3, 4*time.Second, 8), cmds: []relayCommand{fail(7, errReconnectInProgress)}},

	// Connected{c}
	{rule: "T-Connected-Connect", from: connected(1), event: evConnect(),
		to: connected(1), cmds: []relayCommand{replyConnect(nil)}},
	{rule: "T-Connected-ConnLost", from: connected(1), event: evConnLost(1),
		to:   backoff(1, testInitialBackoff),
		cmds: []relayCommand{closeConn(1), startBackoff(testInitialBackoff)}},
	{rule: "T-Connected-Close", from: connected(1), event: evClose(),
		to: closed(true), cmds: []relayCommand{closeConn(1)}},
	{rule: "T-Connected-Call", from: connected(1), event: evCall(7),
		to: connected(1), cmds: []relayCommand{serve(7, 1)}},

	// Backoff{n, delay}
	{rule: "T-Backoff-Connect", from: backoff(2, 2*time.Second, 7), event: evConnect(),
		to:   connectWaiting(dialingRedial(3, 2*time.Second, 7)),
		cmds: []relayCommand{stopBackoff(), startDial()}},
	{rule: "T-Backoff-BackoffElapsed", from: backoff(2, 2*time.Second, 7), event: evBackoffElapsed(),
		to: dialingRedial(3, 2*time.Second, 7), cmds: []relayCommand{startDial()}},
	{rule: "T-Backoff-Close", from: backoff(2, 2*time.Second, 7, 8), event: evClose(),
		to:   closed(true),
		cmds: []relayCommand{stopBackoff(), fail(7, errShuttingDown), fail(8, errShuttingDown)}},
	{rule: "T-Backoff-Call", from: backoff(2, 2*time.Second, 7), event: evCall(8),
		to: backoff(2, 2*time.Second, 7, 8), cmds: []relayCommand{park(8)}},
	{rule: "T-Backoff-WaitExpired", from: backoff(2, 2*time.Second, 7, 8), event: evWaitExpired(8),
		to: backoff(2, 2*time.Second, 7), cmds: []relayCommand{fail(8, errReconnectInProgress)}},

	// Closed
	{rule: "T-Closed-Connect", from: closed(true), event: evConnect(),
		to: closed(true), cmds: []relayCommand{replyConnect(ErrRelayClosed)}},
	{rule: "T-Closed-DialOK", from: closed(true), event: evDialOK(2),
		to: closed(true), cmds: []relayCommand{closeConn(2)}},
	{rule: "T-Closed-Call", from: closed(true), event: evCall(7),
		to: closed(true), cmds: []relayCommand{fail(7, errShuttingDown)}},
}

type ignored struct {
	pair    string // <State>-<Event>
	variant string
	from    machine
	event   relayEvent
}

// ignoredEvents lists the blank cells: the event must leave the machine
// unchanged and emit no commands.
var ignoredEvents = []ignored{
	{pair: "Idle-DialOK", from: idle(), event: evDialOK(1)},
	{pair: "Idle-DialFailed", from: idle(), event: evDialFailed(errTestDial)},
	{pair: "Idle-ConnLost", from: idle(), event: evConnLost(1)},
	{pair: "Idle-BackoffElapsed", from: idle(), event: evBackoffElapsed()},
	{pair: "Idle-WaitExpired", from: idle(), event: evWaitExpired(7)},

	{pair: "DialingFirst-ConnLost", from: dialingFirst(), event: evConnLost(1)},
	{pair: "DialingFirst-BackoffElapsed", from: dialingFirst(), event: evBackoffElapsed()},
	{pair: "DialingFirst-WaitExpired", from: dialingFirst(), event: evWaitExpired(7)},

	{pair: "DialingRedial-ConnLost", from: dialingRedial(3, 4*time.Second, 7), event: evConnLost(1)},
	{pair: "DialingRedial-BackoffElapsed", from: dialingRedial(3, 4*time.Second, 7), event: evBackoffElapsed()},
	{pair: "DialingRedial-WaitExpired", variant: "not parked",
		from: dialingRedial(3, 4*time.Second, 7), event: evWaitExpired(9)},

	{pair: "Connected-DialOK", from: connected(1), event: evDialOK(2)},
	{pair: "Connected-DialFailed", from: connected(1), event: evDialFailed(errTestDial)},
	{pair: "Connected-ConnLost", variant: "other connection", from: connected(1), event: evConnLost(2)},
	{pair: "Connected-BackoffElapsed", from: connected(1), event: evBackoffElapsed()},
	{pair: "Connected-WaitExpired", from: connected(1), event: evWaitExpired(7)},

	{pair: "Backoff-DialOK", from: backoff(2, 2*time.Second, 7), event: evDialOK(2)},
	{pair: "Backoff-DialFailed", from: backoff(2, 2*time.Second, 7), event: evDialFailed(errTestDial)},
	{pair: "Backoff-ConnLost", from: backoff(2, 2*time.Second, 7), event: evConnLost(1)},
	{pair: "Backoff-WaitExpired", variant: "not parked",
		from: backoff(2, 2*time.Second, 7), event: evWaitExpired(9)},

	{pair: "Closed-DialFailed", from: closed(true), event: evDialFailed(errTestDial)},
	{pair: "Closed-ConnLost", from: closed(true), event: evConnLost(1)},
	{pair: "Closed-BackoffElapsed", from: closed(true), event: evBackoffElapsed()},
	{pair: "Closed-Close", from: closed(true), event: evClose()},
	{pair: "Closed-WaitExpired", from: closed(true), event: evWaitExpired(7)},
}

func testName(rule, variant string) string {
	name := strings.ReplaceAll(rule, "-", "_")
	if variant != "" {
		name += "/" + variant
	}
	return name
}

func TestMachine(t *testing.T) {
	for _, tr := range transitions {
		t.Run(testName(tr.rule, tr.variant), func(t *testing.T) {
			to, cmds := tr.from.Step(tr.event)
			require.Equal(t, tr.to, to)
			require.Equal(t, tr.cmds, cmds)
		})
	}
	for _, ig := range ignoredEvents {
		t.Run(testName("Ignored-"+ig.pair, ig.variant), func(t *testing.T) {
			to, cmds := ig.from.Step(ig.event)
			require.Equal(t, ig.from, to)
			require.Empty(t, cmds)
		})
	}
}

var (
	specStates = []string{"Idle", "DialingFirst", "DialingRedial", "Connected", "Backoff", "Closed"}
	specEvents = []string{"Connect", "DialOK", "DialFailed", "ConnLost", "BackoffElapsed", "Close", "Call", "WaitExpired"}
)

func TestMachine_Completeness(t *testing.T) {
	known := map[string]bool{}
	for _, s := range specStates {
		for _, e := range specEvents {
			known[s+"-"+e] = true
		}
	}
	covered := map[string]bool{}
	for _, tr := range transitions {
		pair := strings.TrimPrefix(tr.rule, "T-")
		require.True(t, known[pair], "row %q names no state × event pair", tr.rule)
		require.Equal(t, strings.SplitN(pair, "-", 2)[1], tr.event.kind.String(), "row %q", tr.rule)
		covered[pair] = true
	}
	for _, ig := range ignoredEvents {
		require.True(t, known[ig.pair], "ignored %q names no state × event pair", ig.pair)
		require.Equal(t, strings.SplitN(ig.pair, "-", 2)[1], ig.event.kind.String(), "ignored %q", ig.pair)
		covered[ig.pair] = true
	}
	for _, s := range specStates {
		for _, e := range specEvents {
			require.True(t, covered[s+"-"+e], "%s × %s has neither a transition row nor an ignored entry", s, e)
		}
	}
}

// world plays the shell around a machine: it only raises events the shell
// could raise, and records what the commands did to check I1–I6.
type world struct {
	m        machine
	rnd      *rand.Rand
	log      []string
	nextConn connID
	nextCall callID

	dials          int // dials started and not yet reported
	backoffRunning bool
	live           map[connID]bool // handed over by DialOK and not closed
	conns          []connID
	calls          []callID
	outcomes       map[callID]int // Serve + Fail per call
	parked         map[callID]bool
	connectWaiters int
	outageDelay    time.Duration // last StartBackoff since the last DialOK; 0 at the start of an outage
}

func newWorld(seed int64) *world {
	return &world{
		m:        idle(),
		rnd:      rand.New(rand.NewSource(seed)), // #nosec G404 -- reproducible test sequences
		live:     map[connID]bool{},
		outcomes: map[callID]int{},
		parked:   map[callID]bool{},
	}
}

func (w *world) randomEvent() relayEvent {
	var candidates []relayEvent
	add := func(weight int, e relayEvent) {
		for range weight {
			candidates = append(candidates, e)
		}
	}
	add(3, evConnect())
	add(1, evClose())
	add(6, evCall(w.nextCall+1))
	if w.dials > 0 {
		add(4, evDialOK(w.nextConn+1))
		add(4, evDialFailed(errTestDial))
	}
	if len(w.conns) > 0 {
		add(3, evConnLost(w.conns[len(w.conns)-1]))
		add(1, evConnLost(w.conns[w.rnd.Intn(len(w.conns))]))
	}
	if w.backoffRunning {
		add(4, evBackoffElapsed())
	}
	if len(w.calls) > 0 {
		add(3, evWaitExpired(w.calls[w.rnd.Intn(len(w.calls))]))
	}
	return candidates[w.rnd.Intn(len(candidates))]
}

// apply steps the machine and returns the first invariant it violates.
func (w *world) apply(e relayEvent) error {
	from := w.m
	switch e.kind {
	case eventConnect:
		w.connectWaiters++
	case eventDialOK:
		w.dials--
		w.nextConn = e.conn
		w.conns = append(w.conns, e.conn)
		w.live[e.conn] = true
	case eventDialFailed:
		w.dials--
	case eventBackoffElapsed:
		w.backoffRunning = false
	case eventCall:
		w.nextCall = e.call
		w.calls = append(w.calls, e.call)
	}

	to, cmds := from.Step(e)
	w.m = to
	w.log = append(w.log, fmt.Sprintf("%-40s %v -> %v %v", e, describe(from), describe(to), cmds))

	resolved := map[callID]bool{}
	replies, notifies := 0, 0
	for _, c := range cmds {
		switch c.kind {
		case cmdStartDial:
			w.dials++
			if to.state == stateClosed {
				return errors.New("I3: StartDial in Closed")
			}
		case cmdCloseConn:
			if !w.live[c.conn] {
				return fmt.Errorf("CloseConn(%d) of a connection that is not live", c.conn)
			}
			delete(w.live, c.conn)
		case cmdStartBackoff:
			if w.backoffRunning {
				return errors.New("I1: StartBackoff while a backoff runs")
			}
			w.backoffRunning = true
			if c.delay > w.m.maxBackoff {
				return fmt.Errorf("I4: backoff %v exceeds the maximum %v", c.delay, w.m.maxBackoff)
			}
			if w.outageDelay == 0 && c.delay != w.m.initialBackoff {
				return fmt.Errorf("I4: first backoff of an outage is %v, want %v", c.delay, w.m.initialBackoff)
			}
			if c.delay < w.outageDelay {
				return fmt.Errorf("I4: backoff decreased from %v to %v", w.outageDelay, c.delay)
			}
			w.outageDelay = c.delay
		case cmdStopBackoff:
			w.backoffRunning = false
		case cmdStartHeartbeat, cmdStartReader:
			if to.state == stateClosed {
				return fmt.Errorf("I3: %v in Closed", c.kind)
			}
			if !w.live[c.conn] {
				return fmt.Errorf("%v of a connection that is not live", c)
			}
		case cmdNotifyReconnected:
			notifies++
			if e.kind != eventDialOK || from.state != stateDialing || from.first {
				return errors.New("I5: NotifyReconnected outside the DialOK of a redial")
			}
		case cmdReplyConnect:
			replies++
			if w.connectWaiters == 0 {
				return errors.New("I6: ReplyConnect with no Connect waiting")
			}
			w.connectWaiters = 0
		case cmdServe, cmdFail:
			w.outcomes[c.call]++
			if w.outcomes[c.call] > 1 {
				return fmt.Errorf("I2: call %d answered twice", c.call)
			}
			if c.kind == cmdServe && (!w.live[c.conn] || to.state != stateConnected || to.conn != c.conn) {
				return fmt.Errorf("I2: call %d served on connection %d, which is not the live one", c.call, c.conn)
			}
			delete(w.parked, c.call)
			resolved[c.call] = true
		case cmdPark:
			w.parked[c.call] = true
			resolved[c.call] = true
		}
	}

	if e.kind == eventDialOK {
		w.outageDelay = 0
	}
	if len(w.live) > 1 || w.dials > 1 {
		return fmt.Errorf("I1: %d live connections, %d dials in flight", len(w.live), w.dials)
	}
	if e.kind == eventCall && !resolved[e.call] {
		return fmt.Errorf("I2: call %d was neither served, failed nor parked", e.call)
	}
	if len(w.parked) != len(to.parked) {
		return fmt.Errorf("I2: %d calls parked, the machine holds %v", len(w.parked), to.parked)
	}
	for _, id := range to.parked {
		if !w.parked[id] {
			return fmt.Errorf("I2: the machine holds call %d, which is not parked", id)
		}
	}
	if len(to.parked) > 0 && to.state != stateDialing && to.state != stateBackoff {
		return fmt.Errorf("I2: calls %v left parked in %v", to.parked, to.state)
	}
	if to.state == stateClosed && len(w.live) > 0 {
		return errors.New("I3: a connection is still live in Closed")
	}
	if notifies > 1 {
		return errors.New("I5: NotifyReconnected emitted twice for one redial")
	}
	if e.kind == eventDialOK && from.state == stateDialing && !from.first && notifies != 1 {
		return errors.New("I5: a redial succeeded without NotifyReconnected")
	}
	if replies > 1 {
		return errors.New("I6: two ReplyConnect in one step")
	}
	return nil
}

// finish closes the machine, lets the in-flight dial report, and checks that
// nothing is left unanswered.
func (w *world) finish() error {
	if err := w.apply(evClose()); err != nil {
		return err
	}
	for w.dials > 0 {
		e := evDialFailed(errTestDial)
		if w.rnd.Intn(2) == 0 {
			e = evDialOK(w.nextConn + 1)
		}
		if err := w.apply(e); err != nil {
			return err
		}
	}
	for _, id := range w.calls {
		if w.outcomes[id] != 1 {
			return fmt.Errorf("I2: call %d answered %d times", id, w.outcomes[id])
		}
	}
	if w.connectWaiters != 0 {
		return fmt.Errorf("I6: %d Connect never got a reply", w.connectWaiters)
	}
	if len(w.live) > 0 {
		return errors.New("I3: a connection was never closed")
	}
	return nil
}

func describe(m machine) string {
	s := m.state.String()
	switch m.state {
	case stateDialing:
		s += fmt.Sprintf("{attempt:%d first:%t delay:%v}", m.attempt, m.first, m.delay)
	case stateBackoff:
		s += fmt.Sprintf("{attempt:%d delay:%v}", m.attempt, m.delay)
	case stateConnected:
		s += fmt.Sprintf("{conn:%d}", m.conn)
	}
	if len(m.parked) > 0 {
		s += fmt.Sprintf(" parked:%v", m.parked)
	}
	if m.connectWaiting {
		s += " connect-waiting"
	}
	return s
}

func TestMachine_Invariants(t *testing.T) {
	const (
		seeds = 500
		steps = 200
	)
	for seed := int64(1); seed <= seeds; seed++ {
		w := newWorld(seed)
		err := func() error {
			for range steps {
				if err := w.apply(w.randomEvent()); err != nil {
					return err
				}
				if w.m.state == stateClosed && w.dials == 0 {
					break
				}
			}
			return w.finish()
		}()
		if err != nil {
			t.Fatalf("seed %d: %v\nsequence:\n%s", seed, err, strings.Join(w.log, "\n"))
		}
	}
}

func TestMachine_InvariantsReachEveryState(t *testing.T) {
	reached := map[relayState]bool{}
	for seed := int64(1); seed <= 50; seed++ {
		w := newWorld(seed)
		for range 200 {
			if w.apply(w.randomEvent()) != nil {
				break
			}
			reached[w.m.state] = true
		}
	}
	for _, s := range []relayState{stateIdle, stateDialing, stateConnected, stateBackoff, stateClosed} {
		require.True(t, reached[s], "random sequences never reach %v", s)
	}
}
