package backup

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type BackupConfig struct {
	PrivateKey       []byte
	FileNameGetter   func() string
	BackupAtInterval bool
	Interval         time.Duration
}

type Controller struct {
	config BackupConfig
	core   *core
	quit   chan struct{}
	mutex  sync.Mutex
}

func NewController(config BackupConfig) (*Controller, error) {
	if len(config.PrivateKey) == 0 {
		return nil, fmt.Errorf("private key must be provided")
	}
	if config.FileNameGetter == nil {
		return nil, fmt.Errorf("filename getter must be provided")
	}

	return &Controller{
		config: config,
		core:   newCore(),
		quit:   make(chan struct{}),
	}, nil
}

func (c *Controller) Register(componentName string, dumpFunc func() ([]byte, error), loadFunc func([]byte) error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.core.Register(componentName, dumpFunc, loadFunc)
}

func (c *Controller) Start() {
	if !c.config.BackupAtInterval {
		return
	}

	go func() {
		ticker := time.NewTicker(c.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.PerformBackup()
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *Controller) Stop() {
	close(c.quit)
}

func (c *Controller) PerformBackup() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	backupData, err := c.core.Create(c.config.PrivateKey)
	if err != nil {
		return err
	}

	file, err := os.Create(c.config.FileNameGetter())
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(backupData)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) LoadBackup(filename string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	file, err := os.Open(filename)
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
