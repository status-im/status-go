package communities

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Persistence struct {
	db     *sql.DB
	logger *zap.Logger
}

func (p *Persistence) SaveCommunity(community *Community) error {
	id := community.ID()
	privateKey := community.PrivateKey()
	description, err := community.ToBytes()
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO communities_communities (id, private_key, description, joined, verified) VALUES (?, ?, ?,?,?)`, id, crypto.FromECDSA(privateKey), description, community.config.Joined, community.config.Verified)
	return err
}

func (p *Persistence) queryCommunities(query string) ([]*Community, error) {
	var response []*Community

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var publicKeyBytes, privateKeyBytes, descriptionBytes []byte
		var joined bool
		var verified bool
		err := rows.Scan(&publicKeyBytes, &privateKeyBytes, &descriptionBytes, &joined, &verified)
		if err != nil {
			return nil, err
		}

		org, err := unmarshalCommunityFromDB(publicKeyBytes, privateKeyBytes, descriptionBytes, joined, verified, p.logger)
		if err != nil {
			return nil, err
		}
		response = append(response, org)
	}

	return response, nil

}

func (p *Persistence) AllCommunities() ([]*Community, error) {
	query := `SELECT id, private_key, description,joined,verified FROM communities_communities`
	return p.queryCommunities(query)
}

func (p *Persistence) JoinedCommunities() ([]*Community, error) {
	query := `SELECT id, private_key, description,joined,verified FROM communities_communities WHERE joined`
	return p.queryCommunities(query)
}

func (p *Persistence) CreatedCommunities() ([]*Community, error) {
	query := `SELECT id, private_key, description,joined,verified FROM communities_communities WHERE private_key IS NOT NULL`
	return p.queryCommunities(query)
}

func (p *Persistence) GetByID(id []byte) (*Community, error) {
	var publicKeyBytes, privateKeyBytes, descriptionBytes []byte
	var joined bool
	var verified bool

	err := p.db.QueryRow(`SELECT id, private_key, description, joined,verified FROM communities_communities WHERE id = ?`, id).Scan(&publicKeyBytes, &privateKeyBytes, &descriptionBytes, &joined, &verified)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return unmarshalCommunityFromDB(publicKeyBytes, privateKeyBytes, descriptionBytes, joined, verified, p.logger)
}

func unmarshalCommunityFromDB(publicKeyBytes, privateKeyBytes, descriptionBytes []byte, joined, verified bool, logger *zap.Logger) (*Community, error) {

	var privateKey *ecdsa.PrivateKey
	var err error

	if privateKeyBytes != nil {
		privateKey, err = crypto.ToECDSA(privateKeyBytes)
		if err != nil {
			return nil, err
		}
	}
	metadata := &protobuf.ApplicationMetadataMessage{}

	err = proto.Unmarshal(descriptionBytes, metadata)
	if err != nil {
		return nil, err
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(metadata.Payload, description)
	if err != nil {
		return nil, err
	}

	id, err := crypto.DecompressPubkey(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	config := Config{
		PrivateKey:                    privateKey,
		CommunityDescription:          description,
		MarshaledCommunityDescription: descriptionBytes,
		Logger:                        logger,
		ID:                            id,
		Verified:                      verified,
		Joined:                        joined,
	}
	return New(config)
}
