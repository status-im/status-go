package notificationserver

import (
	"errors"
)

var (
	ErrNoTarget = errors.New("notifier: no target was available")
)

// Notifier handles the notification delivery
type Notifier interface {
	// Notify includes a data payload in the notification & notifies
	// a list of devices
	Notify(devices []string, payload interface{}) error
}
