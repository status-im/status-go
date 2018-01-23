// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package kafka

import (
	"time"
)

// Metadata is used to store metadata for the sarama.ProducerMessages
type Metadata struct {
	StartTime   time.Time
	NumMessages int
}
