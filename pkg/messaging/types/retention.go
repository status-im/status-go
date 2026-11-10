package types

import "time"

// StoreMessageRetention is the store-history window and lifetime of local SDS receipt evidence.
const StoreMessageRetention = 31 * 24 * time.Hour
