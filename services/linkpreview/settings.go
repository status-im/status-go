package linkpreview

import (
	"github.com/status-im/status-go/multiaccounts/settings"
)

//go:generate go tool mockgen -package=mock_settings -source=settings.go -destination=./mock/settings.go

//
// Option 1
//

type Settings interface {
	GetUnfurlingMode() (settings.URLUnfurlingModeType, error)
}

////
//// Option 2
////
//
//type Settings1 interface {
//	GetUnfurlingMode() (settings.URLUnfurlingModeType, error)
//	SaveUnfurlingMode(modeType settings.URLUnfurlingModeType) error
//}
//
////
//// Option 3
////
//
//type Settings2 interface {
//	GetUnfurlingMode() <-chan settings.URLUnfurlingModeType
//}
//
////
//// Option 4
////
//
//type Service2 struct {
//
//}
//
//func (s *Service2) SetUnfurlingMode(mode settings.URLUnfurlingModeType) {
//
//}
