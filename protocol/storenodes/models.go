package storenodes

import (
	"database/sql"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/mailservers"
)

// Storenode is a struct that represents a storenode, it is very closeley related to `mailservers.Mailserver`
type Storenode struct {
	CommunityID types.HexBytes `json:"community_id"`
	StorenodeID string         `json:"storenode_id"`
	Name        string         `json:"name"`
	Address     string         `json:"address"`
	Password    string         `json:"password,omitempty"`
	Fleet       string         `json:"fleet"`
	Version     uint           `json:"version"`
	Clock       uint64         `json:"-"` // used to sync
	Removed     bool           `json:"-"`
	DeletedAt   int64          `json:"-"`
}

type Storenodes []Storenode

func (m Storenode) nullablePassword() (val sql.NullString) {
	if m.Password != "" {
		val.String = m.Password
		val.Valid = true
	}
	return
}

func (m Storenodes) ToProtobuf() []*protobuf.Storenode {
	result := make([]*protobuf.Storenode, 0, len(m))
	for _, n := range m {

		result = append(result, &protobuf.Storenode{
			CommunityId: n.CommunityID,
			StorenodeId: n.StorenodeID,
			Name:        n.Name,
			Address:     n.Address,
			Password:    n.Password,
			Fleet:       n.Fleet,
			Removed:     n.Removed,
			DeletedAt:   n.DeletedAt,
		})
	}
	return result
}

func FromProtobuf(storenodes []*protobuf.Storenode, clock uint64) Storenodes {
	result := make(Storenodes, 0, len(storenodes))
	for _, s := range storenodes {
		result = append(result, Storenode{
			CommunityID: s.CommunityId,
			StorenodeID: s.StorenodeId,
			Name:        s.Name,
			Address:     s.Address,
			Password:    s.Password,
			Fleet:       s.Fleet,
			Removed:     s.Removed,
			DeletedAt:   s.DeletedAt,
			Clock:       clock,
		})
	}
	return result
}

func toMailserver(m Storenode) mailservers.Mailserver {
	return mailservers.Mailserver{
		ID:       m.StorenodeID,
		Name:     m.Name,
		Custom:   true,
		Address:  m.Address,
		Password: m.Password,
		Fleet:    m.Fleet,
		Version:  m.Version,
	}
}
