package linkpreview

import (
	"github.com/status-im/status-go/multiaccounts/settings"
)

//go:generate go tool mockgen -package=mock_settings -source=settings.go -destination=./mock/settings.go

type Settings interface {
	GetUnfurlingMode() (settings.URLUnfurlingModeType, error)
}
