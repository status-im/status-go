//go:build !gowaku_rln
// +build !gowaku_rln

package node

import "context"

func (w *WakuNode) RLNRelay() RLNRelay {
	return nil
}

func (w *WakuNode) mountRlnRelay(ctx context.Context) error {
	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	return nil
}
