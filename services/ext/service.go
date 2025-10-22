package ext

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	commongethtypes "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/api/multiformat"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	communitiestoken "github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/communitytokens"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/collectibles"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/signal"
)

const infinityString = "∞"
const providerID = "community"

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}

// Service is a service that provides some additional API to whisper-based protocols like Whisper or Waku.
type Service struct {
	messaging       *messaging.API
	messenger       *protocol.Messenger
	cancelMessenger chan struct{}
	rpcClient       *rpc.Client
	config          params.NodeConfig
	accountsDB      *accounts.Database
	multiAccountsDB *multiaccounts.Database
	account         *multiaccounts.Account

	logger *zap.Logger
}

// Make sure that Service implements node.Service interface.
var _ node.Lifecycle = (*Service)(nil)

func New(
	config params.NodeConfig,
	rpcClient *rpc.Client,
	logger *zap.Logger,
) *Service {
	return &Service{
		rpcClient: rpcClient,
		config:    config,
		logger:    logger,
	}
}

type InitProtocolParams struct {
	Identity               *ecdsa.PrivateKey
	AppDB                  *sql.DB
	WalletDB               *sql.DB
	HTTPServer             *server.MediaServer
	MultiAccountDB         *multiaccounts.Database
	Account                *multiaccounts.Account
	AccountsManager        *accsmanagement.AccountsManager
	RPCClient              *rpc.Client
	WalletService          *wallet.Service
	CommunityTokensService *communitytokens.Service
	AccountsPublisher      *pubsub.Publisher
	TimeSource             timesource.Provider
	MetricsEnabled         bool
	TokenManager           communities.TokenManager
	TokenBalanceManager    communities.TokenBalanceManager
	NetworkManager         communities.NetworkManager
}

func (s *Service) InitProtocol(params InitProtocolParams) error {
	var err error
	if !s.config.ShhextConfig.PFSEnabled {
		return nil
	}

	// If Messenger has been already set up, we need to shut it down
	// before we init it again. Otherwise, it will lead to goroutines leakage
	// due to not stopped filters.
	if s.messenger != nil {
		if err := s.messenger.Shutdown(); err != nil {
			return err
		}
	}
	if s.messaging != nil {
		if err := s.messaging.Stop(); err != nil {
			return err
		}
	}

	// This directory should have already been created in loadNodeConfig, keeping this to ensure.
	dataDir := filepath.Clean(s.config.RootDataDir)

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	s.accountsDB, err = accounts.NewDB(params.AppDB)
	if err != nil {
		return err
	}
	s.multiAccountsDB = params.MultiAccountDB
	s.account = params.Account

	err = sqlite.Migrate(params.AppDB)
	if err != nil {
		return err
	}

	nodeKey, err := obtainNodeKey(&s.config)
	if err != nil {
		return err
	}

	envelopeEventsConfig := &messagingtypes.EnvelopeEventsConfig{
		MaxMessageDeliveryAttempts: s.config.ShhextConfig.MaxMessageDeliveryAttempts,
		MailServerConfirmations:    s.config.ShhextConfig.MailServerConfirmations,
		EnvelopeEventsHandler:      EnvelopeSignalHandler{},
	}

	tracer := trace.NewNoopTracer()
	if s.config.OTELConfig.Enabled {
		name := params.Account.Name
		if name == "" {
			name = crypto.PubkeyToHex(&params.Identity.PublicKey)[0:8]
		}
		tracer = trace.NewTracer(otel.Tracer(name))
	}

	messaging, err := messaging.NewCore(
		messaging.CoreParams{
			Identity:       params.Identity,
			NodeKey:        nodeKey,
			WakuConfig:     s.config.WakuV2Config,
			ClusterConfig:  s.config.ClusterConfig,
			InstallationID: s.config.ShhextConfig.InstallationID,
			TimeSource:     params.TimeSource,
		},
		messaging.WithSQLitePersistence(params.AppDB),
		messaging.WithLogger(s.logger),
		messaging.WithEnvelopeEventsConfig(envelopeEventsConfig),
		messaging.WithHistoricMessagesRequestFailedHandler(signal.SendHistoricMessagesRequestFailed),
		messaging.WithPeerStatsHandler(signal.SendPeerStats),
		messaging.WithMetrics(params.MetricsEnabled),
		messaging.WithTracer(tracer),
	)
	if err != nil {
		return err
	}
	s.messaging = messaging.API()

	ensVerifier := ens.New(
		s.logger,
		params.TimeSource,
		params.AppDB,
		s.config.ShhextConfig.VerifyENSURL,
		s.config.ShhextConfig.VerifyENSContractAddress,
	)

	options, err := buildMessengerOptions(s.config, params.Identity, params.AppDB, params.WalletDB, params.HTTPServer,
		s.rpcClient, s.multiAccountsDB, params.Account, envelopeEventsConfig, s.accountsDB, params.WalletService,
		params.CommunityTokensService, s.logger, &MessengerSignalsHandler{}, params.AccountsManager, params.AccountsPublisher,
		ensVerifier, params.TokenManager, params.TokenBalanceManager, params.NetworkManager)
	if err != nil {
		return err
	}
	options = append(options, protocol.WithTracer(tracer))

	messenger, err := protocol.NewMessenger(
		params.Identity,
		messaging.API(),
		s.config.ShhextConfig.InstallationID,
		options...,
	)
	if err != nil {
		return err
	}
	s.messenger = messenger

	// Be mindful of adding more initialization code, as it can easily
	// impact login times for mobile users. For example, we avoid calling
	// messenger.InitFilters here.
	return s.messenger.InitInstallations()
}

func (s *Service) StartMessenger() (*protocol.MessengerResponse, error) {
	err := s.messaging.Start()
	if err != nil {
		return nil, err
	}

	// Start a loop that retrieves all messages and propagates them to status-mobile.
	s.cancelMessenger = make(chan struct{})
	response, err := s.messenger.Start()
	if err != nil {
		return nil, err
	}
	s.messenger.StartRetrieveMessagesLoop(time.Second, s.cancelMessenger)

	if s.config.ShhextConfig.BandwidthStatsEnabled {
		go s.retrieveStats(5*time.Second, s.cancelMessenger)
	}

	return response, nil
}

func (s *Service) retrieveStats(tick time.Duration, cancel <-chan struct{}) {
	defer gocommon.LogOnPanic()
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			response := s.messenger.GetStats()
			PublisherSignalHandler{}.Stats(response)
		case <-cancel:
			return
		}
	}
}

func (s *Service) EnableInstallation(installationID string) error {
	_, err := s.messenger.EnableInstallation(installationID)
	return err
}

// DisableInstallation disables an installation for multi-device sync.
func (s *Service) DisableInstallation(installationID string) error {
	return s.messenger.DisableInstallation(installationID)
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []gethrpc.API {
	panic("this is abstract service, use shhext or wakuext implementation")
}

// Start is run when a service is started.
func (s *Service) Start() error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	s.logger.Info("Stopping shhext service")
	if s.cancelMessenger != nil {
		select {
		case <-s.cancelMessenger:
			// channel already closed
		default:
			close(s.cancelMessenger)
			s.cancelMessenger = nil
		}
	}

	if s.messenger != nil {
		if err := s.messenger.Shutdown(); err != nil {
			s.logger.Error("failed to stop messenger", zap.Error(err))
			return err
		}
		s.messenger = nil
	}

	if s.messaging != nil {
		err := s.messaging.Stop()
		if err != nil {
			s.logger.Error("failed to stop messaging", zap.Error(err))
			return err
		}
		s.messaging = nil
	}

	return nil
}

func buildMessengerOptions(
	config params.NodeConfig,
	identity *ecdsa.PrivateKey,
	appDb *sql.DB,
	walletDb *sql.DB,
	httpServer *server.MediaServer,
	rpcClient *rpc.Client,
	multiAccounts *multiaccounts.Database,
	account *multiaccounts.Account,
	envelopeEventsConfig *messagingtypes.EnvelopeEventsConfig,
	accountsDB *accounts.Database,
	walletService *wallet.Service,
	communityTokensService *communitytokens.Service,
	logger *zap.Logger,
	messengerSignalsHandler protocol.MessengerSignalsHandler,
	accountsManager *accsmanagement.AccountsManager,
	accountsPublisher *pubsub.Publisher,
	ensVerifier *ens.Verifier,
	tokenManager communities.TokenManager,
	tokenBalanceManager communities.TokenBalanceManager,
	networkManager communities.NetworkManager,
) ([]protocol.Option, error) {
	personalService := personal.New()
	options := []protocol.Option{
		protocol.WithCustomLogger(logger),
		protocol.WithPushNotifications(),
		protocol.WithDatabase(appDb),
		protocol.WithWalletDatabase(walletDb),
		protocol.WithMultiAccounts(multiAccounts),
		protocol.WithMailserversDatabase(mailserversDB.NewDB(appDb)),
		protocol.WithAccount(account),
		protocol.WithBrowserDatabase(browsers.NewDB(appDb)),
		protocol.WithEnvelopeEventsConfig(envelopeEventsConfig),
		protocol.WithSignalsHandler(messengerSignalsHandler),
		protocol.WithENSVerifier(ensVerifier),
		protocol.WithClusterConfig(config.ClusterConfig),
		protocol.WithTorrentConfig(&config.TorrentConfig),
		protocol.WithCodexConfig(config.CodexConfig),
		protocol.WithHTTPServer(httpServer),
		protocol.WithRPCClient(rpcClient),
		protocol.WithMessageCSV(config.OutputMessageCSVEnabled),
		protocol.WithWalletService(walletService),
		protocol.WithCommunityTokensService(communityTokensService),
		protocol.WithAccountsManager(accountsManager),
		protocol.WithAccountsPublisher(accountsPublisher),
		protocol.WithMessageSigner(personalService),
		protocol.WithTokenManager(tokenManager),
		protocol.WithTokenBalanceManager(tokenBalanceManager),
		protocol.WithNetworkManager(networkManager),
	}

	if config.ShhextConfig.DataSyncEnabled {
		options = append(options, protocol.WithDatasync())
	}

	settings, err := accountsDB.GetSettings()
	if err != sql.ErrNoRows && err != nil {
		return nil, err
	}

	options = append(options, protocol.WithPushNotificationClientConfig(&pushnotificationclient.Config{
		DefaultServers:             config.ShhextConfig.PushNotificationsServers,
		BlockMentions:              settings.PushNotificationsBlockMentions,
		SendEnabled:                settings.SendPushNotifications,
		AllowFromContactsOnly:      settings.PushNotificationsFromContactsOnly,
		RemoteNotificationsEnabled: settings.RemotePushNotificationsEnabled,
	}))

	return options, nil
}

func (s *Service) Messenger() *protocol.Messenger {
	return s.messenger
}

func (s *Service) Metrics() string {
	return s.messaging.Metrics()
}

func tokenURIToCommunityID(tokenURI string) string {
	tmpStr := strings.Split(tokenURI, "/")

	// Community NFTs have a tokenURI of the form "compressedCommunityID/tokenID"
	if len(tmpStr) != 2 {
		return ""
	}
	compressedCommunityID := tmpStr[0]

	hexCommunityID, err := multiformat.DeserializeCompressedKey(compressedCommunityID)
	if err != nil {
		return ""
	}

	pubKey, err := crypto.HexToPubkey(hexCommunityID)
	if err != nil {
		return ""
	}

	communityID := types.EncodeHex(crypto.CompressPubkey(pubKey))

	return communityID
}

func (s *Service) GetCommunityID(tokenURI string) string {
	if tokenURI != "" {
		return tokenURIToCommunityID(tokenURI)
	}
	return ""
}

func (s *Service) FillCollectiblesMetadata(communityID string, cs []*thirdparty.FullCollectibleData) (bool, error) {
	if s.messenger == nil {
		return false, fmt.Errorf("messenger not ready")
	}

	community, err := s.fetchCommunityInfoForCollectibles(communityID, collectibles.IDsFromAssets(cs))
	if err != nil {
		return false, err
	}
	if community == nil {
		return false, nil
	}

	for _, collectible := range cs {
		err := s.FillCollectibleMetadata(community, collectible)
		if err != nil {
			return true, err
		}
	}

	return true, nil
}

func (s *Service) FillCollectibleMetadata(community *communities.Community, collectible *thirdparty.FullCollectibleData) error {
	if s.messenger == nil {
		return fmt.Errorf("messenger not ready")
	}

	if collectible == nil {
		return fmt.Errorf("empty collectible")
	}

	id := collectible.CollectibleData.ID
	communityID := collectible.CollectibleData.CommunityID

	if communityID == "" {
		return fmt.Errorf("invalid communityID")
	}

	tokenMetadata, err := s.fetchCommunityCollectibleMetadata(community, id.ContractID)

	if err != nil {
		return err
	}

	if tokenMetadata == nil {
		return nil
	}

	communityToken, err := s.fetchCommunityToken(communityID, id.ContractID)
	if err != nil {
		return err
	}

	permission := fetchCommunityCollectiblePermission(community, id)

	privilegesLevel := communitiestoken.CommunityLevel
	if permission != nil {
		privilegesLevel = permissionTypeToPrivilegesLevel(permission.GetType())
	}

	imagePayload, _ := images.GetPayloadFromURI(tokenMetadata.GetImage())

	collectible.CollectibleData.ContractType = w_common.ContractTypeERC721
	collectible.CollectibleData.Provider = providerID
	collectible.CollectibleData.Name = tokenMetadata.GetName()
	collectible.CollectibleData.Description = tokenMetadata.GetDescription()
	collectible.CollectibleData.ImagePayload = imagePayload
	collectible.CollectibleData.Traits = getCollectibleCommunityTraits(communityToken)
	collectible.CollectibleData.Soulbound = !communityToken.Transferable

	if collectible.CollectionData == nil {
		collectible.CollectionData = &thirdparty.CollectionData{
			ID:          id.ContractID,
			CommunityID: communityID,
		}
	}
	collectible.CollectionData.ContractType = w_common.ContractTypeERC721
	collectible.CollectionData.Provider = providerID
	collectible.CollectionData.Name = tokenMetadata.GetName()
	collectible.CollectionData.ImagePayload = imagePayload

	collectible.CommunityInfo = communityToInfo(community)

	collectible.CollectibleCommunityInfo = &thirdparty.CollectibleCommunityInfo{
		PrivilegesLevel: privilegesLevel,
	}

	return nil
}

func permissionTypeToPrivilegesLevel(permissionType protobuf.CommunityTokenPermission_Type) communitiestoken.PrivilegesLevel {
	switch permissionType {
	case protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER:
		return communitiestoken.OwnerLevel
	case protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER:
		return communitiestoken.MasterLevel
	default:
		return communitiestoken.CommunityLevel
	}
}

func communityToInfo(community *communities.Community) *thirdparty.CommunityInfo {
	if community == nil {
		return nil
	}

	return &thirdparty.CommunityInfo{
		CommunityName:         community.Name(),
		CommunityColor:        community.Color(),
		CommunityImagePayload: fetchCommunityImage(community),
	}
}

func (s *Service) fetchCommunityFromStoreNodes(communityID string) (*communities.Community, error) {
	community, err := s.messenger.FetchCommunity(&protocol.FetchCommunityRequest{
		CommunityKey:    communityID,
		TryDatabase:     false,
		WaitForResponse: true,
	})
	if err != nil {
		return nil, err
	}
	return community, nil
}

// Fetch latest community from store nodes.
func (s *Service) FetchCommunityInfo(communityID string) (*thirdparty.CommunityInfo, error) {
	if s.messenger == nil {
		return nil, fmt.Errorf("messenger not ready")
	}

	community, err := s.messenger.FindCommunityInfoFromDB(communityID)
	if err != nil && err != communities.ErrOrgNotFound {
		return nil, err
	}

	// Fetch latest version from store nodes
	if community == nil || !community.IsControlNode() {
		community, err = s.fetchCommunityFromStoreNodes(communityID)
		if err != nil {
			return nil, err
		}
	}

	return communityToInfo(community), nil
}

// Fetch latest community from store nodes only if any collectibles data is missing.
func (s *Service) fetchCommunityInfoForCollectibles(communityID string, ids []thirdparty.CollectibleUniqueID) (*communities.Community, error) {
	community, err := s.messenger.FindCommunityInfoFromDB(communityID)
	if err != nil && err != communities.ErrOrgNotFound {
		return nil, err
	}

	if community == nil {
		return s.fetchCommunityFromStoreNodes(communityID)
	}

	if community.IsControlNode() {
		return community, nil
	}

	contractIDs := func() map[string]thirdparty.ContractID {
		result := map[string]thirdparty.ContractID{}
		for _, id := range ids {
			result[id.HashKey()] = id.ContractID
		}
		return result
	}()

	hasAllMetadata := true
	for _, contractID := range contractIDs {
		tokenMetadata, err := s.fetchCommunityCollectibleMetadata(community, contractID)
		if err != nil {
			return nil, err
		}
		if tokenMetadata == nil {
			hasAllMetadata = false
			break
		}
	}

	if !hasAllMetadata {
		return s.fetchCommunityFromStoreNodes(communityID)
	}

	return community, nil
}

func (s *Service) fetchCommunityToken(communityID string, contractID thirdparty.ContractID) (*communitiestoken.CommunityToken, error) {
	if s.messenger == nil {
		return nil, fmt.Errorf("messenger not ready")
	}

	return s.messenger.GetCommunityToken(communityID, int(contractID.ChainID), contractID.Address.String())
}

func (s *Service) fetchCommunityCollectibleMetadata(community *communities.Community, contractID thirdparty.ContractID) (*protobuf.CommunityTokenMetadata, error) {
	tokensMetadata := community.CommunityTokensMetadata()

	for _, tokenMetadata := range tokensMetadata {
		contractAddresses := tokenMetadata.GetContractAddresses()
		if contractAddresses[uint64(contractID.ChainID)] == contractID.Address.Hex() {
			return tokenMetadata, nil
		}
	}

	return nil, nil
}

func tokenCriterionContainsCollectible(tokenCriterion *protobuf.TokenCriteria, id thirdparty.CollectibleUniqueID) bool {
	// Check if token type matches
	if tokenCriterion.Type != protobuf.CommunityTokenType_ERC721 {
		return false
	}

	for chainID, contractAddressStr := range tokenCriterion.ContractAddresses {
		if chainID != uint64(id.ContractID.ChainID) {
			continue
		}

		contractAddress := commongethtypes.HexToAddress(contractAddressStr)
		if contractAddress != id.ContractID.Address {
			continue
		}

		if len(tokenCriterion.TokenIds) == 0 {
			return true
		}

		for _, tokenID := range tokenCriterion.TokenIds {
			tokenIDBigInt := new(big.Int).SetUint64(tokenID)
			if id.TokenID.Cmp(tokenIDBigInt) == 0 {
				return true
			}
		}
	}

	return false
}

func permissionContainsCollectible(permission *communities.CommunityTokenPermission, id thirdparty.CollectibleUniqueID) bool {
	// See if any token criterion contains the collectible we're looking for
	for _, tokenCriterion := range permission.TokenCriteria {
		if tokenCriterionContainsCollectible(tokenCriterion, id) {
			return true
		}
	}
	return false
}

func fetchCommunityCollectiblePermission(community *communities.Community, id thirdparty.CollectibleUniqueID) *communities.CommunityTokenPermission {
	// Permnission types of interest
	permissionTypes := []protobuf.CommunityTokenPermission_Type{
		protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
	}

	for _, permissionType := range permissionTypes {
		permissions := community.TokenPermissionsByType(permissionType)
		// See if any community permission matches the type we're looking for
		for _, permission := range permissions {
			if permissionContainsCollectible(permission, id) {
				return permission
			}
		}
	}

	return nil
}

func fetchCommunityImage(community *communities.Community) []byte {
	imageTypes := []string{
		images.LargeDimName,
		images.SmallDimName,
	}

	communityImages := community.Images()

	for _, imageType := range imageTypes {
		if pbImage, ok := communityImages[imageType]; ok {
			return pbImage.Payload
		}
	}

	return nil
}

func boolToString(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func getCollectibleCommunityTraits(token *communitiestoken.CommunityToken) []thirdparty.CollectibleTrait {
	if token == nil {
		return make([]thirdparty.CollectibleTrait, 0)
	}

	totalStr := infinityString
	availableStr := infinityString
	if !token.InfiniteSupply {
		totalStr = token.Supply.String()
		// TODO: calculate available supply. See services/communitytokens/api.go
		availableStr = totalStr
	}

	transferableStr := boolToString(token.Transferable)

	destructibleStr := boolToString(token.RemoteSelfDestruct)

	return []thirdparty.CollectibleTrait{
		{
			TraitType: "Symbol",
			Value:     token.Symbol,
		},
		{
			TraitType: "Total",
			Value:     totalStr,
		},
		{
			TraitType: "Available",
			Value:     availableStr,
		},
		{
			TraitType: "Transferable",
			Value:     transferableStr,
		},
		{
			TraitType: "Destructible",
			Value:     destructibleStr,
		},
	}
}

func obtainNodeKey(nodeConfig *params.NodeConfig) (*ecdsa.PrivateKey, error) {
	if nodeConfig.NodeKey != "" {
		return crypto.HexToECDSA(nodeConfig.NodeKey)
	}

	nodeKeyStr := os.Getenv("WAKUV2_NODE_KEY")
	if nodeKeyStr != "" {
		nodeKeyBytes, err := hexutil.Decode(nodeKeyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode the go-waku private key: %v", err)
		}

		return crypto.ToECDSA(nodeKeyBytes)
	}

	return nil, nil
}
