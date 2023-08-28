//go:build !gowaku_rln
// +build !gowaku_rln

package node

import "context"

func (w *WakuNode) RLNRelay() RLNRelay {
	return nil
}

func (w *WakuNode) setupRLNRelay() error {
	return nil
}

func (w *WakuNode) startRlnRelay(ctx context.Context) error {
	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	return nil
}
