package communities

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/images"
	testutils "github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/internal/testutils/fake"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities/archive"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	community_token "github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	v "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

type ManagerSuite struct {
	suite.Suite
	manager        *Manager
	archiveManager archive.ArchiveService
}

func buildTorrentConfig() *params.TorrentConfig {
	return &params.TorrentConfig{
		Enabled:    true,
		DataDir:    os.TempDir() + "/archivedata",
		TorrentDir: os.TempDir() + "/torrents",
		Port:       0,
	}
}

func (s *ManagerSuite) buildManagers(ownerVerifier OwnerVerifier) (*Manager, archive.ArchiveService) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.Require().NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := NewManager(key, "", db, logger, nil, ownerVerifier, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	amc := &archivetypes.ArchiveManagerConfig{
		TorrentConfig: buildTorrentConfig(),
		Logger:        logger,
		Persistence:   m.GetPersistence(),
		Messaging:     nil,
		Identity:      key,
		Publisher:     m,
	}
	t := archive.NewArchiveManager(amc)
	s.Require().NoError(err)

	return m, t
}

func (s *ManagerSuite) SetupTest() {
	m, t := s.buildManagers(nil)
	SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveManager = t
}

func intToBig(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}

func uintToDecBig(n uint64) *bigint.BigInt {
	return &bigint.BigInt{Int: big.NewInt(int64(n))}
}

func tokenBalance(tokenID uint64, balance uint64) thirdparty.TokenBalance {
	return thirdparty.TokenBalance{
		TokenID: uintToDecBig(tokenID),
		Balance: uintToDecBig(balance),
	}
}

type testCollectiblesManager struct {
	response map[uint64]map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress
}

func (m *testCollectiblesManager) setResponse(chainID uint64, walletAddress gethcommon.Address, contractAddress gethcommon.Address, balances []thirdparty.TokenBalance) {
	if m.response == nil {
		m.response = make(map[uint64]map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
	}
	if m.response[chainID] == nil {
		m.response[chainID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
	}
	if m.response[chainID][walletAddress] == nil {
		m.response[chainID][walletAddress] = make(thirdparty.TokenBalancesPerContractAddress)
	}

	m.response[chainID][walletAddress][contractAddress] = balances
}

func (m *testCollectiblesManager) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	return m.response[uint64(chainID)][ownerAddress], nil
}

func (m *testCollectiblesManager) GetCollectibleOwnership(id thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error) {
	return nil, errors.New("GetCollectibleOwnership is not implemented for testCollectiblesManager")
}

func (m *testCollectiblesManager) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletCommon.ChainID, contractAddress gethcommon.Address) (*thirdparty.CollectibleContractOwnership, error) {
	ret := &thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          []thirdparty.CollectibleOwner{},
	}

	balancesPerOwner, ok := m.response[uint64(chainID)]
	if !ok {
		return ret, nil
	}

	for ownerAddress, collectibles := range balancesPerOwner {
		for collectibleAddress, balances := range collectibles {
			if collectibleAddress == contractAddress {
				ret.Owners = append(ret.Owners, thirdparty.CollectibleOwner{
					OwnerAddress:  ownerAddress,
					TokenBalances: balances,
				})
				break
			}
		}
	}

	return ret, nil
}

func (m *testCollectiblesManager) FetchCachedBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	return m.response[uint64(chainID)][ownerAddress], nil
}

type testTokenBalanceManager struct {
	response map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big
}

func (m *testTokenBalanceManager) setResponse(chainID uint64, walletAddress, tokenAddress gethcommon.Address, balance int64) {

	if m.response == nil {
		m.response = make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	}

	if m.response[chainID] == nil {
		m.response[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	}

	if m.response[chainID][walletAddress] == nil {
		m.response[chainID][walletAddress] = make(map[gethcommon.Address]*hexutil.Big)
	}

	m.response[chainID][walletAddress][tokenAddress] = intToBig(balance)

}

type testNetworkManager struct {
}

func (m *testNetworkManager) GetAllChainIDs() ([]uint64, error) {
	return []uint64{5}, nil
}

func (m *testTokenBalanceManager) GetBalancesByChain(ctx context.Context, accounts []gethcommon.Address, tokens []*tokentypes.Token) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	return m.response, nil
}

func (m *testTokenBalanceManager) GetCachedBalancesByChain(ctx context.Context, accounts []gethcommon.Address, tokens []*tokentypes.Token) (BalancesByChain, error) {
	return m.response, nil
}

type testTokenManager struct {
}

func (m *testTokenManager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) (*tokentypes.Token, error) {
	return nil, nil
}

func (s *ManagerSuite) setupManagerForTokenPermissions() (*Manager, *testCollectiblesManager, *testTokenBalanceManager) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)

	cm := &testCollectiblesManager{}
	tm := &testTokenManager{}
	tbm := &testTokenBalanceManager{}
	nm := &testNetworkManager{}

	options := []ManagerOption{
		WithCollectiblesManager(cm),
		WithTokenManager(tm),
		WithTokenBalanceManager(tbm),
		WithNetworkManager(nm),
	}

	m, err := NewManager(key, "", db, nil, nil, nil, nil, &TimeSourceStub{}, nil, nil, options...)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	return m, cm, tbm
}

func (s *ManagerSuite) TestRetrieveTokens() {
	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var permissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: tokenCriteria,
			},
		},
	}

	preParsedPermissions := preParsedCommunityPermissionsData(permissions)

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}
	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err := m.PermissionChecker.CheckPermissions(preParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Satisfied)

	// Set response to 0
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err = m.PermissionChecker.CheckPermissions(preParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().False(resp.Satisfied)
}

func (s *ManagerSuite) TestRetrieveCollectibles() {
	m, cm, _ := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"

	tokenID := uint64(10)
	var tokenBalances []thirdparty.TokenBalance

	var tokenCriteria = []*protobuf.TokenCriteria{
		{
			ContractAddresses: contractAddresses,
			TokenIds:          []uint64{tokenID},
			Type:              protobuf.CommunityTokenType_ERC721,
			AmountInWei:       "1",
		},
	}

	var permissions = []*CommunityTokenPermission{
		{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: tokenCriteria,
			},
		},
	}

	preParsedPermissions := preParsedCommunityPermissionsData(permissions)

	accountChainIDsCombination := []*AccountChainIDsCombination{
		{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	// Set response to exactly the right one
	tokenBalances = []thirdparty.TokenBalance{tokenBalance(tokenID, 1)}
	cm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), tokenBalances)
	resp, err := m.PermissionChecker.CheckPermissions(preParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Satisfied)

	// Set balances to 0
	tokenBalances = []thirdparty.TokenBalance{}
	cm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), tokenBalances)
	resp, err = m.PermissionChecker.CheckPermissions(preParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().False(resp.Satisfied)
}

func (s *ManagerSuite) TestCreateCommunity() {
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "token membership description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	actualCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[0].ID()) {
		actualCommunity = communities[0]
	}

	s.Require().Equal(community.ID(), actualCommunity.ID())
	s.Require().Equal(community.PrivateKey(), actualCommunity.PrivateKey())
	s.Require().True(community.IsControlNode())
	s.Require().True(proto.Equal(community.config.CommunityDescription, actualCommunity.config.CommunityDescription))
}

func (s *ManagerSuite) TestCreateCommunity_WithBanner() {
	// Generate test image bigger than BannerDim
	testImage := image.NewRGBA(image.Rect(0, 0, 20, 10))

	tmpTestFilePath := s.T().TempDir() + "/test.png"
	file, err := os.Create(tmpTestFilePath)
	s.NoError(err)
	defer file.Close()

	err = png.Encode(file, testImage)
	s.Require().NoError(err)

	request := &requests.CreateCommunity{
		Name:        "with_banner",
		Description: "community with banner ",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Banner: images.CroppedImage{
			ImagePath: tmpTestFilePath,
			X:         1,
			Y:         1,
			Width:     10,
			Height:    5,
		},
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)
	s.Require().Equal(len(community.config.CommunityDescription.Identity.Images), 1)
	testIdentityImage, isMapContainsKey := community.config.CommunityDescription.Identity.Images[images.BannerIdentityName]
	s.Require().True(isMapContainsKey)
	s.Require().Positive(len(testIdentityImage.Payload))
}

func (s *ManagerSuite) TestEditCommunity() {
	image1Path, err := fake.SaveImage(s.T().TempDir(), 8, 8)
	s.Require().NoError(err)
	image2Path, err := fake.SaveImage(s.T().TempDir(), 8, 8)
	s.Require().NoError(err)

	//create community
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Image:       image1Path,
		ImageAx:     1,
		ImageAy:     1,
		ImageBx:     7,
		ImageBy:     7,
	}

	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	update := &requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        "statusEdited",
			Description: "status community description edited",
			Image:       image2Path,
			ImageAx:     4,
			ImageAy:     4,
			ImageBx:     8,
			ImageBy:     8,
		},
	}

	updatedCommunity, err := s.manager.EditCommunity(update)
	s.Require().NoError(err)
	s.Require().NotNil(updatedCommunity)
	// Make sure the version of the image got updated with the new image
	communityImageVersion, ok := s.manager.communityImageVersions[community.IDString()]
	s.Require().True(ok)
	s.Require().Equal(uint32(1), communityImageVersion)

	//ensure updated community successfully stored
	communities, err := s.manager.All()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	storedCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[0].ID()) {
		storedCommunity = communities[0]
	}

	s.Require().Equal(updatedCommunity.ID(), storedCommunity.ID())
	s.Require().Equal(updatedCommunity.PrivateKey(), storedCommunity.PrivateKey())
	s.Require().Equal(update.CreateCommunity.Name, storedCommunity.config.CommunityDescription.Identity.DisplayName)
	s.Require().Equal(update.CreateCommunity.Description, storedCommunity.config.CommunityDescription.Identity.Description)
}

func (s *ManagerSuite) TestGetControlledCommunitiesChatIDs() {
	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)
	s.Require().NotNil(community)

	controlledChatIDs, err := s.manager.GetOwnedCommunitiesChatIDs()

	s.Require().NoError(err)
	s.Require().Len(controlledChatIDs, 1)
}

func (s *ManagerSuite) TestCheckChannelPermissions_NoPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var viewOnlyPermissions = make([]*CommunityTokenPermission, 0)
	var viewAndPostPermissions = make([]*CommunityTokenPermission, 0)
	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Both viewOnly and viewAndPost permissions are expected to be satisfied
	// because we call `checkChannelPermissions()` with no permissions to check
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().True(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewOnlyPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewOnlyPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: tokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	var viewAndPostPermissions = make([]*CommunityTokenPermission, 0)

	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().False(resp.ViewOnlyPermissions.Satisfied)
	// if viewOnly permissions are not satisfied then viewAndPost
	// permissions shouldn't be satisfied either
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)

	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err = m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewAndPostPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewAndPostPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: tokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	var viewOnlyPermissions = make([]*CommunityTokenPermission, 0)

	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
	// viewOnly permissions are flagged as not satisfied because we have no viewOnly
	// permissions on this channel and the viewAndPost permission is not satisfied either
	s.Require().False(resp.ViewOnlyPermissions.Satisfied)

	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err = m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().True(resp.ViewAndPostPermissions.Satisfied)
	// if viewAndPost is satisfied then viewOnly should be automatically satisfied
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewAndPostPermissionsCombination() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var viewOnlyTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewOnlyPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewOnlyTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	testContractAddresses := make(map[uint64]string)
	testContractAddresses[chainID] = "0x123"

	// Set up token criteria that won't be satisfied
	var viewAndPostTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: testContractAddresses,
			Symbol:            "TEST",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "TEST token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewAndPostPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewAndPostTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	// Set response for viewOnly permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	// Set resopnse for viewAndPost permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(testContractAddresses[chainID]), 0)

	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	resp, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// viewOnly permission should be satisfied, even though viewAndPost is not satisfied
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
}

// Same as the one above, but reversed where the View permission is not satisfied, but the view and post is
func (s *ManagerSuite) TestCheckChannelPermissions_ViewAndPostPermissionsCombination2() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var viewOnlyTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewOnlyPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewOnlyTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	testContractAddresses := make(map[uint64]string)
	testContractAddresses[chainID] = "0x123"

	// Set up token criteria that won't be satisfied
	var viewAndPostTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: testContractAddresses,
			Symbol:            "TEST",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "TEST token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewAndPostPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewAndPostTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	// Set response for viewOnly permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	// Set resopnse for viewAndPost permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(testContractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))

	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	resp, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Both permissions should be satisfied, even though view is not satisfied
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().True(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckAllChannelsPermissions_EmptyPermissions() {

	m, _, _ := s.setupManagerForTokenPermissions()

	createRequest := &requests.CreateCommunity{
		Name:        "channel permission community",
		Description: "some description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := m.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// create community chats
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat1",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err := m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	var chatID string
	for cid := range changes.ChatsAdded {
		chatID = community.IDString() + cid
	}

	response, err := m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Channels, 1)
	// we expect both, viewOnly and viewAndPost permissions to be satisfied
	// as there aren't any permissions on this channel
	s.Require().True(response.Channels[chatID].ViewOnlyPermissions.Satisfied)
	s.Require().True(response.Channels[chatID].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID].ViewOnlyPermissions.Permissions, 0)
	s.Require().Len(response.Channels[chatID].ViewAndPostPermissions.Permissions, 0)
}

func (s *ManagerSuite) TestCheckAllChannelsPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chatID1 string
	var chatID2 string

	// create community
	createRequest := &requests.CreateCommunity{
		Name:        "channel permission community",
		Description: "some description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := m.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// create first community chat
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat1",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err := m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	for chatID := range changes.ChatsAdded {
		chatID1 = community.IDString() + chatID
	}

	// create second community chat
	chat = &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat2",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err = m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	for chatID := range changes.ChatsAdded {
		chatID2 = community.IDString() + chatID
	}

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			AmountInWei:       "1000000000000000000",
			Decimals:          decimals,
		},
	}

	// create view only permission
	viewOnlyPermission := &requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: tokenCriteria,
		ChatIds:       []string{chatID1, chatID2},
	}

	_, changes, err = m.CreateCommunityTokenPermission(viewOnlyPermission)
	s.Require().NoError(err)

	var viewOnlyPermissionID string
	for permissionID := range changes.TokenPermissionsAdded {
		viewOnlyPermissionID = permissionID
	}

	response, err := m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// we've added to chats to the community, so there should be 2 items
	s.Require().Len(response.Channels, 2)

	// viewOnly permissions should not be satisfied because the account doesn't
	// have the necessary funds

	// channel1
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	// channel2
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	// viewAndPost permissions are flagged as not satisfied either because
	// viewOnly permission is not satisfied and there are no viewAndPost permissions

	// channel1
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 0)

	// channel2
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 0)

	// now change balance such that viewOnly permission should be satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// viewOnly permissions should be satisfied for both channels while
	// viewAndPost permissions should not be satisfied (as there aren't any)

	// channel1
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 0)

	// channel2
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 0)

	// next, create viewAndPost permission
	// create view only permission
	viewAndPostPermission := &requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		TokenCriteria: tokenCriteria,
		ChatIds:       []string{chatID1, chatID2},
	}

	_, changes, err = m.CreateCommunityTokenPermission(viewAndPostPermission)
	s.Require().NoError(err)

	var viewAndPostPermissionID string
	for permissionID := range changes.TokenPermissionsAdded {
		viewAndPostPermissionID = permissionID
	}

	// now change balance such that viewAndPost permission is not satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, viewOnly and viewAndPost permissions exist on channel1 and channel2
	// but shouldn't be satisfied

	// channel1
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// channel2
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// now change balance such that both, viewOnly and viewAndPost permission, are satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, viewOnly and viewAndPost permissions exist on channel1 and channel2
	// and are satisfied

	// channel1
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// channel2
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// next, delete viewOnly permission so we can check the viewAndPost permission-only case
	deleteViewOnlyPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: viewOnlyPermissionID,
	}
	_, _, err = m.DeleteCommunityTokenPermission(deleteViewOnlyPermission)
	s.Require().NoError(err)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, channel1 and channel2 now have viewAndPost only permissions that should
	// be satisfied, there's no viewOnly permission anymore the response should mark it
	// as satisfied as well

	// channel1
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 0)

	// channel2
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 0)

	// now change balance such that viewAndPost permission is no longer satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// because viewAndPost permission is not satisfied and there are no viewOnly permissions
	// on the channels, the response should mark the viewOnly permissions as not satisfied as well

	// channel1
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 0)

	// channel2
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 0)
}

func buildMessage(timestamp time.Time, topic types.ContentTopic, hash []byte) types.ReceivedMessage {
	message := types.ReceivedMessage{
		Sig:       []byte{1},
		Timestamp: uint32(timestamp.Unix()),
		Topic:     topic,
		Payload:   []byte{1},
		Padding:   []byte{1},
		Hash:      hash,
	}
	return message
}

func (s *ManagerSuite) buildCommunityWithChat() (*Community, string, error) {
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	if err != nil {
		return nil, "", err
	}
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "added-chat",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}
	changes, err := s.manager.CreateChat(community.ID(), chat, true, "")
	if err != nil {
		return nil, "", err
	}

	chatID := ""
	for cID := range changes.ChatsAdded {
		chatID = cID
		break
	}
	return community, chatID, nil
}

type testOwnerVerifier struct {
	called    int
	ownersMap map[string]string
	// err simulates a transport-class failure of the on-chain owner lookup
	// (RPC timeout, dead proxy, rate limit). When set, SafeGetSignerPubKey
	// fails the way a broken wallet RPC stack does, without rejecting the
	// community as invalid.
	err error
}

func (t *testOwnerVerifier) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	t.called++
	if t.err != nil {
		return "", t.err
	}
	return t.ownersMap[communityID], nil
}

func (s *ManagerSuite) TestCommunityQueue() {

	owner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m, _ := s.buildManagers(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = crypto.PubkeyToHex(&owner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, owner)
	s.Require().NoError(err)

	// Create a signer, that is not the owner
	notTheOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&notTheOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times
	s.Require().Equal(1, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)
}

// TestHighestQueuedValidationClock verifies the queue read the store-node pager
// uses to decide it may stop paging (issue #21470-hf): once a community's
// description is queued for owner validation, HighestQueuedValidationClock
// reports the greatest queued clock so the pager can compare it against the
// clock it already holds. With nothing queued (or for an unrelated community) it
// reports 0, so the pager keeps paging as before.
func (s *ManagerSuite) TestHighestQueuedValidationClock() {
	verifier := &testOwnerVerifier{}
	m, _ := s.buildManagers(verifier)

	communityID := []byte{0x01, 0x02, 0x03}

	clock, err := m.HighestQueuedValidationClock(communityID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), clock, "nothing queued yet")

	// Queue two descriptions with different clocks, both already due for
	// validation (validate_at in the past).
	s.Require().NoError(m.persistence.SaveCommunityToValidate(communityToValidate{
		id: communityID, clock: 5, payload: []byte("a"), validateAt: 1, signer: []byte("s")}))
	s.Require().NoError(m.persistence.SaveCommunityToValidate(communityToValidate{
		id: communityID, clock: 9, payload: []byte("b"), validateAt: 1, signer: []byte("s")}))

	clock, err = m.HighestQueuedValidationClock(communityID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(9), clock, "highest queued clock")

	clock, err = m.HighestQueuedValidationClock([]byte{0xaa, 0xbb})
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), clock, "unrelated community is unaffected")
}

// TestCommunityQueueRetriesTransportError verifies the core correctness fix
// (issue #21470-hf): a transport-class failure of owner verification (RPC
// timeout / dead proxy / rate limit) must NOT reject the community and must NOT
// clear the queue — the description stays queued so verification is retried
// out-of-band, and succeeds once the RPC recovers, publishing
// TokenCommunityValidated. This decouples verification retries from store-node
// paging: the pager stops once the description is queued, and recovery happens
// here rather than by fetching more pages.
func (s *ManagerSuite) TestCommunityQueueRetriesTransportError() {
	owner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m, _ := s.buildManagers(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = crypto.PubkeyToHex(&owner.PublicKey)

	description := community.config.CommunityDescription
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{2: "some-address"},
			},
		}}
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)
	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, owner)
	s.Require().NoError(err)

	notTheOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// The description arrives from a non-owner signer, so it is queued for
	// on-chain owner verification instead of being applied immediately.
	response, err := m.HandleCommunityDescriptionMessage(&notTheOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)
	s.Require().Nil(response, "must be queued, not applied")

	// The queued description is now in hand: the pager can see it and stop.
	queuedClock, err := m.HighestQueuedValidationClock(community.ID())
	s.Require().NoError(err)
	s.Require().Greater(queuedClock, uint64(0))

	// Transport failure during verification: the community must stay queued and
	// nothing may be published.
	subscription := m.Subscribe()
	verifier.err = errors.New("proxy dead: connection refused")

	resp, err := m.ValidateCommunityByID(community.ID())
	s.Require().NoError(err, "transport failure must not surface as a validation error")
	s.Require().Nil(resp)

	queued, err := m.persistence.getCommunityToValidateByID(community.ID())
	s.Require().NoError(err)
	s.Require().NotEmpty(queued, "transport-failed validation must remain queued for retry")

	select {
	case event := <-subscription:
		s.Require().Nil(event.TokenCommunityValidated, "must not publish on transport failure")
	default:
	}

	// Recovery: the RPC works on the next attempt → validated, published, queue
	// cleared.
	verifier.err = nil
	resp, err = m.ValidateCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(resp, "validation must succeed once the RPC recovers")

	published := false
	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("TokenCommunityValidated not published after recovery")
		}
	}

	queued, err = m.persistence.getCommunityToValidateByID(community.ID())
	s.Require().NoError(err)
	s.Require().Empty(queued, "successful validation must clear the queue")
}

// 1) We create a community
// 2) We have 2 owners, but only new owner is returned by the contract
// 3) We receive the old owner community first
// 4) We receive the new owner community second
// 5) We start the queue
// 6) We should only process 4, and ignore anything else if that is successful, as that's the most recent

func (s *ManagerSuite) TestCommunityQueueMultipleDifferentSigners() {

	newOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	oldOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m, _ := s.buildManagers(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = crypto.PubkeyToHex(&newOwner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	// We nil owner verifier so that messages won't be processed
	m.ownerVerifier = nil

	// Send message from old owner first

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, oldOwner)
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&oldOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	// Send message from new owner now

	community.config.CommunityDescription.Clock++

	clock2 := community.config.CommunityDescription.Clock

	payload, err = community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, newOwner)
	s.Require().NoError(err)

	response, err = m.HandleCommunityDescriptionMessage(&newOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	count, err := m.persistence.getCommunitiesToValidateCount()
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().NotNil(communitiesToValidate)
	s.Require().NotNil(communitiesToValidate[community.IDString()])
	s.Require().Len(communitiesToValidate[community.IDString()], 2)

	// We set owner verifier so that we start processing the queue
	m.ownerVerifier = verifier

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times, since we should be checking newest first
	s.Require().Equal(1, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err = m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)

	// Check clock of community is of the last community description
	fetchedCommunity, err := m.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(clock2, fetchedCommunity.config.CommunityDescription.Clock)

}

// 1) We create a community
// 2) We have 2 owners, but only old owner is returned by the contract
// 3) We receive the old owner community first
// 4) We receive the new owner community second (that could be a malicious user)
// 5) We start the queue
// 6) We should process both, but ignore the last community description

func (s *ManagerSuite) TestCommunityQueueMultipleDifferentSignersIgnoreIfNotReturned() {

	newOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	oldOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m, _ := s.buildManagers(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = crypto.PubkeyToHex(&oldOwner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	// We nil owner verifier so that messages won't be processed
	m.ownerVerifier = nil

	clock1 := community.config.CommunityDescription.Clock
	// Send message from old owner first

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, oldOwner)
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&oldOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	// Send message from new owner now

	community.config.CommunityDescription.Clock++

	payload, err = community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, newOwner)
	s.Require().NoError(err)

	response, err = m.HandleCommunityDescriptionMessage(&newOwner.PublicKey, description, payload, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	count, err := m.persistence.getCommunitiesToValidateCount()
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().NotNil(communitiesToValidate)
	s.Require().NotNil(communitiesToValidate[community.IDString()])
	s.Require().Len(communitiesToValidate[community.IDString()], 2)

	// We set owner verifier so that we start processing the queue
	m.ownerVerifier = verifier

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times, since we should be checking newest first
	s.Require().Equal(2, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err = m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)

	// Check clock of community is of the first community description
	fetchedCommunity, err := m.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(clock1, fetchedCommunity.config.CommunityDescription.Clock)
}

func (s *ManagerSuite) TestFillMissingCommunityTokens() {
	// Create community
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "token membership description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)
	s.Require().Len(community.CommunityTokensMetadata(), 0)

	// Create community token but without adding to the description
	token := community_token.CommunityToken{
		TokenType:          protobuf.CommunityTokenType_ERC721,
		CommunityID:        community.IDString(),
		Address:            "0x001",
		Name:               "TestTok",
		Symbol:             "TST",
		Description:        "Desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(0)},
		InfiniteSupply:     true,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        community_token.Deployed,
		Base64Image:        "",
		Decimals:           18,
		Deployer:           "0x0002",
		PrivilegesLevel:    community_token.CommunityLevel,
	}

	err = s.manager.persistence.AddCommunityToken(&token)
	s.Require().NoError(err)

	// Fill community with missing token
	err = s.manager.fillMissingCommunityTokens()
	s.Require().NoError(err)

	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.CommunityTokensMetadata(), 1)
}

func (s *ManagerSuite) TestDetermineChannelsForHRKeysRequest() {
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "token membership description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	channel := &protobuf.CommunityChat{
		Members: map[string]*protobuf.CommunityMember{
			crypto.PubkeyToHex(&s.manager.identity.PublicKey): {},
		},
	}

	description := community.config.CommunityDescription
	description.Chats = map[string]*protobuf.CommunityChat{}
	description.Chats["channel-id"] = channel

	// Simulate channel encrypted
	_, err = community.UpsertTokenPermission(&protobuf.CommunityTokenPermission{
		ChatIds: []string{ChatID(community.IDString(), "channel-id")},
	})
	s.Require().NoError(err)

	err = generateBloomFiltersForChannels(description, s.manager.identity)
	s.Require().NoError(err)

	now := int64(1)
	tenMinutes := int64(10 * 60 * 1000)

	// Member does not have missing encryption keys
	channels, err := s.manager.determineChannelsForHRKeysRequest(community, now)
	s.Require().NoError(err)
	s.Require().Empty(channels)

	// Simulate missing encryption key
	channel.Members = map[string]*protobuf.CommunityMember{}

	// Channel without prior request should be returned
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now)
	s.Require().NoError(err)
	s.Require().Len(channels, 1)
	s.Require().Equal("channel-id", channels[0])

	// Simulate encryption keys request
	err = s.manager.updateEncryptionKeysRequests(community.ID(), []string{"channel-id"}, now)
	s.Require().NoError(err)

	// Channel with prior request should not be returned before backoff interval
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now)
	s.Require().NoError(err)
	s.Require().Len(channels, 0)

	// Channel with prior request should be returned only after backoff interval
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now+tenMinutes)
	s.Require().NoError(err)
	s.Require().Len(channels, 1)
	s.Require().Equal("channel-id", channels[0])

	// Simulate multiple encryption keys request
	err = s.manager.updateEncryptionKeysRequests(community.ID(), []string{"channel-id"}, now+tenMinutes)
	s.Require().NoError(err)
	err = s.manager.updateEncryptionKeysRequests(community.ID(), []string{"channel-id"}, now+2*tenMinutes)
	s.Require().NoError(err)

	// Channel with prior request should not be returned before backoff interval
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now+2*tenMinutes)
	s.Require().NoError(err)
	s.Require().Len(channels, 0)

	// Channel with prior request should be returned only after backoff interval
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now+6*tenMinutes)
	s.Require().NoError(err)
	s.Require().Len(channels, 1)
	s.Require().Equal("channel-id", channels[0])

	// Simulate encryption key being received (it will remove request for given channel)
	err = s.manager.updateEncryptionKeysRequests(community.ID(), []string{}, now)
	s.Require().NoError(err)

	// Channel without prior request should be returned
	channels, err = s.manager.determineChannelsForHRKeysRequest(community, now)
	s.Require().NoError(err)
	s.Require().Len(channels, 1)
	s.Require().Equal("channel-id", channels[0])
}

// Covers solution for: https://github.com/status-im/status-desktop/issues/16226
func (s *ManagerSuite) TestCommunityIDIsHydratedWhenMarshaling() {
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	// Simulate legacy community that wasn't aware of ID field in `CommunityDescription` protobuf
	community.config.CommunityDescription.ID = ""

	// The fix is applied when community is marshaled, effectively hydrating empty ID
	err = s.manager.SaveCommunity(community)
	s.Require().NoError(err)

	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(community.IDString(), community.config.CommunityDescription.ID)
}

// TestHandleCommunityDescriptionRedeliveryGate verifies the redelivery gate
// (issue #21470-hf): a byte-identical redelivery of an already-processed
// community description is short-circuited (nil response, no error), while newer
// clocks are still processed, and new hash-ratchet key material lifts the gate so
// the same description is reprocessed (it may then decrypt private data).
func (s *ManagerSuite) TestHandleCommunityDescriptionRedeliveryGate() {
	// s.manager is the control node that authors the community; m is a separate
	// receiver (spectator) node that ingests the published description.
	m, _ := s.buildManagers(nil)

	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	description := community.config.CommunityDescription
	// Plain community: no token ownership, so it is processed immediately (not queued).
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	buildPayload := func() []byte {
		payload, err := community.MarshaledDescription()
		s.Require().NoError(err)
		payload, err = v.WrapIntoAppLayerMessage(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, signer)
		s.Require().NoError(err)
		return payload
	}

	payload := buildPayload()

	// First delivery is fully processed and yields a response.
	resp1, err := m.HandleCommunityDescriptionMessage(&signer.PublicKey, description, payload, nil)
	s.Require().NoError(err)
	s.Require().NotNil(resp1)

	// Byte-identical redelivery is short-circuited by the gate (nil response, no error).
	resp2, err := m.HandleCommunityDescriptionMessage(&signer.PublicKey, description, payload, nil)
	s.Require().NoError(err)
	s.Require().Nil(resp2)

	// A genuinely newer clock still gets processed.
	community.config.CommunityDescription.Clock++
	newerPayload := buildPayload()
	resp3, err := m.HandleCommunityDescriptionMessage(&signer.PublicKey, description, newerPayload, nil)
	s.Require().NoError(err)
	s.Require().NotNil(resp3)

	// Its identical redelivery is skipped again.
	resp4, err := m.HandleCommunityDescriptionMessage(&signer.PublicKey, description, newerPayload, nil)
	s.Require().NoError(err)
	s.Require().Nil(resp4)

	// New hash-ratchet keys arriving must lift the gate so the same description is
	// reprocessed (it may now decrypt previously-encrypted private data).
	err = m.NewHashRatchetKeys([]*types.HashRatchetInfo{{GroupID: community.ID(), KeyID: []byte("key-id")}})
	s.Require().NoError(err)

	resp5, err := m.HandleCommunityDescriptionMessage(&signer.PublicKey, description, newerPayload, nil)
	s.Require().NoError(err)
	s.Require().NotNil(resp5, "gate must be lifted after new keys arrive")
}
