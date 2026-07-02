//go:build use_logos_storage && !lint

package logosstorage

import "github.com/logos-storage/logos-storage-go-bindings/storage"

type LogosStorageManifest struct {
	Cid         string
	TreeCid     string
	DatasetSize int
	BlockSize   int
	Filename    string
	Mimetype    string
	Protected   bool
}

func toLogosStorageManifest(manifest storage.Manifest) LogosStorageManifest {
	return LogosStorageManifest{
		Cid:         manifest.Cid,
		TreeCid:     manifest.TreeCid,
		DatasetSize: manifest.DatasetSize,
		BlockSize:   manifest.BlockSize,
		Filename:    manifest.Filename,
		Mimetype:    manifest.Mimetype,
		Protected:   manifest.Protected,
	}
}
