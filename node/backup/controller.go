package backup

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
)

type BackupConfig struct {
	PrivateKey     []byte
	FileNameGetter func() (string, error)
	BackupEnabled  bool
	Interval       time.Duration
}

type Controller struct {
	config BackupConfig
	core   *core
	logger *zap.Logger
	quit   chan struct{}
	mutex  sync.Mutex
}

func NewController(config BackupConfig, logger *zap.Logger) (*Controller, error) {
	if len(config.PrivateKey) == 0 {
		return nil, errors.New("private key must be provided")
	}
	if config.FileNameGetter == nil {
		return nil, errors.New("filename getter must be provided")
	}

	return &Controller{
		config: config,
		core:   newCore(),
		logger: logger,
		quit:   make(chan struct{}),
	}, nil
}

func (c *Controller) Register(componentName string, dumpFunc func() ([]byte, error), loadFunc func([]byte) error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.core.Register(componentName, dumpFunc, loadFunc)
}

func (c *Controller) Start() {
	if !c.config.BackupEnabled {
		return
	}

	go func() {
		defer common.LogOnPanic()
		ticker := time.NewTicker(c.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := c.PerformBackup()
				if err != nil {
					c.logger.Error("Error performing backup: %v\n", zap.Error(err))
				}
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *Controller) Stop() {
	close(c.quit)
}

func (c *Controller) PerformBackup() (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	backupData, err := c.core.Create(c.config.PrivateKey)
	if err != nil {
		return "", err
	}

	fileName, err := c.config.FileNameGetter()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(fileName), 0700); err != nil {
		return "", err
	}

	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(backupData)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func (c *Controller) LoadBackup(filePath string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	backupData := make([]byte, fileInfo.Size())
	_, err = file.Read(backupData)
	if err != nil {
		return err
	}

	return c.core.Restore(c.config.PrivateKey, backupData)
}
