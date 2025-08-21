package requests

import "errors"

type LoadLocalBackup struct {
	FilePath string `json:"filePath"`
}

func (c *LoadLocalBackup) Validate() error {
	if c.FilePath == "" {
		return errors.New("filePath must be provided")
	}
	return nil
}
