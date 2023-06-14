package collectibles

import (
	"database/sql"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseSuite))
}

type DatabaseSuite struct {
	suite.Suite

	db *Database
}

func (s *DatabaseSuite) addCommunityToken(db *sql.DB, token *communities.CommunityToken) error {
	_, err := db.Exec(`INSERT INTO community_tokens (community_id, address, type, name, symbol, description, supply,
		infinite_supply, transferable, remote_self_destruct, chain_id, deploy_state, image_base64, decimals) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, token.CommunityID, token.Address, token.TokenType, token.Name,
		token.Symbol, token.Description, token.Supply, token.InfiniteSupply, token.Transferable, token.RemoteSelfDestruct,
		token.ChainID, token.DeployState, token.Base64Image, token.Decimals)
	return err
}

func (s *DatabaseSuite) setupDatabase(db *sql.DB) error {
	token721 := &communities.CommunityToken{
		CommunityID:        "123",
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             123,
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        communities.InProgress,
		Base64Image:        "ABCD",
	}

	token20 := &communities.CommunityToken{
		CommunityID:        "345",
		TokenType:          protobuf.CommunityTokenType_ERC20,
		Address:            "0x345",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             345,
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            2,
		DeployState:        communities.Failed,
		Base64Image:        "QWERTY",
		Decimals:           21,
	}

	err := s.addCommunityToken(db, token721)
	if err != nil {
		return err
	}
	return s.addCommunityToken(db, token20)
}

func (s *DatabaseSuite) SetupTest() {
	s.db = nil

	dbPath, err := ioutil.TempFile("", "status-go-community-tokens-db-")
	s.NoError(err, "creating temp file for db")

	db, err := appdatabase.InitializeDB(dbPath.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.NoError(err, "creating sqlite db instance")

	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	s.db = &Database{db: db}

	err = s.setupDatabase(db)
	s.NoError(err, "setting up database")
}

func (s *DatabaseSuite) TestGetTokenType() {
	contractType, err := s.db.GetTokenType(1, "0x123")
	s.Require().NoError(err)
	s.Equal(contractType, protobuf.CommunityTokenType_ERC721)

	contractType, err = s.db.GetTokenType(2, "0x345")
	s.Require().NoError(err)
	s.Equal(contractType, protobuf.CommunityTokenType_ERC20)

	_, err = s.db.GetTokenType(10, "0x777")
	s.Require().Error(err)
}
