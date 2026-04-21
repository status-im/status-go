package wakuv2

func (w *Waku) PausableName() string { return "waku" }

func (w *Waku) Pause() error {
	w.MarkPaused()
	return nil
}

func (w *Waku) Resume() error {
	w.MarkResumed()
	return nil
}
