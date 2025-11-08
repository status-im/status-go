package linkpreview

import (
	"github.com/status-im/status-go/multiaccounts/settings"
)

//go:generate go tool mockgen -package=mock_persistence -source=persistence.go -destination=./mock/persistence.go

type Persistence interface {
	GetUnfurlingMode() (settings.URLUnfurlingModeType, error)
}
