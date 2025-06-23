package protocol

import (
	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) ExportBackup() ([]byte, error) {
	backup := &protobuf.Backup{} // TODO: fill me in more efficient way than waku backup
	return proto.Marshal(backup)
}

func (m *Messenger) ImportBackup(data []byte) error {
	var backup protobuf.Backup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}

	// TODO: process Backup
	return nil
}
