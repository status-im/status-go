//go:build !use_logos_storage
// +build !use_logos_storage

package logosstorage

import (
	"errors"

	"github.com/status-im/status-go/params"
)

func ApplyLogosStorageConfigOverrides(_ *params.LogosStorageConfig, _ map[string]string) error {
	return errors.New("logos storage support is not enabled in this build")
}
