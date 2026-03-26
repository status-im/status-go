package ipfs

func (d *Downloader) PausableName() string { return "ipfs" }

func (d *Downloader) Pause() error {
	d.MarkPaused()
	return nil
}

func (d *Downloader) Resume() error {
	d.MarkResumed()
	return nil
}
