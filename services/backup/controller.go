package backup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/signal"
)

//go:generate go tool mockgen -package=mock_backup_controller -source controller.go -destination=mock/mock_backup_controller.go

type Config struct {
	PrivateKey       []byte
	FileNameProvider FilenameProvider
	BackupEnabled    bool
	Interval         time.Duration
}

type FilenameProvider interface {
	GetBackupFilename() (string, error)
}

type Provider interface {
	ExportBackup() ([]byte, error)
	ImportBackup(data []byte) error
}

type Controller struct {
	common.PauseBroadcaster

	config Config
	core   *core
	logger *zap.Logger
	quit   chan struct{}
	mutex  sync.Mutex
	wg     *sync.WaitGroup
}

type CompletedEvent struct {
	FileName string
}

func (b CompletedEvent) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		FileName string `json:"fileName,omitempty"`
	}{
		FileName: b.FileName,
	}
	return json.Marshal(responseItem)
}

func NewController(config Config, logger *zap.Logger) (*Controller, error) {
	if len(config.PrivateKey) == 0 {
		return nil, errors.New("private key must be provided")
	}
	if common.IsNil(config.FileNameProvider) {
		return nil, errors.New("filename provider must be provided")
	}

	return &Controller{
		config: config,
		core:   newCore(),
		logger: logger,
		wg:     &sync.WaitGroup{},
		quit:   make(chan struct{}),
	}, nil
}

func (c *Controller) Register(componentName string, provider Provider) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.core.Register(componentName, provider)
}

func (c *Controller) Start() {
	if !c.config.BackupEnabled {
		return
	}
	c.wg.Add(1)

	go func() {
		defer common.LogOnPanic()
		defer c.wg.Done()
		sub := c.Subscribe()
		defer sub.Unsubscribe()
		pt := common.NewPausableTicker(common.PausableTickerConfig{
			Interval: c.config.Interval,
			OnTick: func() {
				_, err := c.PerformBackup()
				if err != nil {
					c.logger.Error("Error performing backup: %v\n", zap.Error(err))
				}
			},
		}, sub.C())
		pt.Run(c.quit)
	}()
}

func (c *Controller) Stop() {
	close(c.quit)
	c.wg.Wait()
}

func (c *Controller) PerformBackup() (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	backupData, err := c.core.Create(c.config.PrivateKey)
	if err != nil {
		return "", err
	}

	fileName, err := c.config.FileNameProvider.GetBackupFilename()
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

	signal.SendLocalBackUpCompleted(CompletedEvent{
		FileName: fileName,
	})

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
