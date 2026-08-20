package backup

func (c *Controller) PausableName() string { return "backup" }

func (c *Controller) Pause() error {
	c.MarkPaused()
	return nil
}

func (c *Controller) Resume() error {
	c.MarkResumed()
	return nil
}
