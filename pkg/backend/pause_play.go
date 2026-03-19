package backend

import (
	"fmt"

	"github.com/status-im/status-go/pkg/backend/node"
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

// LifecycleState returns the current backend lifecycle state.
func (b *StatusBackend) LifecycleState() AppLifecycleState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lifecycleState
}

// pauseLocked must be called with b.mu held.
func (b *StatusBackend) pauseLocked() error {
	if b.lifecycleState == AppLifecycleStopped {
		return nil
	}
	if b.lifecycleState == AppLifecyclePausedBackground {
		return nil
	}
	sn := b.statusNode

	if sn == nil || !sn.IsRunning() {
		b.lifecycleState = AppLifecycleStopped
		return nil
	}
	if messenger := b.currentMessenger(sn); messenger != nil {
		messenger.ToBackground()
	}
	if err := sn.PauseBackground(); err != nil {
		return fmt.Errorf("pause background: %w", err)
	}
	b.lifecycleState = AppLifecyclePausedBackground
	return nil
}

// resumeLocked must be called with b.mu held.
func (b *StatusBackend) resumeLocked() error {
	if b.lifecycleState == AppLifecycleStopped {
		return nil
	}
	if b.lifecycleState == AppLifecycleRunning {
		return nil
	}
	sn := b.statusNode

	if sn == nil || !sn.IsRunning() {
		b.lifecycleState = AppLifecycleStopped
		return nil
	}
	if messenger := b.currentMessenger(sn); messenger != nil {
		messenger.ToForeground()
	}

	b.lifecycleState = AppLifecycleResumingForeground
	if err := sn.ResumeForeground(); err != nil {
		return fmt.Errorf("resume foreground: %w", err)
	}
	b.lifecycleState = AppLifecycleRunning
	return nil
}

func (b *StatusBackend) currentMessenger(sn *node.StatusNode) *protocol.Messenger {
	if sn == nil || sn.WakuV2ExtService() == nil {
		return nil
	}
	return sn.WakuV2ExtService().Messenger()
}
