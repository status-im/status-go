package walletconnect

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// machine is the relay connection state machine of RELAY_SPEC.md. It is pure:
// Step takes one event and returns the next machine and the commands the
// shell must execute, in order.
type machine struct {
	state relayState
	// attempt, first and delay describe stateDialing and stateBackoff.
	attempt int
	first   bool
	delay   time.Duration
	// conn is the live connection in stateConnected.
	conn connID

	everConnected  bool
	connectWaiting bool
	parked         []callID

	initialBackoff time.Duration
	maxBackoff     time.Duration
}

type relayState int

const (
	stateIdle relayState = iota
	stateDialing
	stateConnected
	stateBackoff
	stateClosed
)

// connID names a connection handed to the machine by DialOK.
type connID uint64

// callID names one relay call.
type callID uint64

type eventKind int

const (
	eventConnect eventKind = iota
	eventDialOK
	eventDialFailed
	eventConnLost
	eventBackoffElapsed
	eventClose
	eventCall
	eventWaitExpired
)

type relayEvent struct {
	kind eventKind
	conn connID // eventDialOK, eventConnLost
	err  error  // eventDialFailed
	call callID // eventCall, eventWaitExpired
}

type commandKind int

const (
	cmdStartDial commandKind = iota
	cmdAbortDial
	cmdCloseConn
	cmdStartBackoff
	cmdStopBackoff
	cmdStartHeartbeat
	cmdStartReader
	cmdNotifyReconnected
	cmdReplyConnect
	cmdServe
	cmdPark
	cmdFail
)

type relayCommand struct {
	kind  commandKind
	conn  connID        // cmdCloseConn, cmdStartHeartbeat, cmdStartReader, cmdServe
	delay time.Duration // cmdStartBackoff
	err   error         // cmdReplyConnect, cmdFail
	call  callID        // cmdServe, cmdPark, cmdFail
}

var (
	errNotConnected        = errors.New("not connected")
	errReconnectInProgress = errors.New("relay unavailable: reconnect in progress")
	errShuttingDown        = errors.New("relay client shutting down")
)

func newMachine(initialBackoff, maxBackoff time.Duration) machine {
	return machine{initialBackoff: initialBackoff, maxBackoff: maxBackoff}
}

// Step applies one event. Events a state does not handle leave the machine
// unchanged and return no commands.
func (m machine) Step(e relayEvent) (machine, []relayCommand) {
	switch m.state {
	case stateIdle:
		return m.stepIdle(e)
	case stateDialing:
		if m.first {
			return m.stepDialingFirst(e)
		}
		return m.stepDialingRedial(e)
	case stateConnected:
		return m.stepConnected(e)
	case stateBackoff:
		return m.stepBackoff(e)
	case stateClosed:
		return m.stepClosed(e)
	}
	return m, nil
}

func (m machine) stepIdle(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		return m.dialing(1, true, 0, true), []relayCommand{{kind: cmdStartDial}}
	case eventClose:
		return m.closed(), nil
	case eventCall:
		return m, []relayCommand{failCmd(e.call, errNotConnected)}
	}
	return m, nil
}

func (m machine) stepDialingFirst(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		return m, nil
	case eventDialOK:
		next := m.connected(e.conn)
		next.everConnected = true
		return next, append(connStartCmds(e.conn), replyCmd(nil))
	case eventDialFailed:
		return m.idle(), []relayCommand{replyCmd(e.err)}
	case eventClose:
		return m.closed(), []relayCommand{{kind: cmdAbortDial}, replyCmd(ErrRelayClosed)}
	case eventCall:
		return m, []relayCommand{failCmd(e.call, errNotConnected)}
	}
	return m, nil
}

func (m machine) stepDialingRedial(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		m.connectWaiting = true
		return m, nil
	case eventDialOK:
		cmds := connStartCmds(e.conn)
		for _, id := range m.parked {
			cmds = append(cmds, relayCommand{kind: cmdServe, call: id, conn: e.conn})
		}
		cmds = append(cmds, relayCommand{kind: cmdNotifyReconnected})
		if m.connectWaiting {
			cmds = append(cmds, replyCmd(nil))
		}
		return m.connected(e.conn), cmds
	case eventDialFailed:
		delay := min(2*m.delay, m.maxBackoff)
		cmds := []relayCommand{{kind: cmdStartBackoff, delay: delay}}
		if m.connectWaiting {
			cmds = append(cmds, replyCmd(e.err))
		}
		return m.backoff(m.attempt, delay), cmds
	case eventClose:
		cmds := append([]relayCommand{{kind: cmdAbortDial}}, m.failParked(errShuttingDown)...)
		if m.connectWaiting {
			cmds = append(cmds, replyCmd(ErrRelayClosed))
		}
		return m.closed(), cmds
	case eventCall:
		return m.park(e.call)
	case eventWaitExpired:
		return m.expire(e.call)
	}
	return m, nil
}

func (m machine) stepConnected(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		return m, []relayCommand{replyCmd(nil)}
	case eventConnLost:
		if e.conn != m.conn {
			return m, nil
		}
		return m.backoff(1, m.initialBackoff), []relayCommand{
			{kind: cmdCloseConn, conn: m.conn},
			{kind: cmdStartBackoff, delay: m.initialBackoff},
		}
	case eventClose:
		return m.closed(), []relayCommand{{kind: cmdCloseConn, conn: m.conn}}
	case eventCall:
		return m, []relayCommand{{kind: cmdServe, call: e.call, conn: m.conn}}
	}
	return m, nil
}

func (m machine) stepBackoff(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		return m.dialing(m.attempt+1, false, m.delay, true),
			[]relayCommand{{kind: cmdStopBackoff}, {kind: cmdStartDial}}
	case eventBackoffElapsed:
		return m.dialing(m.attempt+1, false, m.delay, false), []relayCommand{{kind: cmdStartDial}}
	case eventClose:
		return m.closed(), append([]relayCommand{{kind: cmdStopBackoff}}, m.failParked(errShuttingDown)...)
	case eventCall:
		return m.park(e.call)
	case eventWaitExpired:
		return m.expire(e.call)
	}
	return m, nil
}

func (m machine) stepClosed(e relayEvent) (machine, []relayCommand) {
	switch e.kind {
	case eventConnect:
		return m, []relayCommand{replyCmd(ErrRelayClosed)}
	case eventDialOK:
		return m, []relayCommand{{kind: cmdCloseConn, conn: e.conn}}
	case eventCall:
		return m, []relayCommand{failCmd(e.call, errShuttingDown)}
	}
	return m, nil
}

// The constructors below keep the parked calls and everConnected, and reset
// every field that does not belong to the new state.

func (m machine) reset(s relayState) machine {
	return machine{
		state:          s,
		everConnected:  m.everConnected,
		parked:         m.parked,
		initialBackoff: m.initialBackoff,
		maxBackoff:     m.maxBackoff,
	}
}

func (m machine) idle() machine { return m.reset(stateIdle) }

func (m machine) dialing(attempt int, first bool, delay time.Duration, connectWaiting bool) machine {
	next := m.reset(stateDialing)
	next.attempt, next.first, next.delay, next.connectWaiting = attempt, first, delay, connectWaiting
	return next
}

func (m machine) connected(c connID) machine {
	next := m.reset(stateConnected)
	next.conn, next.parked = c, nil
	return next
}

func (m machine) backoff(attempt int, delay time.Duration) machine {
	next := m.reset(stateBackoff)
	next.attempt, next.delay = attempt, delay
	return next
}

func (m machine) closed() machine {
	next := m.reset(stateClosed)
	next.parked = nil
	return next
}

func (m machine) park(id callID) (machine, []relayCommand) {
	m.parked = append(slices.Clone(m.parked), id)
	return m, []relayCommand{{kind: cmdPark, call: id}}
}

func (m machine) expire(id callID) (machine, []relayCommand) {
	i := slices.Index(m.parked, id)
	if i < 0 {
		return m, nil
	}
	m.parked = slices.Delete(slices.Clone(m.parked), i, i+1)
	if len(m.parked) == 0 {
		m.parked = nil
	}
	return m, []relayCommand{failCmd(id, errReconnectInProgress)}
}

func (m machine) failParked(err error) []relayCommand {
	cmds := make([]relayCommand, 0, len(m.parked))
	for _, id := range m.parked {
		cmds = append(cmds, failCmd(id, err))
	}
	return cmds
}

func connStartCmds(c connID) []relayCommand {
	return []relayCommand{{kind: cmdStartHeartbeat, conn: c}, {kind: cmdStartReader, conn: c}}
}

func replyCmd(err error) relayCommand { return relayCommand{kind: cmdReplyConnect, err: err} }

func failCmd(id callID, err error) relayCommand {
	return relayCommand{kind: cmdFail, call: id, err: err}
}

func (s relayState) String() string {
	switch s {
	case stateIdle:
		return "Idle"
	case stateDialing:
		return "Dialing"
	case stateConnected:
		return "Connected"
	case stateBackoff:
		return "Backoff"
	case stateClosed:
		return "Closed"
	}
	return fmt.Sprintf("relayState(%d)", int(s))
}

func (k eventKind) String() string {
	switch k {
	case eventConnect:
		return "Connect"
	case eventDialOK:
		return "DialOK"
	case eventDialFailed:
		return "DialFailed"
	case eventConnLost:
		return "ConnLost"
	case eventBackoffElapsed:
		return "BackoffElapsed"
	case eventClose:
		return "Close"
	case eventCall:
		return "Call"
	case eventWaitExpired:
		return "WaitExpired"
	}
	return fmt.Sprintf("eventKind(%d)", int(k))
}

func (k commandKind) String() string {
	switch k {
	case cmdStartDial:
		return "StartDial"
	case cmdAbortDial:
		return "AbortDial"
	case cmdCloseConn:
		return "CloseConn"
	case cmdStartBackoff:
		return "StartBackoff"
	case cmdStopBackoff:
		return "StopBackoff"
	case cmdStartHeartbeat:
		return "StartHeartbeat"
	case cmdStartReader:
		return "StartReader"
	case cmdNotifyReconnected:
		return "NotifyReconnected"
	case cmdReplyConnect:
		return "ReplyConnect"
	case cmdServe:
		return "Serve"
	case cmdPark:
		return "Park"
	case cmdFail:
		return "Fail"
	}
	return fmt.Sprintf("commandKind(%d)", int(k))
}

func (e relayEvent) String() string {
	switch e.kind {
	case eventDialOK, eventConnLost:
		return fmt.Sprintf("%v(conn %d)", e.kind, e.conn)
	case eventDialFailed:
		return fmt.Sprintf("%v(%v)", e.kind, e.err)
	case eventCall, eventWaitExpired:
		return fmt.Sprintf("%v(call %d)", e.kind, e.call)
	}
	return e.kind.String()
}

func (c relayCommand) String() string {
	switch c.kind {
	case cmdCloseConn, cmdStartHeartbeat, cmdStartReader:
		return fmt.Sprintf("%v(conn %d)", c.kind, c.conn)
	case cmdStartBackoff:
		return fmt.Sprintf("%v(%v)", c.kind, c.delay)
	case cmdReplyConnect:
		return fmt.Sprintf("%v(%v)", c.kind, c.err)
	case cmdServe:
		return fmt.Sprintf("%v(call %d, conn %d)", c.kind, c.call, c.conn)
	case cmdPark:
		return fmt.Sprintf("%v(call %d)", c.kind, c.call)
	case cmdFail:
		return fmt.Sprintf("%v(call %d, %v)", c.kind, c.call, c.err)
	}
	return c.kind.String()
}
