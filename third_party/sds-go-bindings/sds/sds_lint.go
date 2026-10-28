//go:build lint

package sds

import (
	"errors"

	"go.uber.org/zap"
)

// This file provides lint-only stubs that avoid requiring libsds.h/cgo
// so linters can analyze this package without native dependencies.

// ErrLintBuild indicates a stubbed, lint-only build without native libsds.
var ErrLintBuild = errors.New("sds: lint-only build stub: native libsds not linked")

// NewReliabilityManager returns an error in lint builds.
func NewReliabilityManager(logger *zap.Logger, persistence ...PersistenceCallbacks) (*ReliabilityManager, error) {
	if len(persistence) > 1 {
		return nil, errors.New("at most one SDS persistence callback set is allowed")
	}
	if len(persistence) == 1 {
		if err := persistence[0].validate(); err != nil {
			return nil, err
		}
	}
	return nil, ErrLintBuild
}

// Cleanup returns an error in lint builds.
func (rm *ReliabilityManager) Cleanup() error { return ErrLintBuild }

// Reset returns an error in lint builds.
func (rm *ReliabilityManager) Reset() error { return ErrLintBuild }

// WrapOutgoingMessage returns an error in lint builds.
func (rm *ReliabilityManager) WrapOutgoingMessage(message []byte, messageId MessageID, channelId string) ([]byte, error) {
	return nil, ErrLintBuild
}

// UnwrapReceivedMessage returns an error in lint builds.
func (rm *ReliabilityManager) UnwrapReceivedMessage(message []byte) (*UnwrappedMessage, error) {
	return nil, ErrLintBuild
}

// MarkDependenciesMet returns an error in lint builds.
func (rm *ReliabilityManager) MarkDependenciesMet(messageIDs []MessageID, channelId string) error {
	return ErrLintBuild
}

// StartPeriodicTasks returns an error in lint builds.
func (rm *ReliabilityManager) StartPeriodicTasks() error { return ErrLintBuild }
