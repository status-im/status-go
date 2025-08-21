package types

type TransportStats struct {
	UploadRate   uint64 `json:"uploadRate"`
	DownloadRate uint64 `json:"downloadRate"`
}
