package backend

import (
	"fmt"

	"github.com/status-im/status-go/protocol"
)

// AppLifecycleState is the backend runtime state for pause/play control.
type AppLifecycleState string

const (
	AppLifecycleStopped            AppLifecycleState = "stopped"
	AppLifecycleRunning            AppLifecycleState = "running"
	AppLifecyclePausedBackground   AppLifecycleState = "paused_background"
	AppLifecycleResumingForeground AppLifecycleState = "resuming_foreground"
)

func (s AppLifecycleState) String() string { return string(s) }

func (b *StatusBackend) LifecycleState() AppLifecycleState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lifecycleState
}

func (b *StatusBackend) pauseLocked() error {
	if b.lifecycleState == AppLifecycleStopped {
		return nil
	}
	if b.lifecycleState == AppLifecyclePausedBackground {
		return nil
	}
	if b.statusNode == nil || !b.statusNode.IsRunning() {
		b.lifecycleState = AppLifecycleStopped
		return nil
	}
	if messenger := b.currentMessengerLocked(); messenger != nil {
		messenger.ToBackground()
	}
	if err := b.statusNode.PauseBackground(); err != nil {
		return fmt.Errorf("pause background: %w", err)
	}
	b.lifecycleState = AppLifecyclePausedBackground
	return nil
}

func (b *StatusBackend) resumeLocked() error {
	if b.lifecycleState == AppLifecycleStopped {
		return nil
	}
	if b.lifecycleState == AppLifecycleRunning {
		return nil
	}
	if b.statusNode == nil || !b.statusNode.IsRunning() {
		b.lifecycleState = AppLifecycleStopped
		return nil
	}
	if messenger := b.currentMessengerLocked(); messenger != nil {
		messenger.ToForeground()
	}

	b.lifecycleState = AppLifecycleResumingForeground
	if err := b.statusNode.ResumeForeground(); err != nil {
		return fmt.Errorf("resume foreground: %w", err)
	}
	b.lifecycleState = AppLifecycleRunning
	return nil
}

func (b *StatusBackend) currentMessengerLocked() *protocol.Messenger {
	if b.statusNode == nil || b.statusNode.WakuV2ExtService() == nil {
		return nil
	}
	return b.statusNode.WakuV2ExtService().Messenger()
}
