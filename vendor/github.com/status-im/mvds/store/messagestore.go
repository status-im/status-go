// Package store contains everything storage related for MVDS.
package store

import (
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
)

type MessageStore interface {
	Has(id state.MessageID) bool
	Get(id state.MessageID) (protobuf.Message, error)
	Add(message protobuf.Message) error
}
