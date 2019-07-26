// Package store contains everything storage related for MVDS.
package store

import (
	"github.com/vacp2p/mvds/protobuf"
	"github.com/vacp2p/mvds/state"
)

type MessageStore interface {
	Has(id state.MessageID) (bool, error)
	Get(id state.MessageID) (*protobuf.Message, error)
	Add(message *protobuf.Message) error
}
