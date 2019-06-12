package client

import (
	"time"

	"github.com/status-im/status-console-client/protocol/v1"
)

// History is used to track when contact was synced last time.
// Contact extension. Deleted on cascade when parent contact is deleted.
type History struct {
	// Synced is a timestamp in seconds.
	Synced  int64
	Contact Contact
}

func splitIntoSyncedNotSynced(histories []History) (sync []History, nosync []History) {
	for i := range histories {
		if histories[i].Synced != 0 {
			sync = append(sync, histories[i])
		} else {
			nosync = append(nosync, histories[i])
		}
	}
	return
}

func syncedToOpts(histories []History, now time.Time) protocol.RequestOptions {
	opts := protocol.RequestOptions{
		To:    now.Unix(),
		Limit: 1000,
	}
	for i := range histories {
		if opts.From == 0 || opts.From > histories[i].Synced {
			opts.From = histories[i].Synced
		}
		// TODO(dshulyak) remove contact type validation in that function
		// simply always add topic and (if set) public key
		_ = enhanceRequestOptions(histories[i].Contact, &opts)
	}
	return opts
}

func notsyncedToOpts(histories []History, now time.Time) protocol.RequestOptions {
	opts := protocol.DefaultRequestOptions()
	opts.To = now.Unix()
	for i := range histories {
		// TODO(dshulyak) remove contact type validation in that function
		// simply always add topic and (if set) public key
		_ = enhanceRequestOptions(histories[i].Contact, &opts)
	}
	return opts
}
