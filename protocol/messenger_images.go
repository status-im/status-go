package protocol

import "fmt"

func (m *Messenger) ImageServerURL() string {
	return fmt.Sprintf("https://localhost:%d/messages/", m.imageServer.Port)
}
