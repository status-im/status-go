package signal

const (
	// ReEncryptionStarted is sent when db reencryption was started.
	ReEncryptionStarted = "db.reEncryption.started"
	// ReEncryptionFinished is sent when db reencryption was finished.
	ReEncryptionFinished = "db.reEncryption.finished"
)

// Send db.reencryption.started signal.
func SendReEncryptionStarted() {
	CurrentSession().SendReEncryptionStarted()
}

// SendReEncryptionStarted sends in the session s.
func (s Session) SendReEncryptionStarted() {
	s.send(ReEncryptionStarted, nil)
}

// Send db.reencryption.finished signal.
func SendReEncryptionFinished() {
	CurrentSession().SendReEncryptionFinished()
}

// SendReEncryptionFinished sends in the session s.
func (s Session) SendReEncryptionFinished() {
	s.send(ReEncryptionFinished, nil)
}
