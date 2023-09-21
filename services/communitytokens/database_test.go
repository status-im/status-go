package communitytokens

import (
	"database/sql"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/t/helpers"
)

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseSuite))
}

type DatabaseSuite struct {
	suite.Suite

	db *Database
}

func (s *DatabaseSuite) addCommunityToken(db *sql.DB, token *token.CommunityToken) error {
	_, err := db.Exec(`INSERT INTO community_tokens (community_id, address, type, name, symbol, description, supply_str,
		infinite_supply, transferable, remote_self_destruct, chain_id, deploy_state, image_base64, decimals, deployer, privileges_level) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, token.CommunityID, token.Address, token.TokenType, token.Name,
		token.Symbol, token.Description, token.Supply.String(), token.InfiniteSupply, token.Transferable, token.RemoteSelfDestruct,
		token.ChainID, token.DeployState, token.Base64Image, token.Decimals, token.Deployer, token.PrivilegesLevel)
	return err
}

func (s *DatabaseSuite) setupDatabase(db *sql.DB) error {
	token721 := &token.CommunityToken{
		CommunityID:        "123",
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(123)},
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        token.InProgress,
		Base64Image:        "ABCD",
		Deployer:           "0xDEP1",
		PrivilegesLevel:    token.OwnerLevel,
	}

	token20 := &token.CommunityToken{
		CommunityID:        "345",
		TokenType:          protobuf.CommunityTokenType_ERC20,
		Address:            "0x345",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(345)},
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            2,
		DeployState:        token.Failed,
		Base64Image:        "QWERTY",
		Decimals:           21,
		Deployer:           "0xDEP2",
		PrivilegesLevel:    token.CommunityLevel,
	}

	err := s.addCommunityToken(db, token721)
	if err != nil {
		return err
	}
	return s.addCommunityToken(db, token20)
}

func (s *DatabaseSuite) SetupTest() {
	s.db = nil

	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
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

func (s *DatabaseSuite) TestGetPrivilegesLevel() {
	privLevel, err := s.db.GetTokenPrivilegesLevel(1, "0x123")
	s.Require().NoError(err)
	s.Equal(privLevel, token.OwnerLevel)

	privLevel, err = s.db.GetTokenPrivilegesLevel(2, "0x345")
	s.Require().NoError(err)
	s.Equal(privLevel, token.CommunityLevel)

	_, err = s.db.GetTokenType(10, "0x777")
	s.Require().Error(err)
}
