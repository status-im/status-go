package dbi

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

type RegistrationRecord struct {
	Id    peer.ID
	Addrs [][]byte
	Ns    string
	Ttl   int
}

type DB interface {
	Close() error
	Register(p peer.ID, ns string, addrs [][]byte, ttl int) (uint64, error)
	Unregister(p peer.ID, ns string) error
	CountRegistrations(p peer.ID) (int, error)
	Discover(ns string, cookie []byte, limit int) ([]RegistrationRecord, []byte, error)
	ValidCookie(ns string, cookie []byte) bool
}
