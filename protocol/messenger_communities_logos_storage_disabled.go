//go:build disable_history_archives || !use_logos_storage
// +build disable_history_archives !use_logos_storage

package protocol

import (
	"errors"

	logosstorage "github.com/status-im/status-go/services/logosstorage"
)

func (m *Messenger) Connect(peerId string, addrs []string) error {
	return errors.New("logos storage is not enabled in this build")
}

func (m *Messenger) Debug() (logosstorage.LogosStorageDebugInfo, error) {
	return logosstorage.LogosStorageDebugInfo{}, errors.New("logos storage is not enabled in this build")
}
