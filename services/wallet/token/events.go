package token

import "github.com/status-im/status-go/services/wallet/walletevent"

// EventTokenListsUpdated is emitted on the wallet feed when the token lists
// finish loading/refreshing, so in-process consumers (e.g. the balance
// controller) can react without waiting for a periodic re-fetch.
const EventTokenListsUpdated walletevent.EventType = "wallet-token-lists-updated"
