//go:build !gowaku_rln
// +build !gowaku_rln

package node

// RLNRelay is used to access any operation related to Waku RLN protocol
func (w *WakuNode) RLNRelay() RLNRelay {
	return nil
}

func (w *WakuNode) mountRlnRelay() error {
	return nil
}
