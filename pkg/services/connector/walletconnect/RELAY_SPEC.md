# WalletConnect relay connection — specification

This document specifies how `RelayClient` manages its connection to the
WalletConnect relay. It is written against behaviour, not functions: every rule
has an ID, and every ID has a test named after it.

The goal of the refactor it drives is to move every connection decision into a
pure state machine (`Machine.Step`) and leave `RelayClient` as a thin shell that
turns sockets and timers into events and executes the machine's commands.

## Scope

In scope:

- the connection lifecycle: first connect, loss, redial with backoff, close;
- whether a relay call (`Subscribe`, `Publish`, `FetchMessages`, `Unsubscribe`)
  may use the connection now, must wait, or must fail.

Out of scope (non-goals):

- the WalletConnect protocol in `Client`: pairing, sessions, encryption;
- JSON-RPC framing and matching responses to requests — the shell keeps doing
  this with its `pending` map;
- dialing itself (JWT, URL, handshake classification in `dialfailure.go`);
- session persistence and the connector service's retry of restored sessions.

## Terms

- **Call** — one relay JSON-RPC request made through the `Relay` interface.
- **Parked call** — a call waiting for a connection while the client redials.
- **Wait budget** — how long a parked call waits: `relayReconnectWait` (5s).
- **Backoff** — the pause before the next redial: starts at
  `relayReconnectBackoff` (1s), doubles after each failed dial, capped at
  `relayReconnectMaxBackoff` (1m).

## External contract

What a user of the `Relay` interface observes. These rules hold before and after
the refactor and are tested black-box through `relaytest`.

| ID  | Rule |
|-----|------|
| C1  | A call before the first successful `Connect` fails at once with "not connected". |
| C2  | `Connect` on a live connection returns nil without dialing. |
| C3  | `Connect` after `Close` returns `ErrRelayClosed` without dialing. |
| C4  | A failed first `Connect` returns the dial error; the client stays usable and a later `Connect` dials again. |
| C5  | When the connection is lost, the client redials with backoff until it connects or is closed. There is no attempt limit. |
| C6  | A call made while the client redials waits at most the wait budget, then fails with "reconnect in progress". It never waits on the redial itself. |
| C7  | A parked call is served on the new connection if the redial succeeds within the wait budget. |
| C8  | After a redial succeeds, the reconnected handler runs exactly once, on its own goroutine, so it may make calls. It does not run after the first `Connect`. |
| C9  | `Close` returns promptly even when the relay never answers the handshake, and aborts that handshake. |
| C10 | After `Close`, every pending and parked call fails with "shutting down", and no socket is opened again. `Close` is idempotent. |
| C11 | A call whose request is written but gets no response fails after 30s with "relay call timeout". |
| C12 | A failed write marks the connection lost; the call parks and is written once more on the next connection (C6/C7 apply). |
| C13 | A missed heartbeat (no pong within `relayReadDeadline`) or a failed ping marks the connection lost. |
| C14 | Every `irn_subscription` message is acknowledged on the connection it arrived on; the relay delivers unacknowledged messages again after every re-subscribe. |

## States

| State        | Meaning |
|--------------|---------|
| `Idle`       | No connection and none attempted since creation or since the last failed first connect. |
| `Dialing`    | A dial is in flight. Carries `attempt`, `first` (true for a dial started by the first `Connect`) and, for a redial, the `delay` it waited. |
| `Connected`  | A live connection. Carries `conn`. |
| `Backoff`    | Waiting before the next redial. Carries `attempt` and `delay`. |
| `Closed`     | `Close` was called. Terminal. |

`Machine` also holds `everConnected` (a first `Connect` has succeeded) and the
list of parked calls.

## Events

| Event                 | Raised by the shell when |
|-----------------------|--------------------------|
| `Connect`             | `Connect()` is called. |
| `DialOK(conn)`        | a dial returned a connection. |
| `DialFailed(err)`     | a dial returned an error. |
| `ConnLost(conn)`      | a read or write on `conn` failed, or its heartbeat expired. |
| `BackoffElapsed`      | the backoff timer fired. |
| `Close`               | `Close()` is called. |
| `Call(id)`            | a call needs a connection. |
| `WaitExpired(id)`     | a parked call's wait budget ran out. |

## Commands

| Command                   | The shell must |
|---------------------------|----------------|
| `StartDial`               | start one dial. |
| `AbortDial`               | abort the dial in flight (close the handshake socket). |
| `CloseConn(conn)`         | close `conn`. |
| `StartBackoff(d)`         | start the backoff timer for `d`. |
| `StopBackoff`             | stop the backoff timer. |
| `StartHeartbeat(conn)`    | start pinging `conn`. |
| `StartReader(conn)`       | start reading `conn`. |
| `NotifyReconnected`       | run the reconnected handler on its own goroutine. |
| `ReplyConnect(err)`       | return `err` from the pending `Connect()` (nil on success). |
| `Serve(id, conn)`         | let call `id` use `conn`. |
| `Park(id)`                | hold call `id` and start its wait budget. |
| `Fail(id, err)`           | fail call `id` with `err`. |

`Serve`/`Fail` for all parked calls are emitted one per call, in parking order.

## Transitions

A blank cell means the event is ignored in that state: no state change, no
commands. Every non-blank cell is one rule with the ID `T-<State>-<Event>`.

### Idle

| Event            | Next state                   | Commands |
|------------------|------------------------------|----------|
| `Connect`        | `Dialing{attempt:1, first}`  | `StartDial` |
| `Close`          | `Closed`                     | |
| `Call(id)`       | `Idle`                       | `Fail(id, "not connected")` |

### Dialing (first)

| Event            | Next state                   | Commands |
|------------------|------------------------------|----------|
| `Connect`        | `Dialing` (unchanged)        | — the second caller gets the same reply |
| `DialOK(c)`      | `Connected{c}`, `everConnected = true` | `StartHeartbeat(c)`, `StartReader(c)`, `ReplyConnect(nil)` |
| `DialFailed(e)`  | `Idle`                       | `ReplyConnect(e)` |
| `Close`          | `Closed`                     | `AbortDial`, `ReplyConnect(ErrRelayClosed)` |
| `Call(id)`       | `Dialing` (unchanged)        | `Fail(id, "not connected")` |

### Dialing (redial, attempt n)

| Event            | Next state                   | Commands |
|------------------|------------------------------|----------|
| `Connect`        | `Dialing` (unchanged)        | — reply comes with the dial result |
| `DialOK(c)`      | `Connected{c}`               | `StartHeartbeat(c)`, `StartReader(c)`, `Serve` each parked call, `NotifyReconnected`, `ReplyConnect(nil)` if a `Connect` is waiting |
| `DialFailed(e)`  | `Backoff{n, d}` with `d = min(2·delay, max)` | `StartBackoff(d)`, `ReplyConnect(e)` if a `Connect` is waiting |
| `Close`          | `Closed`                     | `AbortDial`, `Fail` each parked call with "shutting down", `ReplyConnect(ErrRelayClosed)` if waiting |
| `Call(id)`       | `Dialing` (unchanged)        | `Park(id)` |
| `WaitExpired(id)`| `Dialing` (unchanged)        | `Fail(id, "reconnect in progress")` |

### Connected{c}

| Event                 | Next state                         | Commands |
|-----------------------|------------------------------------|----------|
| `Connect`             | `Connected` (unchanged)            | `ReplyConnect(nil)` |
| `ConnLost(c)`         | `Backoff{attempt:1, delay:initial}`| `CloseConn(c)`, `StartBackoff(initial)` |
| `ConnLost(other)`     |                                    | ignored: the event is about a replaced connection |
| `Close`               | `Closed`                           | `CloseConn(c)` |
| `Call(id)`            | `Connected` (unchanged)            | `Serve(id, c)` |

### Backoff{n, delay}

| Event            | Next state                   | Commands |
|------------------|------------------------------|----------|
| `Connect`        | `Dialing{n+1, delay}`        | `StopBackoff`, `StartDial` — an explicit `Connect` skips the wait |
| `BackoffElapsed` | `Dialing{n+1, delay}`        | `StartDial` |
| `Close`          | `Closed`                     | `StopBackoff`, `Fail` each parked call with "shutting down" |
| `Call(id)`       | `Backoff` (unchanged)        | `Park(id)` |
| `WaitExpired(id)`| `Backoff` (unchanged)        | `Fail(id, "reconnect in progress")` |

### Closed

| Event            | Next state | Commands |
|------------------|------------|----------|
| `Connect`        | `Closed`   | `ReplyConnect(ErrRelayClosed)` |
| `DialOK(c)`      | `Closed`   | `CloseConn(c)` — a dial that finished after `Close` |
| `Call(id)`       | `Closed`   | `Fail(id, "shutting down")` |

Notes on the table:

- `ConnLost` in any state other than `Connected` is ignored: only the live
  connection can be lost.
- `DialOK`/`DialFailed` outside `Dialing` and `Closed` cannot happen (the machine
  starts at most one dial) and are ignored.
- `WaitExpired` for an id that is no longer parked is ignored.
- Two decisions differ from the pre-refactor code and are deliberate:
  a `Connect` during a redial joins it instead of dialing in parallel, and a
  `Connect` during backoff dials at once.

## Invariants

Checked after every step of randomly generated event sequences.

| ID  | Invariant |
|-----|-----------|
| I1  | At most one connection is live, and at most one dial is in flight. |
| I2  | Every `Call(id)` eventually gets exactly one `Serve(id)` or `Fail(id)`; parked calls are never left behind by a state change. |
| I3  | In `Closed`, no command starts a dial or a heartbeat, and every connection handed to the machine is closed. |
| I4  | Backoff delays are non-decreasing within one outage and never exceed the maximum; a successful dial resets them. |
| I5  | `NotifyReconnected` is emitted only on `DialOK` of a redial, once per redial. |
| I6  | Every `Connect` gets exactly one `ReplyConnect`. |

## Shell

`RelayClient` keeps its public API and the `Relay` interface unchanged.
Internally:

- one goroutine owns the `Machine`: it receives events on a channel, calls
  `Step`, and executes the returned commands;
- `Connect`, `Close` and calls send an event and wait for their command
  (`ReplyConnect`, `Serve`/`Fail`);
- readers, heartbeats, dials and timers send events back; each is tied to the
  connection or dial it serves, so events about a replaced connection are
  recognisable (`ConnLost(other)`);
- the `mu`, `reconnectMu` and `connSet` fields go away; `writeMu` stays
  (gorilla allows one writer), and the `pending` map for responses stays;
- every goroutine defers `panics.LogOnPanic()`.

## Tests

| Layer | File | What |
|-------|------|------|
| Contract | `relay_*_test.go` | One test per `C*` rule, black-box through `relaytest`. Written against the current code first and green there; must stay green after the refactor, also with `-race -count=10`. |
| Transitions | `machine_test.go` | One table row per `T-*` rule: state, event, expected state, expected commands. |
| Completeness | `machine_test.go` | Enumerates every state × event pair and fails if a pair has no row (a blank cell must be listed as "ignored"). |
| Invariants | `machine_test.go` | Random event sequences from a fixed seed, `I1`–`I6` checked after every step; a failure prints the seed and the sequence. |

Test names carry the rule ID, e.g. `TestRelay_C6_CallWaitsAtMostTheBudget`,
`TestMachine_T_Closed_DialOK`.
