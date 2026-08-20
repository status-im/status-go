//go:build disable_history_archives || !use_logos_storage

package protocol

import (
	"errors"

	"github.com/status-im/status-go/pkg/services/logosstorage"
)

func (m *Messenger) Connect(_ string, _ []string) error {
	return errors.New("logos storage is not enabled in this build")
}

func (m *Messenger) Debug() (logosstorage.LogosStorageDebugInfo, error) {
	return logosstorage.LogosStorageDebugInfo{}, errors.New("logos storage is not enabled in this build")
}
