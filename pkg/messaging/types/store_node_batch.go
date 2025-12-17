package types

import "time"

type StoreNodeBatch struct {
	From        time.Time
	To          time.Time
	PubsubTopic string
	Topics      []ContentTopic
	ChatIDs     []string
}
