//go:build use_logos_storage
// +build use_logos_storage

package params

import "github.com/logos-storage/logos-storage-go-bindings/storage"

type LogosStorageConfig struct {
	Enabled                bool
	LogosStorageNodeConfig storage.Config
}
