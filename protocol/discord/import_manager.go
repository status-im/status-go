package discord

import (
	"time"
	"strings"
	"os"
	"encoding/json"
	"go.uber.org/zap"
)

const discordTimestampLayout = "2006-01-02T15:04:05+00:00"

type ImportManager struct {
	subscriptions                []chan *Subscription
	logger                       *zap.Logger
	quit                         chan struct{}
	importTasks                  []*ImportTask
}

func NewImportManager(logger *zap.Logger) (*ImportManager, error) {
  manager := &ImportManager{
    logger: logger,
  }

  return manager, nil
}

func (m *ImportManager) Subscribe() chan *Subscription {
	subscription := make(chan *Subscription, 100)
	m.subscriptions = append(m.subscriptions, subscription)
	return subscription
}

func (m *ImportManager) publishImportProgress(subscription *Subscription) {
  for _, s := range m.subscriptions {
    select {
    case s <- subscription:
		default:
			m.logger.Warn("subscription channel full, dropping message")
    }
  }
}

func (m *ImportManager) StopImports() error {
  close(m.quit)
  return nil
}

func (m *ImportManager) ExtractDiscordDataFromImportFiles(filesToImport []string) (*ExtractedData, map[string]*ImportError) {

	extractedData := &ExtractedData{
		Categories:             map[string]*Category{},
		ExportedData:           make([]*ExportedData, 0),
		OldestMessageTimestamp: 0,
		MessageCount:           0,
	}

	errors := map[string]*ImportError{}

	for _, fileToImport := range filesToImport {
		filePath := strings.Replace(fileToImport, "file://", "", -1)
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			errors[fileToImport] = Error(err.Error())
			continue
		}

		var discordExportedData ExportedData

		err = json.Unmarshal(bytes, &discordExportedData)
		if err != nil {
			errors[fileToImport] = Error(err.Error())
			continue
		}

		if len(discordExportedData.Messages) == 0 {
			errors[fileToImport] = Error(ErrNoMessageData.Error())
			continue
		}

		discordExportedData.Channel.FilePath = filePath
		categoryID := discordExportedData.Channel.CategoryID

		discordCategory := Category{
			ID:   categoryID,
			Name: discordExportedData.Channel.CategoryName,
		}

		_, ok := extractedData.Categories[categoryID]
		if !ok {
			extractedData.Categories[categoryID] = &discordCategory
		}

		extractedData.MessageCount = extractedData.MessageCount + discordExportedData.MessageCount
		extractedData.ExportedData = append(extractedData.ExportedData, &discordExportedData)

		if len(discordExportedData.Messages) > 0 {
			msgTime, err := time.Parse(discordTimestampLayout, discordExportedData.Messages[0].Timestamp)
			if err != nil {
				m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
				continue
			}

			if extractedData.OldestMessageTimestamp == 0 || int(msgTime.Unix()) <= extractedData.OldestMessageTimestamp {
				// Exported discord channel data already comes with `messages` being
				// sorted, starting with the oldest, so we can safely rely on the first
				// message
				extractedData.OldestMessageTimestamp = int(msgTime.Unix())
			}
		}
	}
	return extractedData, errors
}

func (m *ImportManager) ExtractDiscordChannelsAndCategories(filesToImport []string) ([]*Category, []*Channel, int, map[string]*ImportError) {

  var categories []*Category
  var channels []*Channel
  oldestMessageTimestamp := 0

	extractedData, errs := m.ExtractDiscordDataFromImportFiles(filesToImport)

	for _, category := range extractedData.Categories {
    categories = append(categories, category)
	}
	for _, export := range extractedData.ExportedData {
    channels = append(channels, &export.Channel)
	}
	if extractedData.OldestMessageTimestamp != 0 {
		oldestMessageTimestamp = extractedData.OldestMessageTimestamp
	}

	return categories, channels, oldestMessageTimestamp, errs
}

func (m *ImportManager) startPublishImportProgressInterval(c chan *ImportProgress, done chan struct{}) {

	var currentProgress *ImportProgress

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if currentProgress != nil {
					m.publishImportProgress(&Subscription{
            ImportProgress: currentProgress,
          })
				}
			case progressUpdate := <-c:
				currentProgress = progressUpdate
			case <-done:
				if currentProgress != nil {
					m.publishImportProgress(&Subscription{
            ImportProgress: currentProgress,
          })
				}
				return
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *ImportManager) InitializeImport(importTasks []*ImportTask, progressUpdates chan *ImportProgress, done chan struct{}) *ImportProgress {

  m.importTasks = importTasks

  taskTypes := make([]ImportTaskType, 0)
  for _, importTask := range importTasks {
    taskTypes = append(taskTypes, importTask.Type)
  }

  importProgress := &ImportProgress{}
  importProgress.Init(taskTypes)

  // importProgress.Init([]ImportTaskType{
  //   CommunityCreationTask,
  //   ChannelsCreationTask,
  //   ImportMessagesTask,
  //   DownloadAssetsTask,
  //   InitCommunityTask,
  // })

  m.startPublishImportProgressInterval(progressUpdates, done)
  // publish initial import progress
	progressUpdates <- importProgress

  return importProgress
}
