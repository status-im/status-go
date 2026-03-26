# 📚 status-backend API Reference

Practical API reference for building agents/bots and integrations with `status-backend`.
For architecture overview and how to read the source code, see [README.md](./README.md).

---

## 📍 HTTP Endpoints

### GET /health

Health check. Returns HTTP 200 with `{"version": "..."}` if operational.

---

### POST /statusgo/InitializeApplication

Initialize the application. Must be called before login or account creation.

**Request:**
```json
{"dataDir": "/path/to/data"}
```

**Response:**
```json
{
  "accounts": [
    {"key-uid": "0xabc...", "name": "MyBot", "identicon": "..."}
  ]
}
```

- `accounts`: list of existing accounts in the data directory.
- An empty `accounts` array means this is a fresh installation and you need to create an account.

---

### POST /statusgo/CreateAccountAndLogin

Create a new account and log in. Generates a BIP39 mnemonic internally.

**Request:**
```json
{
  "rootDataDir": "/path/to/data",
  "displayName": "MyBot",
  "password": "...",
  "customizationColor": "primary"
}
```

**Response:** `{"error": ""}` on success.

> [!NOTE]
> The `keyUID` is NOT in the HTTP response — it arrives via the `node.login` signal event, or can be read from `settings_getSettings` after login.

**Signal:** Fires `node.login` signal asynchronously.

---

### POST /statusgo/LoginAccount

Log in to an existing account.

**Request:**
```json
{"keyUID": "0xabc...", "password": "..."}
```

**Response:** `{"error": ""}` — but this can be **misleading**. The HTTP response may contain an error if request parsing or validation failed. However, a successful HTTP response (`{"error": ""}`) does not mean login succeeded — the actual login result comes asynchronously via the `node.login` WebSocket signal.

> [!IMPORTANT]
> Always wait for the `node.login` signal and check its `error` field. Do NOT trust the HTTP response alone.

> [!IMPORTANT]
> Calling `LoginAccount` when already logged in returns `{"error": "node is already running"}` or may crash the backend. Always check `settings_getSettings` for a `public-key` field first to determine if already logged in.

---

### POST /statusgo/CallRPC

JSON-RPC gateway to all status-go services.

**Request:**
```json
{"jsonrpc": "2.0", "id": 1, "method": "wakuext_methodName", "params": [...]}
```

**Response:**
```json
{"jsonrpc": "2.0", "id": 1, "result": "...", "error": {"code": -32000, "message": "..."}}
```

> [!NOTE]
> The `id` field is **required** — omitting it causes empty or broken responses.

> [!NOTE]
> JSON-RPC errors are **objects** `{code, message}`, while HTTP API errors (`InitializeApplication`, `LoginAccount`) are **strings**. Handle both formats in your client.

---

## 📡 WebSocket Signals

### Connection

Connect to `ws://<host>:<port>/signals` as the **first step** before any other API calls.

> [!IMPORTANT]
> `status-backend` does NOT respond to WebSocket pings. Disable keepalive/ping in your WebSocket client or the connection will be dropped.

### Signal Format

```json
{"type": "signal.name", "event": {...}}
```

### Key Signals

| Signal | When | Key event fields |
|--------|------|-----------------|
| `node.login` | After `LoginAccount` or `CreateAccountAndLogin` completes | `error`, `settings`, `account` |
| `node.started` | Node process started | — |
| `node.ready` | Node fully initialized | — |
| `messages.new` | New messages received | `messages[]` — array of message objects |
| `mediaserver.started` | Media server ready | — |
| `mailserver.changed` | Connected to mailserver | — |
| `mailserver.available` | Mailserver available | — |
| `history.request.started` | Fetching message history | — |
| `history.request.completed` | History fetch done | — |
| `envelope.sent` | Message envelope sent via Waku | — |

### `node.login` Signal Details

```json
{
  "type": "node.login",
  "event": {
    "error": "",
    "settings": {...},
    "account": {...}
  }
}
```

Special error values:
- `""` (empty) — login successful.
- `"node is already running"` — can be treated as success (already logged in).
- `"failed to get account kdf iterations..."` — the `keyUID` doesn't exist in the data directory.

---

## 🔧 RPC Methods

All methods are called via `POST /statusgo/CallRPC` with JSON-RPC format.

---

### settings_getSettings

Get current account settings.

**Params:** `[]`

**Result:**
```json
{
  "public-key": "0x04abc...",
  "display-name": "MyBot",
  "key-uid": "0xabc...",
  ...
}
```

**Usage:** Check if `result["public-key"]` exists to determine whether the backend is logged in.

---

### wakuext_startMessenger

Start the Waku messenger service. Must be called after login.

**Params:** `[]`

**Behavior:**
- Typically completes in several seconds.
- Subsequent calls return immediately ("already started").
- Will return an error if something went wrong.

The standard approach is to call it synchronously (blocking). If you need to do other work in parallel, you can call it asynchronously as a workaround — in that case, poll readiness by calling `wakuext_joinedCommunities` until it stops returning `"does not exist"` errors.

---

### wakuext_joinedCommunities

List communities the user has joined.

**Params:** `[]`

**Result:**
```json
[{
  "id": "0x03abc...",
  "name": "Community Name",
  "joined": true,
  "isMember": true,
  "verified": true,
  "members": {
    "0x04pubkey...": {"roles": [1]}
  },
  "chats": {
    "chat-uuid": {
      "name": "general",
      "canPost": true
    }
  }
}]
```

#### ⚠️ `joined` vs `isMember`

This distinction is critical for bot developers:

| Field | Meaning | Set by |
|-------|---------|--------|
| `joined` | "I want to be in this community" (local intent) | Local `joinCommunity` call |
| `isMember` | "My public key is in the community's Members map" (network state) | Community control node (owner) |

**`joined: true` + `isMember: false`** means the bot expressed intent to join locally, but the community owner's node hasn't added the bot to the members list yet (or the updated community description hasn't propagated via Waku).

**Only `isMember: true` allows posting messages.** The `canPost` field on a channel requires `isMember: true` at the community level.

> [!NOTE]
> The `joinCommunity` API is deprecated and planned for removal (see [#7381](https://github.com/status-im/status-go/issues/7381)). Once removed, `joined` and `isMember` will converge into a single field.

---

### wakuext_fetchCommunity

Fetch a community description from the Waku network.

**Params:**
```json
[{
  "communityKey": "0x03abc...",
  "tryDatabase": true,
  "waitForResponse": true
}]
```

**Result:** Community object (description, name, members, chats, etc.)

**Usage:** Call before joining to ensure the local node has the community description. Without this, `requestToJoinCommunity` may fail with `"community not found"`.

---

### wakuext_spectateCommunity

Subscribe to a community's Waku pubsub topics without joining. This allows reading community messages before becoming a member.

**Params:** `["0xcommunityId"]`

> [!NOTE]
> Spectating is **not required** for joining a community. The join request and acceptance are exchanged via the default Waku topic, not community-specific topics. However, spectating can be useful to read community messages while waiting for membership approval.

---

### wakuext_requestToJoinCommunity

Send a join request to the community's control node via Waku.

**Params:**
```json
[{
  "communityId": "0x03abc...",
  "addressesToReveal": ["0xWalletAddress"],
  "airdropAddress": "0xWalletAddress"
}]
```

**What it does:**
1. Validates request parameters.
2. Checks if already a member (returns error if so).
3. Creates a `CommunityRequestToJoin` protobuf message.
4. Sends the request to the community control node via Waku pubsub.
5. For open communities: control node auto-accepts.
6. For closed communities: control node queues for manual approval.

**`addressesToReveal`:** Required. Pass at least one wallet address from `accounts_getAccounts`. Even for open communities without token gates, the protocol expects addresses to be revealed.

> [!IMPORTANT]
> Do NOT use `wakuext_joinCommunity` for joining communities. That method is **local-only** — it sets `joined: true` but never contacts the control node. `isMember` will remain `false` and you won't be able to post. Always use `requestToJoinCommunity`. See [#7381](https://github.com/status-im/status-go/issues/7381).

---

### wakuext_joinCommunity

> [!WARNING]
> **Deprecated.** This method is planned for removal — see [#7381](https://github.com/status-im/status-go/issues/7381). Use `wakuext_requestToJoinCommunity` instead.

**LOCAL-ONLY.** Sets `joined: true` in local database but does NOT send any network request.

**Params:** `["0xcommunityId"]`

**What it does:**
1. Sets `joined = true` in local DB.
2. Initializes community chats and filters.
3. Does NOT contact the community owner.
4. Does NOT add the user to the Members map.

---

### wakuext_sendChatMessage

Send a text message to a chat/channel.

**Params:**
```json
[{
  "chatId": "<chatId>",
  "text": "Hello!",
  "contentType": 1
}]
```

**Chat ID formats:**

- **Community channel:** concatenation of community ID and chat UUID without separator:
  `0x03ab2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b28a805944-9f5d-4219-a104-5d2047f5572a`

- **1:1 chat:** the other user's public key:
  `0x0416afecc10e3a1ab4c4c7efe03d845c5511f783542e36b1d56bff454214726c6035b8632c012d4fba1bbcd311e39ba33357d3b93bce64c1b176ca0a2f85f0f701`

**Common error:** `"can't post message type '1' on chat '...'"` — this means `canPost` is false on the channel, usually because `isMember` is false at the community level. See the [Community Join Sequence](#-community-join-sequence) section.

---

### wakuext_chatMessages

Read message history for a chat.

**Params:** `[chatId, cursor, limit]`
- `chatId`: string (communityId + chatUUID)
- `cursor`: string (empty string for first page)
- `limit`: number

---

### wakuext_leaveCommunity

Leave a community.

**Params:** `["0xcommunityId"]`

---

### wakuext_communities

List ALL communities (joined + known but not joined).

**Params:** `[]`

---

### accounts_getAccounts

List wallet accounts and addresses.

**Params:** `[]`

**Result:**
```json
[
  {"address": "0xabc...", "wallet": true, "chat": false, "type": "generated"},
  {"address": "0xdef...", "wallet": false, "chat": true, "type": "key"}
]
```

**Usage:** Find the wallet address (`wallet: true`) for the `addressesToReveal` parameter in `requestToJoinCommunity`.

---

## 🚀 Startup Sequence

The correct order of operations for a bot:

```
1. Connect WebSocket to /signals
2. POST /statusgo/InitializeApplication → get existing accounts
3. Login or Create Account:
   a. If you have a stored keyUID → POST /statusgo/LoginAccount
   b. If accounts exist in response → LoginAccount with first account's key-uid
   c. Otherwise → POST /statusgo/CreateAccountAndLogin
4. Wait for node.login signal (check the error field!)
5. POST /statusgo/CallRPC with wakuext_startMessenger
6. RPC methods are now available
```

> [!TIP]
> Store the `keyUID` after first account creation so you can use `LoginAccount` on subsequent startups.

---

## 🏘️ Community Join Sequence

The correct order for joining a community:

```
1. Get wallet address: accounts_getAccounts → find the entry with wallet: true
2. Fetch community description: wakuext_fetchCommunity
3. Request to join: wakuext_requestToJoinCommunity with addressesToReveal
4. Wait for community owner to accept (open communities: auto-accept)
5. isMember becomes true → wakuext_sendChatMessage now works
```

> [!TIP]
> Optionally call `wakuext_spectateCommunity` before step 3 to read community messages while waiting for approval.

### Common Mistakes

1. **Using `wakuext_joinCommunity` instead of `requestToJoinCommunity`** — `joinCommunity` is local-only and will never result in `isMember: true`. See [#7381](https://github.com/status-im/status-go/issues/7381).

2. **Skipping `fetchCommunity`** — without fetching the community description first, `requestToJoinCommunity` may fail with `"community not found"`.

3. **Not waiting for membership** — after `requestToJoinCommunity`, the bot must wait for the community owner's control node to process the request. For open communities this is automatic but not instant.

---

## ❌ Error Reference

| Error | Meaning | Resolution |
|-------|---------|------------|
| `"does not exist"` | RPC handler not registered yet | Wait for `startMessenger` to complete |
| `"node is already running"` | `LoginAccount` called on a running node | Check `settings_getSettings` for `public-key` before logging in |
| `"can't post message type '1'"` | `isMember` is false, `canPost` is false | Use `requestToJoinCommunity`, wait for membership confirmation |
| `"community already joined"` | `joinCommunity` called when already joined | Skip, or use `requestToJoinCommunity` |
| `"community not found"` | Community description not yet fetched | Call `fetchCommunity` first |
| `"method handler crashed"` | Internal RPC handler error | Retry after delay; ensure messenger is started |
| `"failed to get account kdf iterations"` | `keyUID` doesn't exist in data directory | Use a different `keyUID` or create a new account |
| `"ErrPermissionToJoinNotSatisfied"` | Wallet doesn't meet token requirements | Check community's token gate requirements |

---

## ⚠️ Common Gotchas

1. **`LoginAccount` is asynchronous** — The HTTP response returns immediately with `{"error": ""}` even on failure. Always wait for the `node.login` WebSocket signal and check its `error` field.

2. **`LoginAccount` on a running node crashes** — Always check if already logged in via `settings_getSettings` before calling `LoginAccount`.

3. **WebSocket pings are not supported** — `status-backend` does not respond to WebSocket ping frames. Disable keepalive/ping in your WebSocket client.

4. **JSON-RPC `id` field is required** — Omitting the `id` field in `CallRPC` requests causes empty or broken responses.

5. **Two error formats** — HTTP API endpoints return `{"error": "string"}`. JSON-RPC returns `{"error": {"code": -32000, "message": "string"}}`. Handle both in your client.
