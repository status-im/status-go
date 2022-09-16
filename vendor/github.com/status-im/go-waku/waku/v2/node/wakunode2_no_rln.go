//go:build !gowaku_rln
// +build !gowaku_rln

package node

func (w *WakuNode) RLNRelay() RLNRelay {
	return nil
}

func (w *WakuNode) mountRlnRelay() error {
	return nil
}

func (w *WakuNode) stopRlnRelay() error {
	return nil
}
