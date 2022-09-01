package discord

import (
	"errors"
	"fmt"
	"sync"

	"github.com/status-im/status-go/protocol/protobuf"
)

type ErrorCodeType uint

const (
	NoErrorType ErrorCodeType = iota
	WarningType
	ErrorType
)

var (
	ErrNoChannelData  = errors.New("No channels to import messages from")
	ErrNoMessageData  = errors.New("No messages to import")
	ErrMarshalMessage = errors.New("Couldn't marshal discord message")
)

type MessageType string

const (
	MessageTypeDefault MessageType = "Default"
	MessageTypeReply   MessageType = "Reply"
)

type ImportTask uint

const (
	CommunityCreationTask ImportTask = iota
	ChannelsCreationTask
	ImportMessagesTask
	DownloadAssetsTask
	InitCommunityTask
)

func (t ImportTask) String() string {
	switch t {
	case CommunityCreationTask:
		return "import.communityCreation"
	case ChannelsCreationTask:
		return "import.channelsCreation"
	case ImportMessagesTask:
		return "import.importMessages"
	case DownloadAssetsTask:
		return "import.downloadAssets"
	case InitCommunityTask:
		return "import.initializeCommunity"
	}
	return "unknown"
}

type Channel struct {
	ID           string `json:"id"`
	CategoryName string `json:"category"`
	CategoryID   string `json:"categoryId"`
	Name         string `json:"name"`
	Description  string `json:"topic"`
	FilePath     string `json:"filePath"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExportedData struct {
	Channel      Channel                    `json:"channel"`
	Messages     []*protobuf.DiscordMessage `json:"messages"`
	MessageCount int                        `json:"messageCount"`
}

type ExtractedData struct {
	Categories             map[string]*Category
	ExportedData           []*ExportedData
	OldestMessageTimestamp int
	MessageCount           int
}

type ImportError struct {
	// This code is used to distinguish between errors
	// that are considered "criticial" and those that are not.
	//
	// Critical errors are the ones that prevent the imported community
	// from functioning properly. For example, if the creation of the community
	// or its categories and channels fails, this is a critical error.
	//
	// Non-critical errors are the ones that would not prevent the imported
	// community from functioning. For example, if the channel data to be imported
	// has no messages, or is not parsable.
	Code    ErrorCodeType `json:"code"`
	Message string        `json:"message"`
}

func (d ImportError) Error() string {
	return fmt.Sprintf("%d: %s", d.Code, d.Message)
}

func Error(message string) *ImportError {
	return &ImportError{
		Message: message,
		Code:    ErrorType,
	}
}

func Warning(message string) *ImportError {
	return &ImportError{
		Message: message,
		Code:    WarningType,
	}
}

type ImportTaskProgress struct {
	Type     string         `json:"type"`
	Progress float32        `json:"progress"`
	Errors   []*ImportError `json:"errors"`
	Stopped  bool           `json:"stopped"`
}

type ImportTasks map[ImportTask]*ImportTaskProgress

type ImportProgress struct {
	CommunityID   string                `json:"communityId,omitempty"`
	CommunityName string                `json:"communityName"`
	Tasks         []*ImportTaskProgress `json:"tasks"`
	Progress      float32               `json:"progress"`
	ErrorsCount   uint                  `json:"errorsCount"`
	WarningsCount uint                  `json:"warningsCount"`
	Stopped       bool                  `json:"stopped"`
	m             sync.Mutex
}

func (progress *ImportProgress) Init(tasks []ImportTask) {
	progress.Progress = 0
	progress.Tasks = make([]*ImportTaskProgress, 0)
	for _, task := range tasks {
		progress.Tasks = append(progress.Tasks, &ImportTaskProgress{
			Type:     task.String(),
			Progress: 0,
			Errors:   []*ImportError{},
			Stopped:  false,
		})
	}
	progress.ErrorsCount = 0
	progress.WarningsCount = 0
	progress.Stopped = false
}

func (progress *ImportProgress) Stop() {
	progress.Stopped = true
}

func (progress *ImportProgress) AddTaskError(task ImportTask, err *ImportError) {
	progress.m.Lock()
	for i, t := range progress.Tasks {
		if t.Type == task.String() {
			errors := progress.Tasks[i].Errors
			progress.Tasks[i].Errors = append(errors, err)
		}
	}
	if err.Code > WarningType {
		progress.ErrorsCount++
		return
	}
	if err.Code > NoErrorType {
		progress.WarningsCount++
	}
	progress.m.Unlock()
}

func (progress *ImportProgress) StopTask(task ImportTask) {
	progress.m.Lock()
	for i, t := range progress.Tasks {
		if t.Type == task.String() {
			progress.Tasks[i].Stopped = true
		}
	}
	progress.Stop()
	progress.m.Unlock()
}

func (progress *ImportProgress) UpdateTaskProgress(task ImportTask, value float32) {
	progress.m.Lock()
	for i, t := range progress.Tasks {
		if t.Type == task.String() {
			progress.Tasks[i].Progress = value
		}
	}
	sum := float32(0)
	for _, t := range progress.Tasks {
		sum = sum + t.Progress
	}
	// Update total progress now that sub progress has changed
	progress.Progress = sum / float32(len(progress.Tasks))
	progress.m.Unlock()
}
