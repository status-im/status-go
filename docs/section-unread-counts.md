# Section unread counts

`wakuext_sectionUnreadCounts` returns badge-ready totals from a single local
database snapshot. It does not load thread objects, summaries, participants, or
last messages.

## Request and response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "wakuext_sectionUnreadCounts",
  "params": [{"includeThreads": true}]
}
```

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "sections": [
      {
        "sectionId": "personal",
        "unviewedMessagesCount": 4,
        "unviewedMentionsCount": 2
      },
      {
        "sectionId": "0x0123",
        "unviewedMessagesCount": 12,
        "unviewedMentionsCount": 1
      }
    ]
  }
}
```

`includeThreads` defaults to false. Omitted or null request arguments also
exclude threads. Threads contribute only when both the request and the
Messenger's Threads feature flag enable them.

The result is a complete replacement snapshot, not a delta. It always includes
`personal`, followed by each joined or spectated community in section ID order,
including sections with zero counts. Communities that are neither joined nor
spectated are absent; clients must remove their cached totals. Clients map
`personal` to their own personal-section identifier.

Database and cancellation errors are returned as RPC errors, not empty results.

## Counting rules

- Normal chat counts use persisted counters for active, known-type chats.
  Personal chats have an empty community ID; community chats belong to the
  corresponding joined or spectated community.
- Eligible threads are followed, non-deleted, and non-archived. Archive state
  uses the same inactivity cutoff as thread lists.
- Thread unread messages must be unseen, not hidden, not deleted, and not
  deleted for the local user.
- Mention counts preserve existing semantics: mentions or replies, and every
  unread message in one-to-one threads.
- Mentions remain visible when a chat or community is muted. A muted chat or
  thread contributes its unread count only when that same chat or thread has
  mentions. Mentions in another chat or thread do not bypass its mute.
- Parent-chat and thread counts remain independent. This query does not modify
  either counter.

Use `unviewedMessagesCount > 0` for the unread marker and
`unviewedMentionsCount` for the numeric notification badge.

The thread query uses a partial unread-message index, avoiding reads of message
bodies and historical seen messages. With threads excluded, neither the thread
table nor the message table is queried.

## Refresh protocol

Fetch a snapshot during startup without waiting for section chat models or
thread lists. Listen for invalidations before starting the first fetch.

Messenger responses (both RPC results and `messages.new` events) include
`sectionUnreadCountsChanged: true` when they carry potentially relevant chat,
thread, message, community, history, removal, or read-state changes. This is a
conservative invalidation, not a claim that the numeric total changed.

Mutations that do not necessarily return a relevant Messenger response also
emit `section.unread_counts.changed` with a null event payload. These cover
read-state changes, mute/unmute, direct chat/message deletion, direct chat save,
thread follow/unfollow and deletion, and backup restoration. These invalidations
are emitted after persistence even if a subsequent sync or response-building
step fails. Existing timed-unmute and thread-archive events carry the
Messenger-response invalidation as well.

Coalesce invalidations into a refresh. Allow only one fetch at a time; when
invalidated during a fetch, schedule another fetch after it completes. Never
overwrite a newer completed snapshot with an older request. Only replace
cached totals after a successful fetch; report errors and retry on subsequent
invalidations or an explicit retry. Continue loading individual thread lists
and summaries lazily when their UI needs them.

## Validation

From the status-go root:

```sh
make test-single PKG=./internal/protocol TEST='^TestSectionUnreadCounts'
```

`BenchmarkSectionUnreadCountsLargeAccount` exercises 100 communities, 1,000
chats, 20,000 threads, and 200,000 messages, with thread inclusion both enabled
and disabled:

```sh
make test-single PKG='./internal/protocol -bench=BenchmarkSectionUnreadCountsLargeAccount -benchmem -benchtime=3x' TEST='^$'
```
