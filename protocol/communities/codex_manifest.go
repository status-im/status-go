package communities

// CodexManifest represents the manifest structure returned by Codex API
type CodexManifest struct {
	CID      string `json:"cid"`
	Manifest struct {
		TreeCid     string `json:"treeCid"`
		DatasetSize int64  `json:"datasetSize"`
		BlockSize   int    `json:"blockSize"`
		Protected   bool   `json:"protected"`
		Filename    string `json:"filename"`
		Mimetype    string `json:"mimetype"`
	} `json:"manifest"`
}
