package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
)

type CacheConfigOutputFileInfo struct {
	Hash    string
	Path    string
	ModTime time.Time
}

type CacheConfig struct {
	OutputFiles []CacheConfigOutputFileInfo
}

func GetConfigFilePath(cacheHitDir string) string {
	return path.Join(cacheHitDir, "cache.json")
}

func SaveConfig(config CacheConfig, cacheHitDir string) error {
	file, err := os.Create(GetConfigFilePath(cacheHitDir))
	if err != nil {
		return fmt.Errorf("cannot create cache config file: %w", err)
	}
	defer file.Close()

	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("cannot marshal cache config file: %w", err)
	}

	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("cannot write cache config file: %w", err)
	}
	return nil
}

func LoadConfig(cacheHitDir string) (CacheConfig, error) {
	var config CacheConfig

	fileData, err := os.ReadFile(GetConfigFilePath(cacheHitDir))
	if err != nil {
		return CacheConfig{}, fmt.Errorf("cannot read cache config file: %w", err)
	}

	// Unmarshal JSON data to struct
	err = json.Unmarshal(fileData, &config)
	if err != nil {
		return CacheConfig{}, fmt.Errorf("cannot unmarshal cache config file: %w", err)
	}

	return config, nil
}
