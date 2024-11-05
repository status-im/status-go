package communities

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	multiaccountscommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	community_token "github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
)

type Publisher interface {
	publish(subscription *Subscription)
}

var defaultAnnounceList = [][]string{
	{"udp://tracker.opentrackr.org:1337/announce"},
	{"udp://tracker.openbittorrent.com:6969/announce"},
}
var pieceLength = 100 * 1024

const maxArchiveSizeInBytes = 30000000

var maxNbMembers = 5000
var maxNbPendingRequestedMembers = 100

var memberPermissionsCheckInterval = 8 * time.Hour
var validateInterval = 2 * time.Minute

// Used for testing only
func SetValidateInterval(duration time.Duration) {
	validateInterval = duration
}
func SetMaxNbMembers(maxNb int) {
	maxNbMembers = maxNb
}
func SetMaxNbPendingRequestedMembers(maxNb int) {
	maxNbPendingRequestedMembers = maxNb
}

// errors
var (
	ErrTorrentTimedout                 = errors.New("torrent has timed out")
	ErrCommunityRequestAlreadyRejected = errors.New("that user was already rejected from the community")
	ErrInvalidClock                    = errors.New("invalid clock to cancel request to join")
)

type Manager struct {
	persistence              *Persistence
	encryptor                *encryption.Protocol
	ensSubscription          chan []*ens.VerificationRecord
	subscriptions            []chan *Subscription
	ensVerifier              *ens.Verifier
	ownerVerifier            OwnerVerifier
	identity                 *ecdsa.PrivateKey
	installationID           string
	accountsManager          account.Manager
	tokenManager             TokenManager
	collectiblesManager      CollectiblesManager
	logger                   *zap.Logger
	transport                *transport.Transport
	timesource               common.TimeSource
	quit                     chan struct{}
	walletConfig             *params.WalletConfig
	communityTokensService   CommunityTokensServiceInterface
	membersReevaluationTasks sync.Map // stores `membersReevaluationTask`
	forceMembersReevaluation map[string]chan struct{}
	stopped                  bool
	RekeyInterval            time.Duration
	PermissionChecker        PermissionChecker
	keyDistributor           KeyDistributor
	communityLock            *CommunityLock
	mediaServer              server.MediaServerInterface
}

type CommunityLock struct {
	logger *zap.Logger
	locks  map[string]*sync.Mutex
	mutex  sync.Mutex
}

func NewCommunityLock(logger *zap.Logger) *CommunityLock {
	return &CommunityLock{
		logger: logger.Named("CommunityLock"),
		locks:  make(map[string]*sync.Mutex),
	}
}

func (c *CommunityLock) Lock(communityID types.HexBytes) {
	c.mutex.Lock()
	communityIDStr := types.EncodeHex(communityID)
	lock, ok := c.locks[communityIDStr]
	if !ok {
		lock = &sync.Mutex{}
		c.locks[communityIDStr] = lock
	}
	c.mutex.Unlock()

	lock.Lock()
}

func (c *CommunityLock) Unlock(communityID types.HexBytes) {
	c.mutex.Lock()
	communityIDStr := types.EncodeHex(communityID)
	lock, ok := c.locks[communityIDStr]
	c.mutex.Unlock()

	if ok {
		lock.Unlock()
	} else {
		c.logger.Warn("trying to unlock a non-existent lock", zap.String("communityID", communityIDStr))
	}
}

func (c *CommunityLock) Init() {
	c.locks = make(map[string]*sync.Mutex)
}

type HistoryArchiveDownloadTask struct {
	CancelChan chan struct{}
	Waiter     sync.WaitGroup
	m          sync.RWMutex
	Cancelled  bool
}

type HistoryArchiveDownloadTaskInfo struct {
	TotalDownloadedArchivesCount int
	TotalArchivesCount           int
	Cancelled                    bool
}

type ArchiveFileService interface {
	CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error)
	CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error)
	SaveMessageArchiveID(communityID types.HexBytes, hash string) error
	GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error)
	SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error
	ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error)
	GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error)
	LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error)
}

type ArchiveService interface {
	ArchiveFileService

	SetOnline(bool)
	SetTorrentConfig(*params.TorrentConfig)
	StartTorrentClient() error
	Stop() error
	IsReady() bool
	GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error)
	GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error)
	GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error)
	CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error
	StartHistoryArchiveTasksInterval(community *Community, interval time.Duration)
	StopHistoryArchiveTasksInterval(communityID types.HexBytes)
	SeedHistoryArchiveTorrent(communityID types.HexBytes) error
	UnseedHistoryArchiveTorrent(communityID types.HexBytes)
	IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool
	GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask
	AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask)
	DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error)
	TorrentFileExists(communityID string) bool
}

type ArchiveManagerConfig struct {
	TorrentConfig *params.TorrentConfig
	Logger        *zap.Logger
	Persistence   *Persistence
	Transport     *transport.Transport
	Identity      *ecdsa.PrivateKey
	Encryptor     *encryption.Protocol
	Publisher     Publisher
}

func (t *HistoryArchiveDownloadTask) IsCancelled() bool {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.Cancelled
}

func (t *HistoryArchiveDownloadTask) Cancel() {
	t.m.Lock()
	defer t.m.Unlock()
	t.Cancelled = true
	close(t.CancelChan)
}

type membersReevaluationTask struct {
	lastStartTime       time.Time
	lastSuccessTime     time.Time
	onDemandRequestTime time.Time
	mutex               sync.Mutex
}

type managerOptions struct {
	accountsManager        account.Manager
	tokenManager           TokenManager
	collectiblesManager    CollectiblesManager
	walletConfig           *params.WalletConfig
	communityTokensService CommunityTokensServiceInterface
	permissionChecker      PermissionChecker

	// allowForcingCommunityMembersReevaluation indicates whether we should allow forcing community members reevaluation.
	// This will allow using `force` argument in ScheduleMembersReevaluation.
	// Should only be used in tests.
	allowForcingCommunityMembersReevaluation bool
}

type TokenManager interface {
	GetBalancesByChain(ctx context.Context, accounts, tokens []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error)
	GetCachedBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error)
	FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) *token.Token
	GetAllChainIDs() ([]uint64, error)
}

type CollectibleContractData struct {
	TotalSupply    *bigint.BigInt
	Transferable   bool
	RemoteBurnable bool
	InfiniteSupply bool
}

type AssetContractData struct {
	TotalSupply    *bigint.BigInt
	InfiniteSupply bool
}

type CommunityTokensServiceInterface interface {
	GetCollectibleContractData(chainID uint64, contractAddress string) (*CollectibleContractData, error)
	SetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, newSignerPubKey string) (string, error)
	GetAssetContractData(chainID uint64, contractAddress string) (*AssetContractData, error)
	SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error)
	DeploymentSignatureDigest(chainID uint64, addressFrom string, communityID string) ([]byte, error)
	ProcessCommunityTokenAction(message *protobuf.CommunityTokenAction) error
}

type DefaultTokenManager struct {
	tokenManager   *token.Manager
	networkManager network.ManagerInterface
}

func NewDefaultTokenManager(tm *token.Manager, nm network.ManagerInterface) *DefaultTokenManager {
	return &DefaultTokenManager{tokenManager: tm, networkManager: nm}
}

type BalancesByChain = map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big

func (m *DefaultTokenManager) GetAllChainIDs() ([]uint64, error) {
	networks, err := m.networkManager.Get(false)
	if err != nil {
		return nil, err
	}

	areTestNetworksEnabled, err := m.networkManager.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		if areTestNetworksEnabled == network.IsTest {
			chainIDs = append(chainIDs, network.ChainID)
		}
	}
	return chainIDs, nil
}

type CollectiblesManager interface {
	FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletcommon.ChainID, ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error)
	FetchCachedBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletcommon.ChainID, ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error)
	GetCollectibleOwnership(id thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error)
	FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletcommon.ChainID, contractAddress gethcommon.Address) (*thirdparty.CollectibleContractOwnership, error)
}

func (m *DefaultTokenManager) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error) {
	clients, err := m.tokenManager.RPCClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	resp, err := m.tokenManager.GetBalancesByChain(context.Background(), clients, accounts, tokenAddresses)
	return resp, err
}

func (m *DefaultTokenManager) GetCachedBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error) {
	resp, err := m.tokenManager.GetCachedBalancesByChain(accounts, tokenAddresses, chainIDs)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (m *DefaultTokenManager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) *token.Token {
	return m.tokenManager.FindOrCreateTokenByAddress(ctx, chainID, address)
}

type ManagerOption func(*managerOptions)

func WithAccountManager(accountsManager account.Manager) ManagerOption {
	return func(opts *managerOptions) {
		opts.accountsManager = accountsManager
	}
}

func WithPermissionChecker(permissionChecker PermissionChecker) ManagerOption {
	return func(opts *managerOptions) {
		opts.permissionChecker = permissionChecker
	}
}

func WithCollectiblesManager(collectiblesManager CollectiblesManager) ManagerOption {
	return func(opts *managerOptions) {
		opts.collectiblesManager = collectiblesManager
	}
}

func WithTokenManager(tokenManager TokenManager) ManagerOption {
	return func(opts *managerOptions) {
		opts.tokenManager = tokenManager
	}
}

func WithWalletConfig(walletConfig *params.WalletConfig) ManagerOption {
	return func(opts *managerOptions) {
		opts.walletConfig = walletConfig
	}
}

func WithCommunityTokensService(communityTokensService CommunityTokensServiceInterface) ManagerOption {
	return func(opts *managerOptions) {
		opts.communityTokensService = communityTokensService
	}
}

func WithAllowForcingCommunityMembersReevaluation(enabled bool) ManagerOption {
	return func(opts *managerOptions) {
		opts.allowForcingCommunityMembersReevaluation = enabled
	}
}

type OwnerVerifier interface {
	SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error)
}

func NewManager(
	identity *ecdsa.PrivateKey,
	installationID string,
	db *sql.DB,
	encryptor *encryption.Protocol,
	logger *zap.Logger,
	ensverifier *ens.Verifier,
	ownerVerifier OwnerVerifier,
	transport *transport.Transport,
	timesource common.TimeSource,
	keyDistributor KeyDistributor,
	mediaServer server.MediaServerInterface,
	opts ...ManagerOption,
) (*Manager, error) {
	if identity == nil {
		return nil, errors.New("empty identity")
	}

	if timesource == nil {
		return nil, errors.New("no timesource")
	}

	var err error
	if logger == nil {
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	managerConfig := managerOptions{}
	for _, opt := range opts {
		opt(&managerConfig)
	}

	manager := &Manager{
		logger:         logger,
		encryptor:      encryptor,
		identity:       identity,
		installationID: installationID,
		ownerVerifier:  ownerVerifier,
		quit:           make(chan struct{}),
		transport:      transport,
		timesource:     timesource,
		keyDistributor: keyDistributor,
		communityLock:  NewCommunityLock(logger),
		mediaServer:    mediaServer,
	}

	manager.persistence = &Persistence{
		db:                      db,
		recordBundleToCommunity: manager.dbRecordBundleToCommunity,
	}

	if managerConfig.accountsManager != nil {
		manager.accountsManager = managerConfig.accountsManager
	}

	if managerConfig.collectiblesManager != nil {
		manager.collectiblesManager = managerConfig.collectiblesManager
	}

	if managerConfig.tokenManager != nil {
		manager.tokenManager = managerConfig.tokenManager
	}

	if managerConfig.walletConfig != nil {
		manager.walletConfig = managerConfig.walletConfig
	}

	if managerConfig.communityTokensService != nil {
		manager.communityTokensService = managerConfig.communityTokensService
	}

	if ensverifier != nil {
		sub := ensverifier.Subscribe()
		manager.ensSubscription = sub
		manager.ensVerifier = ensverifier
	}

	if managerConfig.permissionChecker != nil {
		manager.PermissionChecker = managerConfig.permissionChecker
	} else {
		manager.PermissionChecker = &DefaultPermissionChecker{
			tokenManager:        manager.tokenManager,
			collectiblesManager: manager.collectiblesManager,
			logger:              logger,
			ensVerifier:         ensverifier,
		}
	}

	if managerConfig.allowForcingCommunityMembersReevaluation {
		manager.logger.Warn("allowing forcing community members reevaluation, this should only be used in test environment")
		manager.forceMembersReevaluation = make(map[string]chan struct{}, 10)
	}

	return manager, nil
}

type Subscription struct {
	Community                                *Community
	CreatingHistoryArchivesSignal            *signal.CreatingHistoryArchivesSignal
	HistoryArchivesCreatedSignal             *signal.HistoryArchivesCreatedSignal
	NoHistoryArchivesCreatedSignal           *signal.NoHistoryArchivesCreatedSignal
	HistoryArchivesSeedingSignal             *signal.HistoryArchivesSeedingSignal
	HistoryArchivesUnseededSignal            *signal.HistoryArchivesUnseededSignal
	HistoryArchiveDownloadedSignal           *signal.HistoryArchiveDownloadedSignal
	DownloadingHistoryArchivesStartedSignal  *signal.DownloadingHistoryArchivesStartedSignal
	DownloadingHistoryArchivesFinishedSignal *signal.DownloadingHistoryArchivesFinishedSignal
	ImportingHistoryArchiveMessagesSignal    *signal.ImportingHistoryArchiveMessagesSignal
	CommunityEventsMessage                   *CommunityEventsMessage
	AcceptedRequestsToJoin                   []types.HexBytes
	RejectedRequestsToJoin                   []types.HexBytes
	CommunityPrivilegedMemberSyncMessage     *CommunityPrivilegedMemberSyncMessage
	TokenCommunityValidated                  *CommunityResponse
}

type CommunityResponse struct {
	Community       *Community                             `json:"community"`
	Changes         *CommunityChanges                      `json:"changes"`
	RequestsToJoin  []*RequestToJoin                       `json:"requestsToJoin"`
	FailedToDecrypt []*CommunityPrivateDataFailedToDecrypt `json:"-"`
}

func (m *Manager) SetMediaServer(mediaServer server.MediaServerInterface) {
	m.mediaServer = mediaServer
}

func (m *Manager) Subscribe() chan *Subscription {
	subscription := make(chan *Subscription, 100)
	m.subscriptions = append(m.subscriptions, subscription)
	return subscription
}

func (m *Manager) Start() error {
	m.stopped = false
	m.communityLock.Init()
	if m.ensVerifier != nil {
		m.runENSVerificationLoop()
	}

	if m.ownerVerifier != nil {
		m.runOwnerVerificationLoop()
	}

	go func() {
		defer utils.LogOnPanic()
		_ = m.fillMissingCommunityTokens()
	}()

	return nil
}

func (m *Manager) runENSVerificationLoop() {
	go func() {
		defer utils.LogOnPanic()
		for {
			select {
			case <-m.quit:
				m.logger.Debug("quitting ens verification loop")
				return
			case records, more := <-m.ensSubscription:
				if !more {
					m.logger.Debug("no more ens records, quitting")
					return
				}
				m.logger.Info("received records", zap.Any("records", records))
			}
		}
	}()
}

// This function is mostly a way to fix any community that is missing tokens in its description
func (m *Manager) fillMissingCommunityTokens() error {
	controlledCommunities, err := m.Controlled()
	if err != nil {
		m.logger.Error("failed to retrieve orgs", zap.Error(err))
		return err
	}

	unlock := func() {
		for _, c := range controlledCommunities {
			m.communityLock.Unlock(c.ID())
		}
	}
	for _, c := range controlledCommunities {
		m.communityLock.Lock(c.ID())
	}
	defer unlock()

	for _, community := range controlledCommunities {
		tokens, err := m.GetCommunityTokens(community.IDString())
		if err != nil {
			m.logger.Error("failed to retrieve community tokens", zap.Error(err))
			return err
		}

		for _, token := range tokens {
			if token.DeployState != community_token.Deployed {
				continue
			}
			tokenMetadata := &protobuf.CommunityTokenMetadata{
				ContractAddresses: map[uint64]string{uint64(token.ChainID): token.Address},
				Description:       token.Description,
				Image:             token.Base64Image,
				Symbol:            token.Symbol,
				TokenType:         token.TokenType,
				Name:              token.Name,
				Decimals:          uint32(token.Decimals),
				Version:           token.Version,
			}
			modified, err := community.UpsertCommunityTokensMetadata(tokenMetadata)
			if err != nil {
				m.logger.Error("failed to add token metadata to the description", zap.Error(err))
				return err
			}
			if modified {
				err = m.saveAndPublish(community)
				if err != nil {
					m.logger.Error("failed to save the new community", zap.Error(err))
					return err
				}
			}
		}
	}
	return nil
}

// Only for testing
func (m *Manager) CommunitiesToValidate() (map[string][]communityToValidate, error) { // nolint: golint
	return m.persistence.getCommunitiesToValidate()
}

func (m *Manager) runOwnerVerificationLoop() {
	m.logger.Info("starting owner verification loop")
	go func() {
		defer utils.LogOnPanic()
		for {
			select {
			case <-m.quit:
				m.logger.Debug("quitting owner verification loop")
				return
			case <-time.After(validateInterval):
				// If ownerverifier is nil, we skip, this is useful for testing
				if m.ownerVerifier == nil {
					continue
				}

				communitiesToValidate, err := m.persistence.getCommunitiesToValidate()

				if err != nil {
					m.logger.Error("failed to fetch communities to validate", zap.Error(err))
					continue
				}
				for id, communities := range communitiesToValidate {
					m.logger.Info("validating communities", zap.String("id", id), zap.Int("count", len(communities)))

					_, _ = m.validateCommunity(communities)
				}
			}
		}
	}()
}

func (m *Manager) ValidateCommunityByID(communityID types.HexBytes) (*CommunityResponse, error) {
	communitiesToValidate, err := m.persistence.getCommunityToValidateByID(communityID)
	if err != nil {
		m.logger.Error("failed to validate community by ID", zap.String("id", communityID.String()), zap.Error(err))
		return nil, err
	}
	return m.validateCommunity(communitiesToValidate)

}

func (m *Manager) validateCommunity(communityToValidateData []communityToValidate) (*CommunityResponse, error) {
	for _, community := range communityToValidateData {
		signer, description, err := UnwrapCommunityDescriptionMessage(community.payload)
		if err != nil {
			m.logger.Error("failed to unwrap community", zap.Error(err))
			continue
		}

		chainID := CommunityDescriptionTokenOwnerChainID(description)
		if chainID == 0 {
			// This should not happen
			m.logger.Error("chain id is 0, ignoring")
			continue
		}

		m.logger.Info("validating community", zap.String("id", types.EncodeHex(community.id)), zap.String("signer", common.PubkeyToHex(signer)))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		owner, err := m.ownerVerifier.SafeGetSignerPubKey(ctx, chainID, types.EncodeHex(community.id))
		if err != nil {
			m.logger.Error("failed to get owner", zap.Error(err))
			continue
		}

		ownerPK, err := common.HexToPubkey(owner)
		if err != nil {
			m.logger.Error("failed to convert pk string to ecdsa", zap.Error(err))
			continue
		}

		// TODO: handle shards
		response, err := m.HandleCommunityDescriptionMessage(signer, description, community.payload, ownerPK, nil)
		if err != nil {
			m.logger.Error("failed to handle community", zap.Error(err))
			err = m.persistence.DeleteCommunityToValidate(community.id, community.clock)
			if err != nil {
				m.logger.Error("failed to delete community to validate", zap.Error(err))
			}
			continue
		}

		if response != nil {

			m.logger.Info("community validated", zap.String("id", types.EncodeHex(community.id)), zap.String("signer", common.PubkeyToHex(signer)))
			m.publish(&Subscription{TokenCommunityValidated: response})
			err := m.persistence.DeleteCommunitiesToValidateByCommunityID(community.id)
			if err != nil {
				m.logger.Error("failed to delete communities to validate", zap.Error(err))
			}
			return response, nil
		}
	}

	return nil, nil
}

func (m *Manager) Stop() error {
	m.stopped = true
	close(m.quit)
	for _, c := range m.subscriptions {
		close(c)
	}
	return nil
}

func (m *Manager) publish(subscription *Subscription) {
	if m.stopped {
		return
	}
	for _, s := range m.subscriptions {
		select {
		case s <- subscription:
		default:
			m.logger.Warn("subscription channel full, dropping message")
		}
	}
}

func (m *Manager) All() ([]*Community, error) {
	return m.persistence.AllCommunities(&m.identity.PublicKey)
}

type CommunityShard struct {
	CommunityID string       `json:"communityID"`
	Shard       *shard.Shard `json:"shard"`
}

type CuratedCommunities struct {
	ContractCommunities         []string
	ContractFeaturedCommunities []string
}

type KnownCommunitiesResponse struct {
	ContractCommunities         []string              `json:"contractCommunities"`
	ContractFeaturedCommunities []string              `json:"contractFeaturedCommunities"`
	Descriptions                map[string]*Community `json:"communities"`
	UnknownCommunities          []string              `json:"unknownCommunities"`
}

func (m *Manager) GetStoredDescriptionForCommunities(communityIDs []string) (*KnownCommunitiesResponse, error) {
	response := &KnownCommunitiesResponse{
		Descriptions: make(map[string]*Community),
	}

	for i := range communityIDs {
		communityID := communityIDs[i]
		communityIDBytes, err := types.DecodeHex(communityID)
		if err != nil {
			return nil, err
		}

		community, err := m.GetByID(types.HexBytes(communityIDBytes))
		if err != nil && err != ErrOrgNotFound {
			return nil, err
		}

		if community != nil {
			response.Descriptions[community.IDString()] = community
		} else {
			response.UnknownCommunities = append(response.UnknownCommunities, communityID)
		}

		response.ContractCommunities = append(response.ContractCommunities, communityID)
	}

	return response, nil
}

func (m *Manager) Joined() ([]*Community, error) {
	return m.persistence.JoinedCommunities(&m.identity.PublicKey)
}

func (m *Manager) Spectated() ([]*Community, error) {
	return m.persistence.SpectatedCommunities(&m.identity.PublicKey)
}

func (m *Manager) CommunityUpdateLastOpenedAt(communityID types.HexBytes, timestamp int64) (*Community, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	err = m.persistence.UpdateLastOpenedAt(community.ID(), timestamp)
	if err != nil {
		return nil, err
	}
	community.UpdateLastOpenedAt(timestamp)
	return community, nil
}

func (m *Manager) JoinedAndPendingCommunitiesWithRequests() ([]*Community, error) {
	return m.persistence.JoinedAndPendingCommunitiesWithRequests(&m.identity.PublicKey)
}

func (m *Manager) DeletedCommunities() ([]*Community, error) {
	return m.persistence.DeletedCommunities(&m.identity.PublicKey)
}

func (m *Manager) Controlled() ([]*Community, error) {
	communities, err := m.persistence.CommunitiesWithPrivateKey(&m.identity.PublicKey)
	if err != nil {
		return nil, err
	}

	controlled := make([]*Community, 0, len(communities))

	for _, c := range communities {
		if c.IsControlNode() {
			controlled = append(controlled, c)
		}
	}

	return controlled, nil
}

// CreateCommunity takes a description, generates an ID for it, saves it and return it
func (m *Manager) CreateCommunity(request *requests.CreateCommunity, publish bool) (*Community, error) {

	description, err := request.ToCommunityDescription()
	if err != nil {
		return nil, err
	}

	description.Members = make(map[string]*protobuf.CommunityMember)
	description.Members[common.PubkeyToHex(&m.identity.PublicKey)] = &protobuf.CommunityMember{Roles: []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_OWNER}}

	err = ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	description.Clock = 1

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	description.ID = types.EncodeHex(crypto.CompressPubkey(&key.PublicKey))

	config := Config{
		ID:                   &key.PublicKey,
		PrivateKey:           key,
		ControlNode:          &key.PublicKey,
		ControlDevice:        true,
		Logger:               m.logger,
		Joined:               true,
		JoinedAt:             time.Now().Unix(),
		MemberIdentity:       m.identity,
		CommunityDescription: description,
		Shard:                nil,
		LastOpenedAt:         0,
	}

	var descriptionEncryptor DescriptionEncryptor
	if m.encryptor != nil {
		descriptionEncryptor = m
	}
	community, err := New(config, m.timesource, descriptionEncryptor, m.mediaServer)
	if err != nil {
		return nil, err
	}

	// We join any community we create
	community.Join()

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	// Save grant for own community
	grant, err := community.BuildGrant(&m.identity.PublicKey, "")
	if err != nil {
		return nil, err
	}
	err = m.persistence.SaveCommunityGrant(community.IDString(), grant, uint64(time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	// Mark this device as the control node
	syncControlNode := &protobuf.SyncCommunityControlNode{
		Clock:          1,
		InstallationId: m.installationID,
	}
	err = m.SaveSyncControlNode(community.ID(), syncControlNode)
	if err != nil {
		return nil, err
	}

	if publish {
		m.publish(&Subscription{Community: community})
	}

	return community, nil
}

func (m *Manager) CreateCommunityTokenPermission(request *requests.CreateCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	// ensure key is generated before marshaling,
	// as it requires key to encrypt description
	if community.IsControlNode() && m.encryptor != nil {
		key, err := m.encryptor.GenerateHashRatchetKey(community.ID())
		if err != nil {
			return nil, nil, err
		}
		keyID, err := key.GetKeyID()
		if err != nil {
			return nil, nil, err
		}
		m.logger.Info("generate key for token", zap.String("group-id", types.Bytes2Hex(community.ID())), zap.String("key-id", types.Bytes2Hex(keyID)))
	}

	community, changes, err := m.createCommunityTokenPermission(request, community)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) EditCommunityTokenPermission(request *requests.EditCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	tokenPermission := request.ToCommunityTokenPermission()

	changes, err := community.UpsertTokenPermission(&tokenPermission)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

type reevaluateMemberRole struct {
	old protobuf.CommunityMember_Roles
	new protobuf.CommunityMember_Roles
}

func (rmr reevaluateMemberRole) hasChanged() bool {
	return rmr.old != rmr.new
}

func (rmr reevaluateMemberRole) isPrivileged() bool {
	return rmr.new != protobuf.CommunityMember_ROLE_NONE
}

func (rmr reevaluateMemberRole) hasChangedToPrivileged() bool {
	return rmr.hasChanged() && rmr.old == protobuf.CommunityMember_ROLE_NONE
}

func (rmr reevaluateMemberRole) hasChangedPrivilegedRole() bool {
	return (rmr.old == protobuf.CommunityMember_ROLE_ADMIN && rmr.new == protobuf.CommunityMember_ROLE_TOKEN_MASTER) ||
		(rmr.old == protobuf.CommunityMember_ROLE_TOKEN_MASTER && rmr.new == protobuf.CommunityMember_ROLE_ADMIN)
}

type reevaluateMembersResult struct {
	membersToRemove             map[string]struct{}
	membersRoles                map[string]*reevaluateMemberRole
	membersToRemoveFromChannels map[string]map[string]struct{}
	membersToAddToChannels      map[string]map[string]protobuf.CommunityMember_ChannelRole
}

func (rmr *reevaluateMembersResult) newPrivilegedRoles() (map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey, error) {
	result := map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey{}

	for memberKey, roles := range rmr.membersRoles {
		if roles.hasChangedToPrivileged() || roles.hasChangedPrivilegedRole() {
			memberPubKey, err := common.HexToPubkey(memberKey)
			if err != nil {
				return nil, err
			}
			if result[roles.new] == nil {
				result[roles.new] = []*ecdsa.PublicKey{}
			}
			result[roles.new] = append(result[roles.new], memberPubKey)
		}
	}

	return result, nil
}

// Fetch all owners for all collectibles.
func (m *Manager) fetchCollectiblesOwners(collectibles map[walletcommon.ChainID]map[gethcommon.Address]struct{}) (CollectiblesOwners, error) {
	if m.collectiblesManager == nil {
		return nil, errors.New("no collectibles manager")
	}

	collectiblesOwners := make(CollectiblesOwners)
	for chainID, contractAddresses := range collectibles {
		collectiblesOwners[chainID] = make(map[gethcommon.Address]*thirdparty.CollectibleContractOwnership)

		for contractAddress := range contractAddresses {
			ownership, err := m.collectiblesManager.FetchCollectibleOwnersByContractAddress(context.Background(), chainID, contractAddress)
			if err != nil {
				return nil, err
			}
			collectiblesOwners[chainID][contractAddress] = ownership
		}
	}
	return collectiblesOwners, nil
}

// use it only for testing purposes
func (m *Manager) ReevaluateMembers(communityID types.HexBytes) (*Community, map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey, error) {
	return m.reevaluateMembers(communityID)
}

// First, the community is read from the database,
// then the members are reevaluated, and only then
// the community is locked and changes are applied.
// NOTE: Changes made to the same community
// while reevaluation is ongoing are respected
// and do not affect the result of this function.
// If permissions are changed in the meantime,
// they will be accommodated with the next reevaluation.
func (m *Manager) reevaluateMembers(communityID types.HexBytes) (*Community, map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}

	if !community.IsControlNode() {
		return nil, nil, ErrNotEnoughPermissions
	}

	communityPermissionsPreParsedData, channelPermissionsPreParsedData := PreParsePermissionsData(community.tokenPermissions())

	// Optimization: Fetch all collectibles owners before members iteration to avoid asking providers for the same collectibles.
	collectiblesOwners, err := m.fetchCollectiblesOwners(CollectibleAddressesFromPreParsedPermissionsData(communityPermissionsPreParsedData, channelPermissionsPreParsedData))
	if err != nil {
		return nil, nil, err
	}

	result := &reevaluateMembersResult{
		membersToRemove:             map[string]struct{}{},
		membersRoles:                map[string]*reevaluateMemberRole{},
		membersToRemoveFromChannels: map[string]map[string]struct{}{},
		membersToAddToChannels:      map[string]map[string]protobuf.CommunityMember_ChannelRole{},
	}

	membersAccounts, err := m.persistence.GetCommunityRequestsToJoinRevealedAddresses(community.ID())
	if err != nil {
		return nil, nil, err
	}

	for memberKey := range community.Members() {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, nil, err
		}

		if memberKey == common.PubkeyToHex(&m.identity.PublicKey) || community.IsMemberOwner(memberPubKey) {
			continue
		}

		revealedAccount, memberHasWallet := membersAccounts[memberKey]
		if !memberHasWallet {
			result.membersToRemove[memberKey] = struct{}{}
			continue
		}

		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(revealedAccount)

		result.membersRoles[memberKey] = &reevaluateMemberRole{
			old: community.MemberRole(memberPubKey),
			new: protobuf.CommunityMember_ROLE_NONE,
		}

		becomeTokenMasterPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER]
		if becomeTokenMasterPermissions != nil {
			permissionResponse, err := m.PermissionChecker.CheckPermissionsWithPreFetchedData(becomeTokenMasterPermissions, accountsAndChainIDs, true, collectiblesOwners)
			if err != nil {
				return nil, nil, err
			}

			if permissionResponse.Satisfied {
				result.membersRoles[memberKey].new = protobuf.CommunityMember_ROLE_TOKEN_MASTER
				// Skip further validation if user has TokenMaster permissions
				continue
			}
		}

		becomeAdminPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_ADMIN]
		if becomeAdminPermissions != nil {
			permissionResponse, err := m.PermissionChecker.CheckPermissionsWithPreFetchedData(becomeAdminPermissions, accountsAndChainIDs, true, collectiblesOwners)
			if err != nil {
				return nil, nil, err
			}

			if permissionResponse.Satisfied {
				result.membersRoles[memberKey].new = protobuf.CommunityMember_ROLE_ADMIN
				// Skip further validation if user has Admin permissions
				continue
			}
		}

		becomeMemberPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_MEMBER]
		if becomeMemberPermissions != nil {
			permissionResponse, err := m.PermissionChecker.CheckPermissionsWithPreFetchedData(becomeMemberPermissions, accountsAndChainIDs, true, collectiblesOwners)
			if err != nil {
				return nil, nil, err
			}

			if !permissionResponse.Satisfied {
				result.membersToRemove[memberKey] = struct{}{}
				// Skip channels validation if user has been removed
				continue
			}
		}

		addToChannels, removeFromChannels, err := m.reevaluateMemberChannelsPermissions(community, memberPubKey, channelPermissionsPreParsedData, accountsAndChainIDs, collectiblesOwners)
		if err != nil {
			return nil, nil, err
		}
		result.membersToAddToChannels[memberKey] = addToChannels
		result.membersToRemoveFromChannels[memberKey] = removeFromChannels
	}

	newPrivilegedRoles, err := result.newPrivilegedRoles()
	if err != nil {
		return nil, nil, err
	}

	// Note: community itself may have changed in the meantime of permissions reevaluation.
	community, err = m.applyReevaluateMembersResult(communityID, result)
	if err != nil {
		return nil, nil, err
	}

	return community, newPrivilegedRoles, m.saveAndPublish(community)
}

// Apply results on the most up-to-date community.
func (m *Manager) applyReevaluateMembersResult(communityID types.HexBytes, result *reevaluateMembersResult) (*Community, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	if !community.IsControlNode() {
		return nil, ErrNotEnoughPermissions
	}

	// Remove members.
	for memberKey := range result.membersToRemove {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, err
		}
		_, err = community.RemoveUserFromOrg(memberPubKey)
		if err != nil {
			return nil, err
		}
	}

	// Ensure members have proper roles.
	for memberKey, roles := range result.membersRoles {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, err
		}

		if !community.HasMember(memberPubKey) {
			continue
		}

		_, err = community.SetRoleToMember(memberPubKey, roles.new)
		if err != nil {
			return nil, err
		}

		// Ensure privileged members can post in all chats.
		if roles.isPrivileged() {
			for channelID := range community.Chats() {
				_, err = community.AddMemberToChat(channelID, memberPubKey, []protobuf.CommunityMember_Roles{roles.new}, protobuf.CommunityMember_CHANNEL_ROLE_POSTER)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Remove members from channels.
	for memberKey, channels := range result.membersToRemoveFromChannels {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, err
		}

		for channelID := range channels {
			_, err = community.RemoveUserFromChat(memberPubKey, channelID)
			if err != nil {
				return nil, err
			}
		}
	}

	// Add unprivileged members to channels.
	for memberKey, channels := range result.membersToAddToChannels {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, err
		}

		if !community.HasMember(memberPubKey) {
			continue
		}

		for channelID, channelRole := range channels {
			_, err = community.AddMemberToChat(channelID, memberPubKey, []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_NONE}, channelRole)
			if err != nil {
				return nil, err
			}
		}
	}

	return community, nil
}

func (m *Manager) reevaluateMemberChannelsPermissions(community *Community, memberPubKey *ecdsa.PublicKey,
	channelPermissionsPreParsedData map[string]*PreParsedCommunityPermissionsData, accountsAndChainIDs []*AccountChainIDsCombination, collectiblesOwners CollectiblesOwners) (map[string]protobuf.CommunityMember_ChannelRole, map[string]struct{}, error) {

	addToChannels := map[string]protobuf.CommunityMember_ChannelRole{}
	removeFromChannels := map[string]struct{}{}

	// check which permissions we satisfy and which not
	channelPermissionsCheckResult, err := m.checkChannelsPermissionsWithPreFetchedData(channelPermissionsPreParsedData, accountsAndChainIDs, true, collectiblesOwners)
	if err != nil {
		return nil, nil, err
	}

	for channelID := range community.Chats() {
		channelPermissionsCheckResult, hasChannelPermission := channelPermissionsCheckResult[community.ChatID(channelID)]

		// ensure member is added if channel has no permissions
		if !hasChannelPermission {
			addToChannels[channelID] = protobuf.CommunityMember_CHANNEL_ROLE_POSTER
			continue
		}

		viewAndPostSatisfied, viewAndPostPermissionExists := channelPermissionsCheckResult[protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL]
		viewOnlySatisfied, viewOnlyPermissionExists := channelPermissionsCheckResult[protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL]

		satisfied := false
		channelRole := protobuf.CommunityMember_CHANNEL_ROLE_VIEWER
		if viewAndPostPermissionExists && viewAndPostSatisfied {
			satisfied = viewAndPostSatisfied
			channelRole = protobuf.CommunityMember_CHANNEL_ROLE_POSTER
		} else if !satisfied && viewOnlyPermissionExists {
			satisfied = viewOnlySatisfied
		}

		if satisfied {
			addToChannels[channelID] = channelRole
		} else {
			removeFromChannels[channelID] = struct{}{}
		}
	}

	return addToChannels, removeFromChannels, nil
}

func (m *Manager) checkChannelsPermissionsImpl(channelsPermissionsPreParsedData map[string]*PreParsedCommunityPermissionsData, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool, collectiblesOwners CollectiblesOwners) (map[string]map[protobuf.CommunityTokenPermission_Type]bool, error) {
	checkPermissions := func(channelsPermissionPreParsedData *PreParsedCommunityPermissionsData) (*CheckPermissionsResponse, error) {
		if collectiblesOwners != nil {
			return m.PermissionChecker.CheckPermissionsWithPreFetchedData(channelsPermissionPreParsedData, accountsAndChainIDs, true, collectiblesOwners)
		} else {
			return m.PermissionChecker.CheckPermissions(channelsPermissionPreParsedData, accountsAndChainIDs, true)
		}
	}

	channelPermissionsCheckResult := make(map[string]map[protobuf.CommunityTokenPermission_Type]bool)
	for _, channelsPermissionPreParsedData := range channelsPermissionsPreParsedData {
		permissionResponse, err := checkPermissions(channelsPermissionPreParsedData)
		if err != nil {
			return channelPermissionsCheckResult, err
		}
		// Note: in `PreParsedCommunityPermissionsData` for channels there will be only one permission
		// no need to iterate over `Permissions`
		for _, chatId := range channelsPermissionPreParsedData.Permissions[0].ChatIds {
			if _, exists := channelPermissionsCheckResult[chatId]; !exists {
				channelPermissionsCheckResult[chatId] = make(map[protobuf.CommunityTokenPermission_Type]bool)
			}
			satisfied, exists := channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type]
			if exists && satisfied {
				continue
			}
			channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type] = permissionResponse.Satisfied
		}
	}
	return channelPermissionsCheckResult, nil
}

func (m *Manager) checkChannelsPermissionsWithPreFetchedData(channelsPermissionsPreParsedData map[string]*PreParsedCommunityPermissionsData, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool, collectiblesOwners CollectiblesOwners) (map[string]map[protobuf.CommunityTokenPermission_Type]bool, error) {
	return m.checkChannelsPermissionsImpl(channelsPermissionsPreParsedData, accountsAndChainIDs, shortcircuit, collectiblesOwners)
}

func (m *Manager) checkChannelsPermissions(channelsPermissionsPreParsedData map[string]*PreParsedCommunityPermissionsData, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (map[string]map[protobuf.CommunityTokenPermission_Type]bool, error) {
	return m.checkChannelsPermissionsImpl(channelsPermissionsPreParsedData, accountsAndChainIDs, shortcircuit, nil)
}

func (m *Manager) StartMembersReevaluationLoop(communityID types.HexBytes, reevaluateOnStart bool) {
	go m.reevaluateMembersLoop(communityID, reevaluateOnStart)
}

func (m *Manager) reevaluateMembersLoop(communityID types.HexBytes, reevaluateOnStart bool) {
	defer utils.LogOnPanic()
	if _, exists := m.membersReevaluationTasks.Load(communityID.String()); exists {
		return
	}

	m.membersReevaluationTasks.Store(communityID.String(), &membersReevaluationTask{})
	defer m.membersReevaluationTasks.Delete(communityID.String())

	var forceReevaluation chan struct{}
	if m.forceMembersReevaluation != nil {
		forceReevaluation = make(chan struct{}, 10)
		m.forceMembersReevaluation[communityID.String()] = forceReevaluation
	}

	type criticalError struct {
		error
	}

	shouldReevaluate := func(task *membersReevaluationTask, force bool) bool {
		task.mutex.Lock()
		defer task.mutex.Unlock()

		// Ensure reevaluation is performed not more often than once per 5 minutes
		if !force && task.lastSuccessTime.After(time.Now().Add(-5*time.Minute)) {
			return false
		}

		if !task.lastSuccessTime.Before(time.Now().Add(-memberPermissionsCheckInterval)) &&
			!task.lastStartTime.Before(task.onDemandRequestTime) {
			return false
		}

		return true
	}

	reevaluateMembers := func(force bool) (err error) {
		t, exists := m.membersReevaluationTasks.Load(communityID.String())
		if !exists {
			return criticalError{
				error: errors.New("missing task"),
			}
		}
		task, ok := t.(*membersReevaluationTask)
		if !ok {
			return criticalError{
				error: errors.New("invalid task type"),
			}
		}

		if !shouldReevaluate(task, force) {
			return nil
		}

		task.mutex.Lock()
		task.lastStartTime = time.Now()
		task.mutex.Unlock()

		err = m.reevaluateCommunityMembersPermissions(communityID)
		if err != nil {
			if errors.Is(err, ErrOrgNotFound) {
				return criticalError{
					error: err,
				}
			}
			return err
		}

		task.mutex.Lock()
		task.lastSuccessTime = time.Now()
		task.mutex.Unlock()

		m.logger.Info("reevaluation finished", zap.String("communityID", communityID.String()), zap.Duration("elapsed", task.lastSuccessTime.Sub(task.lastStartTime)))
		return nil
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	reevaluate := reevaluateOnStart
	force := false

	for {
		if reevaluate {
			err := reevaluateMembers(force)
			if err != nil {
				var criticalError *criticalError
				if errors.As(err, &criticalError) {
					return
				}
			}
		}

		force = false
		reevaluate = false

		select {
		case <-ticker.C:
			reevaluate = true
			continue

		case <-forceReevaluation:
			reevaluate = true
			force = true
			continue

		case <-m.quit:
			return
		}
	}
}

func (m *Manager) ForceMembersReevaluation(communityID types.HexBytes) error {
	if m.forceMembersReevaluation == nil {
		return errors.New("forcing members reevaluation is not allowed")
	}
	return m.scheduleMembersReevaluation(communityID, true)
}

func (m *Manager) ScheduleMembersReevaluation(communityID types.HexBytes) error {
	return m.scheduleMembersReevaluation(communityID, false)
}

func (m *Manager) scheduleMembersReevaluation(communityID types.HexBytes, forceImmediateReevaluation bool) error {
	t, exists := m.membersReevaluationTasks.Load(communityID.String())
	if !exists {
		// No reevaluation task yet. We start the loop which will create it
		m.StartMembersReevaluationLoop(communityID, true)
		return nil
	}

	task, ok := t.(*membersReevaluationTask)
	if !ok {
		return errors.New("invalid task type")
	}
	task.mutex.Lock()
	defer task.mutex.Unlock()
	task.onDemandRequestTime = time.Now()

	if forceImmediateReevaluation {
		m.forceMembersReevaluation[communityID.String()] <- struct{}{}
	}

	return nil
}

func (m *Manager) DeleteCommunityTokenPermission(request *requests.DeleteCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	changes, err := community.DeleteTokenPermission(request.PermissionID)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) reevaluateCommunityMembersPermissions(communityID types.HexBytes) error {
	// Publish when the reevluation started since it can take a while
	signal.SendCommunityMemberReevaluationStarted(types.EncodeHex(communityID))

	community, newPrivilegedMembers, err := m.reevaluateMembers(communityID)

	// Publish the reevaluation ending, even if it errored
	// A possible improvement would be to pass the error here
	signal.SendCommunityMemberReevaluationEnded(types.EncodeHex(communityID))

	if err != nil {
		return err
	}

	return m.ShareRequestsToJoinWithPrivilegedMembers(community, newPrivilegedMembers)
}

func (m *Manager) DeleteCommunity(id types.HexBytes) error {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	err := m.persistence.DeleteCommunity(id)
	if err != nil {
		return err
	}
	return m.persistence.DeleteCommunitySettings(id)
}

func (m *Manager) updateShard(community *Community, shard *shard.Shard, clock uint64) error {
	community.config.Shard = shard
	if shard == nil {
		return m.persistence.DeleteCommunityShard(community.ID())
	}

	return m.persistence.SaveCommunityShard(community.ID(), shard, clock)
}

func (m *Manager) UpdateShard(community *Community, shard *shard.Shard, clock uint64) error {
	m.communityLock.Lock(community.ID())
	defer m.communityLock.Unlock(community.ID())

	return m.updateShard(community, shard, clock)
}

// SetShard assigns a shard to a community
func (m *Manager) SetShard(communityID types.HexBytes, shard *shard.Shard) (*Community, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	community.increaseClock()

	err = m.updateShard(community, shard, community.Clock())
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) UpdatePubsubTopicPrivateKey(topic string, privKey *ecdsa.PrivateKey) error {
	if privKey != nil {
		return m.transport.StorePubsubTopicKey(topic, privKey)
	}

	return m.transport.RemovePubsubTopicKey(topic)
}

// EditCommunity takes a description, updates the community with the description,
// saves it and returns it
func (m *Manager) EditCommunity(request *requests.EditCommunity) (*Community, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, err
	}

	newDescription, err := request.ToCommunityDescription()
	if err != nil {
		return nil, fmt.Errorf("can't create community description: %v", err)
	}

	// If permissions weren't explicitly set on original request, use existing ones
	if newDescription.Permissions.Access == protobuf.CommunityPermissions_UNKNOWN_ACCESS {
		newDescription.Permissions.Access = community.config.CommunityDescription.Permissions.Access
	}
	// Use existing images for the entries that were not updated
	// NOTE: This will NOT allow deletion of the community image; it will need to
	// be handled separately.
	for imageName := range community.config.CommunityDescription.Identity.Images {
		_, exists := newDescription.Identity.Images[imageName]
		if !exists {
			// If no image was set in ToCommunityDescription then Images is nil.
			if newDescription.Identity.Images == nil {
				newDescription.Identity.Images = make(map[string]*protobuf.IdentityImage)
			}
			newDescription.Identity.Images[imageName] = community.config.CommunityDescription.Identity.Images[imageName]
		}
	}
	// TODO: handle delete image (if needed)

	err = ValidateCommunityDescription(newDescription)
	if err != nil {
		return nil, err
	}

	if !(community.IsControlNode() || community.hasPermissionToSendCommunityEvent(protobuf.CommunityEvent_COMMUNITY_EDIT)) {
		return nil, ErrNotAuthorized
	}

	// Edit the community values
	community.Edit(newDescription)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() {
		community.increaseClock()
	} else {
		err := community.addNewCommunityEvent(community.ToCommunityEditCommunityEvent(newDescription))
		if err != nil {
			return nil, err
		}
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) RemovePrivateKey(id types.HexBytes) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return community, err
	}

	if !community.IsControlNode() {
		return community, ErrNotControlNode
	}

	community.config.PrivateKey = nil
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return community, err
	}
	return community, nil
}

func (m *Manager) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !community.IsControlNode() {
		return nil, ErrNotControlNode
	}

	return community.config.PrivateKey, nil
}

func (m *Manager) ImportCommunity(key *ecdsa.PrivateKey, clock uint64) (*Community, error) {
	communityID := crypto.CompressPubkey(&key.PublicKey)

	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil && err != ErrOrgNotFound {
		return nil, err
	}

	if community == nil {
		createCommunityRequest := requests.CreateCommunity{
			Membership: protobuf.CommunityPermissions_MANUAL_ACCEPT,
			Name:       "unknown imported",
		}

		description, err := createCommunityRequest.ToCommunityDescription()
		if err != nil {
			return nil, err
		}

		err = ValidateCommunityDescription(description)
		if err != nil {
			return nil, err
		}

		description.Clock = 1
		description.ID = types.EncodeHex(communityID)

		config := Config{
			ID:                   &key.PublicKey,
			PrivateKey:           key,
			ControlNode:          &key.PublicKey,
			ControlDevice:        true,
			Logger:               m.logger,
			Joined:               true,
			JoinedAt:             time.Now().Unix(),
			MemberIdentity:       m.identity,
			CommunityDescription: description,
			LastOpenedAt:         0,
		}

		var descriptionEncryptor DescriptionEncryptor
		if m.encryptor != nil {
			descriptionEncryptor = m
		}
		community, err = New(config, m.timesource, descriptionEncryptor, m.mediaServer)
		if err != nil {
			return nil, err
		}
	} else {
		community.config.PrivateKey = key
		community.config.ControlDevice = true
	}

	community.Join()
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	// Save grant for own community
	grant, err := community.BuildGrant(&m.identity.PublicKey, "")
	if err != nil {
		return nil, err
	}
	err = m.persistence.SaveCommunityGrant(community.IDString(), grant, uint64(time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	// Mark this device as the control node
	syncControlNode := &protobuf.SyncCommunityControlNode{
		Clock:          clock,
		InstallationId: m.installationID,
	}
	err = m.SaveSyncControlNode(community.ID(), syncControlNode)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) CreateChat(communityID types.HexBytes, chat *protobuf.CommunityChat, publish bool, thirdPartyID string) (*CommunityChanges, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	chatID := uuid.New().String()
	if thirdPartyID != "" {
		chatID = chatID + thirdPartyID
	}

	changes, err := community.CreateChat(chatID, chat)
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func (m *Manager) EditChat(communityID types.HexBytes, chatID string, chat *protobuf.CommunityChat) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(chatID, communityID.String()) {
		chatID = strings.TrimPrefix(chatID, communityID.String())
	}

	oldChat, err := community.GetChat(chatID)
	if err != nil {
		return nil, nil, err
	}

	// We can't edit permissions and members with an Edit, so we set to what we had, otherwise they will be lost
	chat.Permissions = oldChat.Permissions
	chat.Members = oldChat.Members

	changes, err := community.EditChat(chatID, chat)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) DeleteChat(communityID types.HexBytes, chatID string) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}

	// Check for channel permissions
	changes := community.emptyCommunityChanges()
	for tokenPermissionID, tokenPermission := range community.tokenPermissions() {
		chats := tokenPermission.ChatIdsAsMap()
		_, hasChat := chats[chatID]
		if !hasChat {
			continue
		}

		if len(chats) == 1 {
			// Delete channel permission, if there is only one channel
			deletePermissionChanges, err := community.DeleteTokenPermission(tokenPermissionID)
			if err != nil {
				return nil, nil, err
			}
			changes.Merge(deletePermissionChanges)
		} else {
			// Remove the channel from the permission, if there are other channels
			delete(chats, chatID)

			var chatIDs []string
			for chatID := range chats {
				chatIDs = append(chatIDs, chatID)
			}
			tokenPermission.ChatIds = chatIDs

			updatePermissionChanges, err := community.UpsertTokenPermission(tokenPermission.CommunityTokenPermission)
			if err != nil {
				return nil, nil, err
			}
			changes.Merge(updatePermissionChanges)
		}
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(chatID, communityID.String()) {
		chatID = strings.TrimPrefix(chatID, communityID.String())
	}

	deleteChanges, err := community.DeleteChat(chatID)
	if err != nil {
		return nil, nil, err
	}
	changes.Merge(deleteChanges)

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) CreateCategory(request *requests.CreateCommunityCategory, publish bool) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	categoryID := uuid.New().String()
	if request.ThirdPartyID != "" {
		categoryID = categoryID + request.ThirdPartyID
	}

	// Remove communityID prefix from chatID if exists
	for i, cid := range request.ChatIDs {
		if strings.HasPrefix(cid, request.CommunityID.String()) {
			request.ChatIDs[i] = strings.TrimPrefix(cid, request.CommunityID.String())
		}
	}

	changes, err := community.CreateCategory(categoryID, request.CategoryName, request.ChatIDs)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) EditCategory(request *requests.EditCommunityCategory) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	// Remove communityID prefix from chatID if exists
	for i, cid := range request.ChatIDs {
		if strings.HasPrefix(cid, request.CommunityID.String()) {
			request.ChatIDs[i] = strings.TrimPrefix(cid, request.CommunityID.String())
		}
	}

	changes, err := community.EditCategory(request.CategoryID, request.CategoryName, request.ChatIDs)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) EditChatFirstMessageTimestamp(communityID types.HexBytes, chatID string, timestamp uint32) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(chatID, communityID.String()) {
		chatID = strings.TrimPrefix(chatID, communityID.String())
	}

	changes, err := community.UpdateChatFirstMessageTimestamp(chatID, timestamp)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	m.publish(&Subscription{Community: community})

	return community, changes, nil
}

func (m *Manager) UpdateChatFirstMessageTimestamp(community *Community, chatID string, timestamp uint32) (*CommunityChanges, error) {
	communityID := community.ID().String()
	chatID = strings.TrimPrefix(chatID, communityID)
	return community.UpdateChatFirstMessageTimestamp(chatID, timestamp)
}

func (m *Manager) ReorderCategories(request *requests.ReorderCommunityCategories) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	changes, err := community.ReorderCategories(request.CategoryID, request.Position)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) ReorderChat(request *requests.ReorderCommunityChat) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(request.ChatID, request.CommunityID.String()) {
		request.ChatID = strings.TrimPrefix(request.ChatID, request.CommunityID.String())
	}

	changes, err := community.ReorderChat(request.CategoryID, request.ChatID, request.Position)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil
}

func (m *Manager) DeleteCategory(request *requests.DeleteCommunityCategory) (*Community, *CommunityChanges, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	changes, err := community.DeleteCategory(request.CategoryID)
	if err != nil {
		return nil, nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, nil, err
	}

	return changes.Community, changes, nil
}

func (m *Manager) GenerateRequestsToJoinForAutoApprovalOnNewOwnership(communityID types.HexBytes, kickedMembers map[string]*protobuf.CommunityMember) ([]*RequestToJoin, error) {
	var requestsToJoin []*RequestToJoin
	clock := uint64(time.Now().Unix())
	for pubKeyStr := range kickedMembers {
		requestToJoin := &RequestToJoin{
			PublicKey:        pubKeyStr,
			Clock:            clock,
			CommunityID:      communityID,
			State:            RequestToJoinStateAwaitingAddresses,
			Our:              true,
			RevealedAccounts: make([]*protobuf.RevealedAccount, 0),
		}

		requestToJoin.CalculateID()

		requestsToJoin = append(requestsToJoin, requestToJoin)
	}

	return requestsToJoin, m.persistence.SaveRequestsToJoin(requestsToJoin)
}

func (m *Manager) Queue(signer *ecdsa.PublicKey, community *Community, clock uint64, payload []byte) error {

	m.logger.Info("queuing community", zap.String("id", community.IDString()), zap.String("signer", common.PubkeyToHex(signer)))

	communityToValidate := communityToValidate{
		id:         community.ID(),
		clock:      clock,
		payload:    payload,
		validateAt: uint64(time.Now().UnixNano()),
		signer:     crypto.CompressPubkey(signer),
	}
	err := m.persistence.SaveCommunityToValidate(communityToValidate)
	if err != nil {
		m.logger.Error("failed to save community", zap.Error(err))
		return err
	}

	return nil
}

func (m *Manager) HandleCommunityDescriptionMessage(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, payload []byte, verifiedOwner *ecdsa.PublicKey, communityShard *protobuf.Shard) (*CommunityResponse, error) {
	m.logger.Debug("HandleCommunityDescriptionMessage", zap.String("communityID", description.ID), zap.Uint64("clock", description.Clock))

	if signer == nil {
		return nil, errors.New("signer can't be nil")
	}

	var id []byte
	var err error
	if len(description.ID) != 0 {
		id, err = types.DecodeHex(description.ID)
		if err != nil {
			return nil, err
		}
	} else {
		// Backward compatibility
		id = crypto.CompressPubkey(signer)
	}

	failedToDecrypt, processedDescription, err := m.preprocessDescription(id, description)
	if err != nil {
		return nil, err
	}
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)
	community, err := m.GetByID(id)
	if err != nil && err != ErrOrgNotFound {
		return nil, err
	}

	// We don't process failed to decrypt if the whole metadata is encrypted
	// and we joined the community already
	if community != nil && community.Joined() && len(failedToDecrypt) != 0 && processedDescription != nil && len(processedDescription.Members) == 0 {
		return &CommunityResponse{FailedToDecrypt: failedToDecrypt}, nil
	}

	// We should queue only if the community has a token owner, and the owner has been verified
	hasTokenOwnership := HasTokenOwnership(processedDescription)
	shouldQueue := hasTokenOwnership && verifiedOwner == nil

	if community == nil {
		pubKey, err := crypto.DecompressPubkey(id)
		if err != nil {
			return nil, err
		}
		var cShard *shard.Shard
		if communityShard == nil {
			cShard = &shard.Shard{Cluster: shard.MainStatusShardCluster, Index: shard.DefaultShardIndex}
		} else {
			cShard = shard.FromProtobuff(communityShard)
		}
		config := Config{
			CommunityDescription:                processedDescription,
			Logger:                              m.logger,
			CommunityDescriptionProtocolMessage: payload,
			MemberIdentity:                      m.identity,
			ID:                                  pubKey,
			ControlNode:                         signer,
			Shard:                               cShard,
		}

		var descriptionEncryptor DescriptionEncryptor
		if m.encryptor != nil {
			descriptionEncryptor = m
		}
		community, err = New(config, m.timesource, descriptionEncryptor, m.mediaServer)
		if err != nil {
			return nil, err
		}

		// A new community, we need to check if we need to validate async.
		// That would be the case if it has a contract. We queue everything and process separately.
		if shouldQueue {
			return nil, m.Queue(signer, community, processedDescription.Clock, payload)
		}
	} else {
		// only queue if already known control node is different than the signer
		// and if the clock is greater
		shouldQueue = shouldQueue && !common.IsPubKeyEqual(community.ControlNode(), signer) &&
			community.config.CommunityDescription.Clock < processedDescription.Clock
		if shouldQueue {
			return nil, m.Queue(signer, community, processedDescription.Clock, payload)
		}
	}

	if hasTokenOwnership && verifiedOwner != nil {
		// Override verified owner
		m.logger.Info("updating verified owner",
			zap.String("communityID", community.IDString()),
			zap.String("verifiedOwner", common.PubkeyToHex(verifiedOwner)),
			zap.String("signer", common.PubkeyToHex(signer)),
			zap.String("controlNode", common.PubkeyToHex(community.ControlNode())),
		)

		// If we are not the verified owner anymore, drop the private key
		if !common.IsPubKeyEqual(verifiedOwner, &m.identity.PublicKey) {
			community.config.PrivateKey = nil
		}

		// new control node will be set in the 'UpdateCommunityDescription'
		if !common.IsPubKeyEqual(verifiedOwner, signer) {
			return nil, ErrNotAuthorized
		}
	} else if !common.IsPubKeyEqual(community.ControlNode(), signer) {
		return nil, ErrNotAuthorized
	}

	r, err := m.handleCommunityDescriptionMessageCommon(community, processedDescription, payload, verifiedOwner)
	if err != nil {
		return nil, err
	}
	r.FailedToDecrypt = failedToDecrypt
	return r, nil
}

func (m *Manager) NewHashRatchetKeys(keys []*encryption.HashRatchetInfo) error {
	return m.persistence.InvalidateDecryptedCommunityCacheForKeys(keys)
}

func (m *Manager) preprocessDescription(id types.HexBytes, description *protobuf.CommunityDescription) ([]*CommunityPrivateDataFailedToDecrypt, *protobuf.CommunityDescription, error) {
	decryptedCommunity, err := m.persistence.GetDecryptedCommunityDescription(id, description.Clock)
	if err != nil {
		return nil, nil, err
	}
	if decryptedCommunity != nil {
		return nil, decryptedCommunity, nil
	}

	response, err := decryptDescription(id, m, description, m.logger)
	if err != nil {
		return response, description, err
	}

	upgradeTokenPermissions(description)

	// Workaround for https://github.com/status-im/status-desktop/issues/12188
	hydrateChannelsMembers(description)

	return response, description, m.persistence.SaveDecryptedCommunityDescription(id, response, description)
}

func (m *Manager) handleCommunityDescriptionMessageCommon(community *Community, description *protobuf.CommunityDescription, payload []byte, newControlNode *ecdsa.PublicKey) (*CommunityResponse, error) {
	prevClock := community.config.CommunityDescription.Clock
	prevResendAccountsClock := community.config.CommunityDescription.ResendAccountsClock

	changes, err := community.UpdateCommunityDescription(description, payload, newControlNode)
	if err != nil {
		return nil, err
	}

	if err = m.handleCommunityTokensMetadata(community); err != nil {
		return nil, err
	}

	hasCommunityArchiveInfo, err := m.persistence.HasCommunityArchiveInfo(community.ID())
	if err != nil {
		return nil, err
	}

	cdMagnetlinkClock := community.config.CommunityDescription.ArchiveMagnetlinkClock
	if !hasCommunityArchiveInfo {
		err = m.persistence.SaveCommunityArchiveInfo(community.ID(), cdMagnetlinkClock, 0)
		if err != nil {
			return nil, err
		}
	} else {
		magnetlinkClock, err := m.persistence.GetMagnetlinkMessageClock(community.ID())
		if err != nil {
			return nil, err
		}
		if cdMagnetlinkClock > magnetlinkClock {
			err = m.persistence.UpdateMagnetlinkMessageClock(community.ID(), cdMagnetlinkClock)
			if err != nil {
				return nil, err
			}
		}
	}

	pkString := common.PubkeyToHex(&m.identity.PublicKey)
	if m.tokenManager != nil && description.CommunityTokensMetadata != nil && len(description.CommunityTokensMetadata) > 0 {
		for _, tokenMetadata := range description.CommunityTokensMetadata {
			if tokenMetadata.TokenType != protobuf.CommunityTokenType_ERC20 {
				continue
			}

			for chainID, address := range tokenMetadata.ContractAddresses {
				_ = m.tokenManager.FindOrCreateTokenByAddress(context.Background(), chainID, gethcommon.HexToAddress(address))
			}
		}
	}

	// If the community require membership, we set whether we should leave/join the community after a state change
	if community.ManualAccept() || community.AutoAccept() {
		if changes.HasNewMember(pkString) {
			hasPendingRequest, err := m.persistence.HasPendingRequestsToJoinForUserAndCommunity(pkString, changes.Community.ID())
			if err != nil {
				return nil, err
			}
			// If there's any pending request, we should join the community
			// automatically
			changes.ShouldMemberJoin = hasPendingRequest
		}

		if changes.HasMemberLeft(pkString) && community.Joined() {
			softKick := !changes.IsMemberBanned(pkString) &&
				(changes.ControlNodeChanged != nil || prevResendAccountsClock < community.Description().ResendAccountsClock)

			// If we joined previously the community, that means we have been kicked
			changes.MemberKicked = !softKick
			// soft kick previously joined member on community owner change or on ResendAccountsClock change
			changes.MemberSoftKicked = softKick
		}
	}

	if description.Clock > prevClock {
		err = m.persistence.DeleteCommunityEvents(community.ID())
		if err != nil {
			return nil, err
		}
		community.config.EventsData = nil
	}

	// Set Joined if we are part of the member list
	if !community.Joined() && community.hasMember(&m.identity.PublicKey) {
		changes.ShouldMemberJoin = true
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	// We mark our requests as completed, though maybe we should mark
	// any request for any user that has been added as completed
	if err := m.markRequestToJoinAsAccepted(&m.identity.PublicKey, community); err != nil {
		return nil, err
	}
	// Check if there's a change and we should be joining

	return &CommunityResponse{
		Community: community,
		Changes:   changes,
	}, nil
}

func (m *Manager) signEvents(community *Community) error {
	for i := range community.config.EventsData.Events {
		communityEvent := &community.config.EventsData.Events[i]
		if communityEvent.Signature == nil || len(communityEvent.Signature) == 0 {
			err := communityEvent.Sign(m.identity)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) HandleCommunityEventsMessage(signer *ecdsa.PublicKey, message *protobuf.CommunityEventsMessage) (*CommunityResponse, error) {
	if signer == nil {
		return nil, errors.New("signer can't be nil")
	}

	eventsMessage, err := CommunityEventsMessageFromProtobuf(message)
	if err != nil {
		return nil, err
	}

	m.communityLock.Lock(eventsMessage.CommunityID)
	defer m.communityLock.Unlock(eventsMessage.CommunityID)

	community, err := m.GetByID(eventsMessage.CommunityID)
	if err != nil {
		return nil, err
	}

	if !community.IsPrivilegedMember(signer) {
		return nil, errors.New("user has not permissions to send events")
	}

	originCommunity := community.CreateDeepCopy()

	var lastlyAppliedEvents map[string]uint64
	if community.IsControlNode() {
		lastlyAppliedEvents, err = m.persistence.GetAppliedCommunityEvents(community.ID())
		if err != nil {
			return nil, err
		}
	}

	additionalCommunityResponse, err := m.handleCommunityEventsAndMetadata(community, eventsMessage, lastlyAppliedEvents)
	if err != nil {
		return nil, err
	}

	// Control node applies events and publish updated CommunityDescription
	if community.IsControlNode() {
		appliedEvents := map[string]uint64{}
		if community.config.EventsData != nil {
			for _, event := range community.config.EventsData.Events {
				appliedEvents[event.EventTypeID()] = event.CommunityEventClock
			}
		}
		community.config.EventsData = nil // clear events, they are already applied
		community.increaseClock()

		if m.keyDistributor != nil {
			encryptionKeyActions := EvaluateCommunityEncryptionKeyActions(originCommunity, community)
			err := m.keyDistributor.Generate(community, encryptionKeyActions)
			if err != nil {
				return nil, err
			}
		}

		err = m.persistence.SaveCommunity(community)
		if err != nil {
			return nil, err
		}

		err = m.persistence.UpsertAppliedCommunityEvents(community.ID(), appliedEvents)
		if err != nil {
			return nil, err
		}

		m.publish(&Subscription{Community: community})
	} else {
		err = m.persistence.SaveCommunity(community)
		if err != nil {
			return nil, err
		}
		err := m.persistence.SaveCommunityEvents(community)
		if err != nil {
			return nil, err
		}
	}

	return &CommunityResponse{
		Community:      community,
		Changes:        EvaluateCommunityChanges(originCommunity, community),
		RequestsToJoin: additionalCommunityResponse.RequestsToJoin,
	}, nil
}

func (m *Manager) handleAdditionalAdminChanges(community *Community) (*CommunityResponse, error) {
	communityResponse := CommunityResponse{
		RequestsToJoin: make([]*RequestToJoin, 0),
	}

	if !(community.IsControlNode() || community.HasPermissionToSendCommunityEvents()) {
		// we're a normal user/member node, so there's nothing for us to do here
		return &communityResponse, nil
	}

	if community.config.EventsData == nil {
		return &communityResponse, nil
	}

	handledMembers := map[string]struct{}{}

	for i := len(community.config.EventsData.Events) - 1; i >= 0; i-- {
		communityEvent := &community.config.EventsData.Events[i]
		if _, handled := handledMembers[communityEvent.MemberToAction]; handled {
			continue
		}
		switch communityEvent.Type {
		case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
			handledMembers[communityEvent.MemberToAction] = struct{}{}
			requestsToJoin, err := m.handleCommunityEventRequestAccepted(community, communityEvent)
			if err != nil {
				return nil, err
			}
			if requestsToJoin != nil {
				communityResponse.RequestsToJoin = append(communityResponse.RequestsToJoin, requestsToJoin...)
			}

		case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
			handledMembers[communityEvent.MemberToAction] = struct{}{}
			requestsToJoin, err := m.handleCommunityEventRequestRejected(community, communityEvent)
			if err != nil {
				return nil, err
			}
			if requestsToJoin != nil {
				communityResponse.RequestsToJoin = append(communityResponse.RequestsToJoin, requestsToJoin...)
			}

		default:
		}
	}
	return &communityResponse, nil
}

func (m *Manager) saveOrUpdateRequestToJoin(communityID types.HexBytes, requestToJoin *RequestToJoin) (bool, error) {
	updated := false

	existingRequestToJoin, err := m.persistence.GetRequestToJoin(requestToJoin.ID)
	if err != nil && err != sql.ErrNoRows {
		return updated, err
	}

	if existingRequestToJoin != nil {
		// node already knows about this request to join, so let's compare clocks
		// and update it if necessary
		if existingRequestToJoin.Clock <= requestToJoin.Clock {
			pk, err := common.HexToPubkey(existingRequestToJoin.PublicKey)
			if err != nil {
				return updated, err
			}
			err = m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), communityID, requestToJoin.State)
			if err != nil {
				return updated, err
			}
			updated = true
		}
	} else {
		err := m.persistence.SaveRequestToJoin(requestToJoin)
		if err != nil {
			return updated, err
		}
	}

	return updated, nil
}

func (m *Manager) handleCommunityEventRequestAccepted(community *Community, communityEvent *CommunityEvent) ([]*RequestToJoin, error) {
	acceptedRequestsToJoin := make([]types.HexBytes, 0)

	requestsToJoin := make([]*RequestToJoin, 0)

	signer := communityEvent.MemberToAction
	request := communityEvent.RequestToJoin

	requestToJoin := &RequestToJoin{
		PublicKey:          signer,
		Clock:              request.Clock,
		ENSName:            request.EnsName,
		CommunityID:        request.CommunityId,
		State:              RequestToJoinStateAcceptedPending,
		CustomizationColor: multiaccountscommon.IDToColorFallbackToBlue(request.CustomizationColor),
	}
	requestToJoin.CalculateID()

	existingRequestToJoin, err := m.persistence.GetRequestToJoin(requestToJoin.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if existingRequestToJoin != nil {
		alreadyProcessedByControlNode := existingRequestToJoin.State == RequestToJoinStateAccepted
		if alreadyProcessedByControlNode || existingRequestToJoin.State == RequestToJoinStateCanceled {
			return requestsToJoin, nil
		}
	}

	requestUpdated, err := m.saveOrUpdateRequestToJoin(community.ID(), requestToJoin)
	if err != nil {
		return nil, err
	}

	// If request to join exists in control node, add request to acceptedRequestsToJoin.
	// Otherwise keep the request as RequestToJoinStateAcceptedPending,
	// as privileged users don't have revealed addresses. This can happen if control node received
	// community event message before user request to join.
	if community.IsControlNode() && requestUpdated {
		acceptedRequestsToJoin = append(acceptedRequestsToJoin, requestToJoin.ID)
	}

	requestsToJoin = append(requestsToJoin, requestToJoin)

	if community.IsControlNode() {
		m.publish(&Subscription{AcceptedRequestsToJoin: acceptedRequestsToJoin})
	}
	return requestsToJoin, nil
}

func (m *Manager) handleCommunityEventRequestRejected(community *Community, communityEvent *CommunityEvent) ([]*RequestToJoin, error) {
	rejectedRequestsToJoin := make([]types.HexBytes, 0)

	requestsToJoin := make([]*RequestToJoin, 0)

	signer := communityEvent.MemberToAction
	request := communityEvent.RequestToJoin

	requestToJoin := &RequestToJoin{
		PublicKey:          signer,
		Clock:              request.Clock,
		ENSName:            request.EnsName,
		CommunityID:        request.CommunityId,
		State:              RequestToJoinStateDeclinedPending,
		CustomizationColor: multiaccountscommon.IDToColorFallbackToBlue(request.CustomizationColor),
	}
	requestToJoin.CalculateID()

	existingRequestToJoin, err := m.persistence.GetRequestToJoin(requestToJoin.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if existingRequestToJoin != nil {
		alreadyProcessedByControlNode := existingRequestToJoin.State == RequestToJoinStateDeclined
		if alreadyProcessedByControlNode || existingRequestToJoin.State == RequestToJoinStateCanceled {
			return requestsToJoin, nil
		}
	}

	requestUpdated, err := m.saveOrUpdateRequestToJoin(community.ID(), requestToJoin)
	if err != nil {
		return nil, err
	}
	// If request to join exists in control node, add request to rejectedRequestsToJoin.
	// Otherwise keep the request as RequestToJoinStateDeclinedPending,
	// as privileged users don't have revealed addresses. This can happen if control node received
	// community event message before user request to join.
	if community.IsControlNode() && requestUpdated {
		rejectedRequestsToJoin = append(rejectedRequestsToJoin, requestToJoin.ID)
	}

	requestsToJoin = append(requestsToJoin, requestToJoin)

	if community.IsControlNode() {
		m.publish(&Subscription{RejectedRequestsToJoin: rejectedRequestsToJoin})
	}
	return requestsToJoin, nil
}

// markRequestToJoinAsAccepted marks all the pending requests to join as completed
// if we are members
func (m *Manager) markRequestToJoinAsAccepted(pk *ecdsa.PublicKey, community *Community) error {
	if community.HasMember(pk) {
		return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateAccepted)
	}
	return nil
}

func (m *Manager) markRequestToJoinAsCanceled(pk *ecdsa.PublicKey, community *Community) error {
	return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateCanceled)
}

func (m *Manager) markRequestToJoinAsAcceptedPending(pk *ecdsa.PublicKey, community *Community) error {
	return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateAcceptedPending)
}

func (m *Manager) DeletePendingRequestToJoin(request *RequestToJoin) error {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return err
	}

	err = m.persistence.DeletePendingRequestToJoin(request.ID)
	if err != nil {
		return err
	}

	if community.IsControlNode() {
		err = m.saveAndPublish(community)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) UpdateClockInRequestToJoin(id types.HexBytes, clock uint64) error {
	return m.persistence.UpdateClockInRequestToJoin(id, clock)
}

func (m *Manager) SetMuted(id types.HexBytes, muted bool) error {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	return m.persistence.SetMuted(id, muted)
}

func (m *Manager) MuteCommunityTill(communityID []byte, muteTill time.Time) error {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	return m.persistence.MuteCommunityTill(communityID, muteTill)
}
func (m *Manager) CancelRequestToJoin(request *requests.CancelRequestToJoinCommunity) (*RequestToJoin, *Community, error) {
	dbRequest, err := m.persistence.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, nil, err
	}

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	pk, err := common.HexToPubkey(dbRequest.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	dbRequest.State = RequestToJoinStateCanceled
	if err := m.markRequestToJoinAsCanceled(pk, community); err != nil {
		return nil, nil, err
	}

	return dbRequest, community, nil
}

func (m *Manager) CheckPermissionToJoin(id []byte, addresses []gethcommon.Address) (*CheckPermissionToJoinResponse, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	return m.PermissionChecker.CheckPermissionToJoin(community, addresses)
}

func (m *Manager) accountsSatisfyPermissionsToJoin(
	communityPermissionsPreParsedData map[protobuf.CommunityTokenPermission_Type]*PreParsedCommunityPermissionsData,
	accountsAndChainIDs []*AccountChainIDsCombination) (bool, protobuf.CommunityMember_Roles, error) {

	if m.accountsHasPrivilegedPermission(communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER], accountsAndChainIDs) {
		return true, protobuf.CommunityMember_ROLE_TOKEN_MASTER, nil
	}
	if m.accountsHasPrivilegedPermission(communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_ADMIN], accountsAndChainIDs) {
		return true, protobuf.CommunityMember_ROLE_ADMIN, nil
	}

	preParsedBecomeMemberPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_MEMBER]
	if preParsedBecomeMemberPermissions != nil {
		permissionResponse, err := m.PermissionChecker.CheckPermissions(preParsedBecomeMemberPermissions, accountsAndChainIDs, true)
		if err != nil {
			return false, protobuf.CommunityMember_ROLE_NONE, err
		}

		return permissionResponse.Satisfied, protobuf.CommunityMember_ROLE_NONE, nil
	}

	return true, protobuf.CommunityMember_ROLE_NONE, nil
}

func (m *Manager) accountsSatisfyPermissionsToJoinChannels(
	community *Community,
	channelPermissionsPreParsedData map[string]*PreParsedCommunityPermissionsData,
	accountsAndChainIDs []*AccountChainIDsCombination) (map[string]*protobuf.CommunityChat, map[string]*protobuf.CommunityChat, error) {

	viewChats := make(map[string]*protobuf.CommunityChat)
	viewAndPostChats := make(map[string]*protobuf.CommunityChat)

	if len(channelPermissionsPreParsedData) == 0 {
		for channelID, channel := range community.config.CommunityDescription.Chats {
			viewAndPostChats[channelID] = channel
		}

		return viewChats, viewAndPostChats, nil
	}

	// check which permissions we satisfy and which not
	channelPermissionsCheckResult, err := m.checkChannelsPermissions(channelPermissionsPreParsedData, accountsAndChainIDs, true)
	if err != nil {
		m.logger.Warn("check channel permission failed: %v", zap.Error(err))
		return viewChats, viewAndPostChats, err
	}

	for channelID, channel := range community.config.CommunityDescription.Chats {
		chatID := community.ChatID(channelID)
		channelPermissionsCheckResult, exists := channelPermissionsCheckResult[chatID]

		if !exists {
			viewAndPostChats[channelID] = channel
			continue
		}

		viewAndPostSatisfied, exists := channelPermissionsCheckResult[protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL]
		if exists && viewAndPostSatisfied {
			delete(viewChats, channelID)
			viewAndPostChats[channelID] = channel
			continue
		}

		viewOnlySatisfied, exists := channelPermissionsCheckResult[protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL]
		if exists && viewOnlySatisfied {
			if _, exists := viewAndPostChats[channelID]; !exists {
				viewChats[channelID] = channel
			}
		}
	}

	return viewChats, viewAndPostChats, nil
}

func (m *Manager) AcceptRequestToJoin(dbRequest *RequestToJoin) (*Community, error) {
	m.communityLock.Lock(dbRequest.CommunityID)
	defer m.communityLock.Unlock(dbRequest.CommunityID)

	pk, err := common.HexToPubkey(dbRequest.PublicKey)
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() {
		revealedAccounts, err := m.persistence.GetRequestToJoinRevealedAddresses(dbRequest.ID)
		if err != nil {
			return nil, err
		}

		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(revealedAccounts)

		communityPermissionsPreParsedData, channelPermissionsPreParsedData := PreParsePermissionsData(community.tokenPermissions())

		permissionsSatisfied, role, err := m.accountsSatisfyPermissionsToJoin(communityPermissionsPreParsedData, accountsAndChainIDs)
		if err != nil {
			return nil, err
		}

		if !permissionsSatisfied {
			return community, ErrNoPermissionToJoin
		}

		memberRoles := []protobuf.CommunityMember_Roles{}
		if role != protobuf.CommunityMember_ROLE_NONE {
			memberRoles = []protobuf.CommunityMember_Roles{role}
		}

		_, err = community.AddMember(pk, memberRoles, dbRequest.Clock)
		if err != nil {
			return nil, err
		}

		viewChannels, postChannels, err := m.accountsSatisfyPermissionsToJoinChannels(community, channelPermissionsPreParsedData, accountsAndChainIDs)
		if err != nil {
			return nil, err
		}

		for channelID := range viewChannels {
			_, err = community.AddMemberToChat(channelID, pk, memberRoles, protobuf.CommunityMember_CHANNEL_ROLE_VIEWER)
			if err != nil {
				return nil, err
			}
		}

		for channelID := range postChannels {
			_, err = community.AddMemberToChat(channelID, pk, memberRoles, protobuf.CommunityMember_CHANNEL_ROLE_POSTER)
			if err != nil {
				return nil, err
			}
		}

		dbRequest.State = RequestToJoinStateAccepted
		if err := m.markRequestToJoinAsAccepted(pk, community); err != nil {
			return nil, err
		}

		dbRequest.RevealedAccounts = revealedAccounts
		if err = m.shareAcceptedRequestToJoinWithPrivilegedMembers(community, dbRequest); err != nil {
			return nil, err
		}

		// if accepted member has a privilege role, share with him requests to join
		memberRole := community.MemberRole(pk)
		if memberRole == protobuf.CommunityMember_ROLE_OWNER || memberRole == protobuf.CommunityMember_ROLE_ADMIN ||
			memberRole == protobuf.CommunityMember_ROLE_TOKEN_MASTER {

			newPrivilegedMember := make(map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey)
			newPrivilegedMember[memberRole] = []*ecdsa.PublicKey{pk}
			if err = m.ShareRequestsToJoinWithPrivilegedMembers(community, newPrivilegedMember); err != nil {
				return nil, err
			}
		}
	} else if community.hasPermissionToSendCommunityEvent(protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT) {
		err := community.addNewCommunityEvent(community.ToCommunityRequestToJoinAcceptCommunityEvent(dbRequest.PublicKey, dbRequest.ToCommunityRequestToJoinProtobuf()))
		if err != nil {
			return nil, err
		}

		dbRequest.State = RequestToJoinStateAcceptedPending
		if err := m.markRequestToJoinAsAcceptedPending(pk, community); err != nil {
			return nil, err
		}
	} else {
		return nil, ErrNotAuthorized
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) GetRequestToJoin(ID types.HexBytes) (*RequestToJoin, error) {
	return m.persistence.GetRequestToJoin(ID)
}

func (m *Manager) DeclineRequestToJoin(dbRequest *RequestToJoin) (*Community, error) {
	m.communityLock.Lock(dbRequest.CommunityID)
	defer m.communityLock.Unlock(dbRequest.CommunityID)

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return nil, err
	}

	adminEventCreated, err := community.DeclineRequestToJoin(dbRequest)
	if err != nil {
		return nil, err
	}

	requestToJoinState := RequestToJoinStateDeclined
	if adminEventCreated {
		requestToJoinState = RequestToJoinStateDeclinedPending // can only be declined by control node
	}

	dbRequest.State = requestToJoinState
	err = m.persistence.SetRequestToJoinState(dbRequest.PublicKey, dbRequest.CommunityID, requestToJoinState)
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) shouldUserRetainDeclined(signer *ecdsa.PublicKey, community *Community, requestClock uint64) (bool, error) {
	requestID := CalculateRequestID(common.PubkeyToHex(signer), types.HexBytes(community.IDString()))
	request, err := m.persistence.GetRequestToJoin(requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return request.ShouldRetainDeclined(requestClock)
}

func (m *Manager) HandleCommunityCancelRequestToJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityCancelRequestToJoin) (*RequestToJoin, error) {
	m.communityLock.Lock(request.CommunityId)
	defer m.communityLock.Unlock(request.CommunityId)

	community, err := m.GetByID(request.CommunityId)
	if err != nil {
		return nil, err
	}

	previousRequestToJoin, err := m.GetRequestToJoinByPkAndCommunityID(signer, community.ID())
	if err != nil {
		return nil, err
	}

	if request.Clock <= previousRequestToJoin.Clock {
		return nil, ErrInvalidClock
	}

	retainDeclined, err := m.shouldUserRetainDeclined(signer, community, request.Clock)
	if err != nil {
		return nil, err
	}
	if retainDeclined {
		return nil, ErrCommunityRequestAlreadyRejected
	}

	err = m.markRequestToJoinAsCanceled(signer, community)
	if err != nil {
		return nil, err
	}

	requestToJoin, err := m.persistence.GetRequestToJoinByPk(common.PubkeyToHex(signer), community.ID(), RequestToJoinStateCanceled)
	if err != nil {
		return nil, err
	}

	if community.HasMember(signer) {
		_, err = community.RemoveUserFromOrg(signer)
		if err != nil {
			return nil, err
		}

		err = m.saveAndPublish(community)
		if err != nil {
			return nil, err
		}
	}

	return requestToJoin, nil
}

func (m *Manager) HandleCommunityRequestToJoin(signer *ecdsa.PublicKey, receiver *ecdsa.PublicKey, request *protobuf.CommunityRequestToJoin) (*Community, *RequestToJoin, error) {
	community, err := m.GetByID(request.CommunityId)
	if err != nil {
		return nil, nil, err
	}

	err = community.ValidateRequestToJoin(signer, request)
	if err != nil {
		return nil, nil, err
	}

	nbPendingRequestsToJoin, err := m.persistence.GetNumberOfPendingRequestsToJoin(community.ID())
	if err != nil {
		return nil, nil, err
	}
	if nbPendingRequestsToJoin >= maxNbPendingRequestedMembers {
		return nil, nil, errors.New("max number of requests to join reached")
	}

	requestToJoin := &RequestToJoin{
		PublicKey:          common.PubkeyToHex(signer),
		Clock:              request.Clock,
		ENSName:            request.EnsName,
		CommunityID:        request.CommunityId,
		State:              RequestToJoinStatePending,
		RevealedAccounts:   request.RevealedAccounts,
		CustomizationColor: multiaccountscommon.IDToColorFallbackToBlue(request.CustomizationColor),
	}
	requestToJoin.CalculateID()

	existingRequestToJoin, err := m.persistence.GetRequestToJoin(requestToJoin.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}

	if existingRequestToJoin == nil {
		err = m.SaveRequestToJoin(requestToJoin)
		if err != nil {
			return nil, nil, err
		}
	} else {
		retainDeclined, err := existingRequestToJoin.ShouldRetainDeclined(request.Clock)
		if err != nil {
			return nil, nil, err
		}
		if retainDeclined {
			return nil, nil, ErrCommunityRequestAlreadyRejected
		}

		switch existingRequestToJoin.State {
		case RequestToJoinStatePending, RequestToJoinStateDeclined, RequestToJoinStateCanceled:
			// Another request have been received, save request back to pending state
			err = m.SaveRequestToJoin(requestToJoin)
			if err != nil {
				return nil, nil, err
			}
		case RequestToJoinStateAccepted:
			// if member leaved the community and tries to request to join again
			if !community.HasMember(signer) {
				err = m.SaveRequestToJoin(requestToJoin)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	if community.IsControlNode() {
		// verify if revealed addresses indeed belong to requester
		for _, revealedAccount := range request.RevealedAccounts {
			recoverParams := account.RecoverParams{
				Message:   types.EncodeHex(crypto.Keccak256(crypto.CompressPubkey(signer), community.ID(), requestToJoin.ID)),
				Signature: types.EncodeHex(revealedAccount.Signature),
			}

			matching, err := m.accountsManager.CanRecover(recoverParams, types.HexToAddress(revealedAccount.Address))
			if err != nil {
				return nil, nil, err
			}
			if !matching {
				// if ownership of only one wallet address cannot be verified,
				// we mark the request as cancelled and stop
				requestToJoin.State = RequestToJoinStateDeclined
				return community, requestToJoin, nil
			}
		}

		// Save revealed addresses + signatures so they can later be added
		// to the control node's local table of known revealed addresses
		err = m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoin.ID, requestToJoin.RevealedAccounts)
		if err != nil {
			return nil, nil, err
		}

		if existingRequestToJoin != nil {
			// request to join was already processed by privileged user
			// and waits to get confirmation for its decision
			if existingRequestToJoin.State == RequestToJoinStateDeclinedPending {
				requestToJoin.State = RequestToJoinStateDeclined
				return community, requestToJoin, nil
			} else if existingRequestToJoin.State == RequestToJoinStateAcceptedPending {
				requestToJoin.State = RequestToJoinStateAccepted
				return community, requestToJoin, nil

			} else if existingRequestToJoin.State == RequestToJoinStateAwaitingAddresses {
				// community ownership changed, accept request automatically
				requestToJoin.State = RequestToJoinStateAccepted
				return community, requestToJoin, nil
			}
		}

		// Check if we reached the limit, if we did, change the community setting to be On Request
		if community.AutoAccept() && community.MembersCount() >= maxNbMembers {
			community.EditPermissionAccess(protobuf.CommunityPermissions_MANUAL_ACCEPT)
			err = m.saveAndPublish(community)
			if err != nil {
				return nil, nil, err
			}
		}

		// If user is already a member, then accept request automatically
		// It may happen when member removes itself from community and then tries to rejoin
		// More specifically, CommunityRequestToLeave may be delivered later than CommunityRequestToJoin, or not delivered at all
		acceptAutomatically := community.AutoAccept() || community.HasMember(signer)
		if acceptAutomatically {
			// Don't check permissions here,
			// it will be done further in the processing pipeline.
			requestToJoin.State = RequestToJoinStateAccepted
			return community, requestToJoin, nil
		}
	}

	return community, requestToJoin, nil
}

func (m *Manager) HandleCommunityEditSharedAddresses(signer *ecdsa.PublicKey, request *protobuf.CommunityEditSharedAddresses) error {
	m.communityLock.Lock(request.CommunityId)
	defer m.communityLock.Unlock(request.CommunityId)

	community, err := m.GetByID(request.CommunityId)
	if err != nil {
		return err
	}

	if !community.IsControlNode() {
		return ErrNotOwner
	}

	publicKey := common.PubkeyToHex(signer)

	if err := community.ValidateEditSharedAddresses(publicKey, request); err != nil {
		return err
	}

	community.UpdateMemberLastUpdateClock(publicKey, request.Clock)
	// verify if revealed addresses indeed belong to requester
	for _, revealedAccount := range request.RevealedAccounts {
		recoverParams := account.RecoverParams{
			Message:   types.EncodeHex(crypto.Keccak256(crypto.CompressPubkey(signer), community.ID())),
			Signature: types.EncodeHex(revealedAccount.Signature),
		}

		matching, err := m.accountsManager.CanRecover(recoverParams, types.HexToAddress(revealedAccount.Address))
		if err != nil {
			return err
		}
		if !matching {
			// if ownership of only one wallet address cannot be verified we stop
			return errors.New("wrong wallet address used")
		}
	}

	err = m.handleCommunityEditSharedAddresses(publicKey, request.CommunityId, request.RevealedAccounts, request.Clock)
	if err != nil {
		return err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return err
	}

	if community.IsControlNode() {
		m.publish(&Subscription{Community: community})
	}

	subscriptionMsg := &CommunityPrivilegedMemberSyncMessage{
		Receivers: community.GetTokenMasterMembers(),
		CommunityPrivilegedUserSyncMessage: &protobuf.CommunityPrivilegedUserSyncMessage{
			Type:        protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_MEMBER_EDIT_SHARED_ADDRESSES,
			CommunityId: community.ID(),
			SyncEditSharedAddresses: &protobuf.SyncCommunityEditSharedAddresses{
				PublicKey:         common.PubkeyToHex(signer),
				EditSharedAddress: request,
			},
		},
	}

	m.publish(&Subscription{CommunityPrivilegedMemberSyncMessage: subscriptionMsg})

	return nil
}

func (m *Manager) handleCommunityEditSharedAddresses(publicKey string, communityID types.HexBytes, revealedAccounts []*protobuf.RevealedAccount, clock uint64) error {
	requestToJoinID := CalculateRequestID(publicKey, communityID)
	err := m.UpdateClockInRequestToJoin(requestToJoinID, clock)
	if err != nil {
		return err
	}

	err = m.persistence.RemoveRequestToJoinRevealedAddresses(requestToJoinID)
	if err != nil {
		return err
	}

	return m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoinID, revealedAccounts)
}

func calculateChainIDsSet(accountsAndChainIDs []*AccountChainIDsCombination, requirementsChainIDs map[uint64]bool) []uint64 {

	revealedAccountsChainIDs := make([]uint64, 0)
	revealedAccountsChainIDsMap := make(map[uint64]bool)

	// we want all chainIDs provided by revealed addresses that also exist
	// in the token requirements
	for _, accountAndChainIDs := range accountsAndChainIDs {
		for _, chainID := range accountAndChainIDs.ChainIDs {
			if requirementsChainIDs[chainID] && !revealedAccountsChainIDsMap[chainID] {
				revealedAccountsChainIDsMap[chainID] = true
				revealedAccountsChainIDs = append(revealedAccountsChainIDs, chainID)
			}
		}
	}
	return revealedAccountsChainIDs
}

type CollectiblesByChain = map[uint64]map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress

func (m *Manager) GetOwnedERC721Tokens(walletAddresses []gethcommon.Address, tokenRequirements map[uint64]map[string]*protobuf.TokenCriteria, chainIDs []uint64) (CollectiblesByChain, error) {
	if m.collectiblesManager == nil {
		return nil, errors.New("no collectibles manager")
	}

	ctx := context.Background()

	ownedERC721Tokens := make(CollectiblesByChain)

	for chainID, erc721Tokens := range tokenRequirements {

		skipChain := true
		for _, cID := range chainIDs {
			if chainID == cID {
				skipChain = false
			}
		}

		if skipChain {
			continue
		}

		contractAddresses := make([]gethcommon.Address, 0)
		for contractAddress := range erc721Tokens {
			contractAddresses = append(contractAddresses, gethcommon.HexToAddress(contractAddress))
		}

		if _, exists := ownedERC721Tokens[chainID]; !exists {
			ownedERC721Tokens[chainID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
		}

		for _, owner := range walletAddresses {
			balances, err := m.collectiblesManager.FetchBalancesByOwnerAndContractAddress(ctx, walletcommon.ChainID(chainID), owner, contractAddresses)
			if err != nil {
				m.logger.Info("couldn't fetch owner assets", zap.Error(err))
				return nil, err
			}
			ownedERC721Tokens[chainID][owner] = balances
		}
	}
	return ownedERC721Tokens, nil
}

func (m *Manager) CheckChannelPermissions(communityID types.HexBytes, chatID string, addresses []gethcommon.Address) (*CheckChannelPermissionsResponse, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	if chatID == "" {
		return nil, errors.New(fmt.Sprintf("couldn't check channel permissions, invalid chat id: %s", chatID))
	}

	viewOnlyPermissions := community.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL)
	viewAndPostPermissions := community.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL)
	viewOnlyPreParsedPermissions := preParsedCommunityPermissionsData(viewOnlyPermissions)
	viewAndPostPreParsedPermissions := preParsedCommunityPermissionsData(viewAndPostPermissions)

	allChainIDs, err := m.tokenManager.GetAllChainIDs()
	if err != nil {
		return nil, err
	}
	accountsAndChainIDs := combineAddressesAndChainIDs(addresses, allChainIDs)

	response, err := m.checkChannelPermissions(viewOnlyPreParsedPermissions, viewAndPostPreParsedPermissions, accountsAndChainIDs, false)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCheckChannelPermissionResponse(communityID.String(), chatID, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

type CheckChannelPermissionsResponse struct {
	ViewOnlyPermissions    *CheckChannelViewOnlyPermissionsResult    `json:"viewOnlyPermissions"`
	ViewAndPostPermissions *CheckChannelViewAndPostPermissionsResult `json:"viewAndPostPermissions"`
}

type CheckChannelViewOnlyPermissionsResult struct {
	Satisfied   bool                                      `json:"satisfied"`
	Permissions map[string]*PermissionTokenCriteriaResult `json:"permissions"`
}

type CheckChannelViewAndPostPermissionsResult struct {
	Satisfied   bool                                      `json:"satisfied"`
	Permissions map[string]*PermissionTokenCriteriaResult `json:"permissions"`
}

func computeViewOnlySatisfied(hasViewOnlyPermissions bool, hasViewAndPostPermissions bool, checkedViewOnlySatisfied bool, checkedViewAndPostSatisified bool) bool {
	if (hasViewAndPostPermissions && !hasViewOnlyPermissions) || (hasViewOnlyPermissions && hasViewAndPostPermissions && checkedViewAndPostSatisified) {
		return checkedViewAndPostSatisified
	} else {
		return checkedViewOnlySatisfied
	}
}

func computeViewAndPostSatisfied(hasViewOnlyPermissions bool, hasViewAndPostPermissions bool, checkedViewAndPostSatisified bool) bool {
	if hasViewOnlyPermissions && !hasViewAndPostPermissions {
		return false
	} else {
		return checkedViewAndPostSatisified
	}
}

func (m *Manager) checkChannelPermissions(viewOnlyPreParsedPermissions *PreParsedCommunityPermissionsData, viewAndPostPreParsedPermissions *PreParsedCommunityPermissionsData, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckChannelPermissionsResponse, error) {
	viewOnlyPermissionsResponse, err := m.PermissionChecker.CheckPermissions(viewOnlyPreParsedPermissions, accountsAndChainIDs, shortcircuit)
	if err != nil {
		return nil, err
	}

	viewAndPostPermissionsResponse, err := m.PermissionChecker.CheckPermissions(viewAndPostPreParsedPermissions, accountsAndChainIDs, shortcircuit)
	if err != nil {
		return nil, err
	}

	hasViewOnlyPermissions := viewOnlyPreParsedPermissions != nil
	hasViewAndPostPermissions := viewAndPostPreParsedPermissions != nil

	return computeCheckChannelPermissionsResponse(hasViewOnlyPermissions, hasViewAndPostPermissions,
			viewOnlyPermissionsResponse, viewAndPostPermissionsResponse),
		nil
}

func computeCheckChannelPermissionsResponse(hasViewOnlyPermissions bool, hasViewAndPostPermissions bool,
	viewOnlyPermissionsResponse *CheckPermissionsResponse, viewAndPostPermissionsResponse *CheckPermissionsResponse) *CheckChannelPermissionsResponse {

	response := &CheckChannelPermissionsResponse{
		ViewOnlyPermissions: &CheckChannelViewOnlyPermissionsResult{
			Satisfied:   false,
			Permissions: make(map[string]*PermissionTokenCriteriaResult),
		},
		ViewAndPostPermissions: &CheckChannelViewAndPostPermissionsResult{
			Satisfied:   false,
			Permissions: make(map[string]*PermissionTokenCriteriaResult),
		},
	}

	viewOnlySatisfied := !hasViewOnlyPermissions || viewOnlyPermissionsResponse.Satisfied
	viewAndPostSatisfied := !hasViewAndPostPermissions || viewAndPostPermissionsResponse.Satisfied

	response.ViewOnlyPermissions.Satisfied = computeViewOnlySatisfied(hasViewOnlyPermissions, hasViewAndPostPermissions,
		viewOnlySatisfied, viewAndPostSatisfied)
	if viewOnlyPermissionsResponse != nil {
		response.ViewOnlyPermissions.Permissions = viewOnlyPermissionsResponse.Permissions
	}

	response.ViewAndPostPermissions.Satisfied = computeViewAndPostSatisfied(hasViewOnlyPermissions, hasViewAndPostPermissions,
		viewAndPostSatisfied)

	if viewAndPostPermissionsResponse != nil {
		response.ViewAndPostPermissions.Permissions = viewAndPostPermissionsResponse.Permissions
	}

	return response
}

func (m *Manager) CheckAllChannelsPermissions(communityID types.HexBytes, addresses []gethcommon.Address) (*CheckAllChannelsPermissionsResponse, error) {

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	channels := community.Chats()

	allChainIDs, err := m.tokenManager.GetAllChainIDs()
	if err != nil {
		return nil, err
	}
	accountsAndChainIDs := combineAddressesAndChainIDs(addresses, allChainIDs)

	_, channelsPermissionsPreParsedData := PreParsePermissionsData(community.tokenPermissions())

	channelPermissionsCheckResult := make(map[string]map[protobuf.CommunityTokenPermission_Type]*CheckPermissionsResponse)

	for permissionId, channelsPermissionPreParsedData := range channelsPermissionsPreParsedData {
		permissionResponse, err := m.PermissionChecker.CheckPermissions(channelsPermissionPreParsedData, accountsAndChainIDs, false)
		if err != nil {
			return nil, err
		}

		// Note: in `PreParsedCommunityPermissionsData` for channels there will be only one permission for channels
		for _, chatId := range channelsPermissionPreParsedData.Permissions[0].ChatIds {
			if _, exists := channelPermissionsCheckResult[chatId]; !exists {
				channelPermissionsCheckResult[chatId] = make(map[protobuf.CommunityTokenPermission_Type]*CheckPermissionsResponse)
			}
			storedPermissionResponse, exists := channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type]
			if !exists {
				channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type] =
					permissionResponse
			} else {
				channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type].Permissions[permissionId] =
					permissionResponse.Permissions[permissionId]
				channelPermissionsCheckResult[chatId][channelsPermissionPreParsedData.Permissions[0].Type].Satisfied =
					storedPermissionResponse.Satisfied || permissionResponse.Satisfied
			}
		}
	}

	response := &CheckAllChannelsPermissionsResponse{
		Channels: make(map[string]*CheckChannelPermissionsResponse),
	}

	for channelID := range channels {
		chatId := community.ChatID(channelID)

		channelCheckPermissionsResponse, exists := channelPermissionsCheckResult[chatId]

		var channelPermissionsResponse *CheckChannelPermissionsResponse
		if !exists {
			channelPermissionsResponse = computeCheckChannelPermissionsResponse(false, false, nil, nil)
		} else {
			viewPermissionsResponse, viewExists := channelCheckPermissionsResponse[protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL]
			postPermissionsResponse, postExists := channelCheckPermissionsResponse[protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL]
			channelPermissionsResponse = computeCheckChannelPermissionsResponse(viewExists, postExists, viewPermissionsResponse, postPermissionsResponse)
		}

		err = m.persistence.SaveCheckChannelPermissionResponse(community.IDString(), chatId, channelPermissionsResponse)
		if err != nil {
			return nil, err
		}
		response.Channels[chatId] = channelPermissionsResponse
	}
	return response, nil
}

func (m *Manager) GetCheckChannelPermissionResponses(communityID types.HexBytes) (*CheckAllChannelsPermissionsResponse, error) {

	response, err := m.persistence.GetCheckChannelPermissionResponses(communityID.String())
	if err != nil {
		return nil, err
	}
	return &CheckAllChannelsPermissionsResponse{Channels: response}, nil
}

type CheckAllChannelsPermissionsResponse struct {
	Channels map[string]*CheckChannelPermissionsResponse `json:"channels"`
}

func (m *Manager) HandleCommunityRequestToJoinResponse(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestToJoinResponse) (*RequestToJoin, error) {
	if len(request.CommunityDescriptionProtocolMessage) > 0 {
		description, err := unmarshalCommunityDescriptionMessage(request.CommunityDescriptionProtocolMessage, signer)
		if err != nil {
			return nil, err
		}
		_, err = m.HandleCommunityDescriptionMessage(signer, description, request.CommunityDescriptionProtocolMessage, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	m.communityLock.Lock(request.CommunityId)
	defer m.communityLock.Unlock(request.CommunityId)

	pkString := common.PubkeyToHex(&m.identity.PublicKey)

	community, err := m.GetByID(request.CommunityId)
	if err != nil {
		return nil, err
	}

	if community.Encrypted() && len(request.Grant) > 0 {
		_, err = m.HandleCommunityGrant(community, request.Grant, request.Clock)
		if err != nil && err != ErrGrantOlder && err != ErrGrantExpired {
			m.logger.Error("Error handling a community grant", zap.Error(err))
		}
	}

	if request.Accepted {
		err = m.markRequestToJoinAsAccepted(&m.identity.PublicKey, community)
		if err != nil {
			return nil, err
		}
	} else {
		err = m.persistence.SetRequestToJoinState(pkString, community.ID(), RequestToJoinStateDeclined)
		if err != nil {
			return nil, err
		}
	}

	return m.persistence.GetRequestToJoinByPkAndCommunityID(pkString, community.ID())
}

func (m *Manager) HandleCommunityRequestToLeave(signer *ecdsa.PublicKey, proto *protobuf.CommunityRequestToLeave) error {
	requestToLeave := NewRequestToLeave(common.PubkeyToHex(signer), proto)
	if err := m.persistence.SaveRequestToLeave(requestToLeave); err != nil {
		return err
	}

	// Ensure corresponding requestToJoin clock is older than requestToLeave
	requestToJoin, err := m.persistence.GetRequestToJoin(requestToLeave.ID)
	if err != nil {
		return err
	}
	if requestToJoin.Clock > requestToLeave.Clock {
		return ErrOldRequestToLeave
	}

	return nil
}

func UnwrapCommunityDescriptionMessage(payload []byte) (*ecdsa.PublicKey, *protobuf.CommunityDescription, error) {

	applicationMetadataMessage := &protobuf.ApplicationMetadataMessage{}
	err := proto.Unmarshal(payload, applicationMetadataMessage)
	if err != nil {
		return nil, nil, err
	}
	if applicationMetadataMessage.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, nil, ErrInvalidMessage
	}
	signer, err := utils.RecoverKey(applicationMetadataMessage)
	if err != nil {
		return nil, nil, err
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(applicationMetadataMessage.Payload, description)
	if err != nil {
		return nil, nil, err
	}

	return signer, description, nil
}

func (m *Manager) JoinCommunity(id types.HexBytes, forceJoin bool) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !forceJoin && community.Joined() {
		// Nothing to do, we are already joined
		return community, ErrOrgAlreadyJoined
	}
	community.Join()
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}
	return community, nil
}

func (m *Manager) SpectateCommunity(id types.HexBytes) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	community.Spectate()
	if err = m.persistence.SaveCommunity(community); err != nil {
		return nil, err
	}
	return community, nil
}

func (m *Manager) GetMagnetlinkMessageClock(communityID types.HexBytes) (uint64, error) {
	return m.persistence.GetMagnetlinkMessageClock(communityID)
}

func (m *Manager) GetCommunityRequestToJoinClock(pk *ecdsa.PublicKey, communityID string) (uint64, error) {
	communityIDBytes, err := types.DecodeHex(communityID)
	if err != nil {
		return 0, err
	}

	joinClock, err := m.persistence.GetRequestToJoinClockByPkAndCommunityID(common.PubkeyToHex(pk), communityIDBytes)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return joinClock, nil
}

func (m *Manager) GetRequestToJoinByPkAndCommunityID(pk *ecdsa.PublicKey, communityID []byte) (*RequestToJoin, error) {
	return m.persistence.GetRequestToJoinByPkAndCommunityID(common.PubkeyToHex(pk), communityID)
}

func (m *Manager) UpdateCommunityDescriptionMagnetlinkMessageClock(communityID types.HexBytes, clock uint64) error {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByIDString(communityID.String())
	if err != nil {
		return err
	}
	community.config.CommunityDescription.ArchiveMagnetlinkClock = clock
	return m.persistence.SaveCommunity(community)
}

func (m *Manager) UpdateMagnetlinkMessageClock(communityID types.HexBytes, clock uint64) error {
	return m.persistence.UpdateMagnetlinkMessageClock(communityID, clock)
}

func (m *Manager) UpdateLastSeenMagnetlink(communityID types.HexBytes, magnetlinkURI string) error {
	return m.persistence.UpdateLastSeenMagnetlink(communityID, magnetlinkURI)
}

func (m *Manager) GetLastSeenMagnetlink(communityID types.HexBytes) (string, error) {
	return m.persistence.GetLastSeenMagnetlink(communityID)
}

func (m *Manager) LeaveCommunity(id types.HexBytes) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	community.RemoveOurselvesFromOrg(&m.identity.PublicKey)
	community.Leave()

	if err = m.persistence.SaveCommunity(community); err != nil {
		return nil, err
	}

	return community, nil
}

// Same as LeaveCommunity, but we have an option to stay spectating
func (m *Manager) KickedOutOfCommunity(id types.HexBytes, spectateMode bool) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	community.RemoveOurselvesFromOrg(&m.identity.PublicKey)
	community.Leave()
	if spectateMode {
		community.Spectate()
	}

	if err = m.persistence.SaveCommunity(community); err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) AddMemberOwnerToCommunity(communityID types.HexBytes, pk *ecdsa.PublicKey) (*Community, error) {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	_, err = community.AddMember(pk, []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_OWNER}, community.Clock())
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})
	return community, nil
}

func (m *Manager) RemoveUserFromCommunity(id types.HexBytes, pk *ecdsa.PublicKey) (*Community, error) {
	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	_, err = community.RemoveUserFromOrg(pk)
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) UnbanUserFromCommunity(request *requests.UnbanUserFromCommunity) (*Community, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	_, err = community.UnbanUserFromCommunity(publicKey)
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) AddRoleToMember(request *requests.AddRoleToMember) (*Community, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !community.hasMember(publicKey) {
		return nil, ErrMemberNotFound
	}

	_, err = community.AddRoleToMember(publicKey, request.Role)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return community, nil
}

func (m *Manager) RemoveRoleFromMember(request *requests.RemoveRoleFromMember) (*Community, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !community.hasMember(publicKey) {
		return nil, ErrMemberNotFound
	}

	_, err = community.RemoveRoleFromMember(publicKey, request.Role)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return community, nil
}

func (m *Manager) BanUserFromCommunity(request *requests.BanUserFromCommunity) (*Community, error) {
	m.communityLock.Lock(request.CommunityID)
	defer m.communityLock.Unlock(request.CommunityID)

	id := request.CommunityID

	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	_, err = community.BanUserFromCommunity(publicKey, &protobuf.CommunityBanInfo{DeleteAllMessages: request.DeleteAllMessages})
	if err != nil {
		return nil, err
	}

	err = m.saveAndPublish(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) dbRecordBundleToCommunity(r *CommunityRecordBundle) (*Community, error) {
	var descriptionEncryptor DescriptionEncryptor
	if m.encryptor != nil {
		descriptionEncryptor = m
	}

	initializer := func(community *Community) error {
		_, description, err := m.preprocessDescription(community.ID(), community.config.CommunityDescription)
		if err != nil {
			return err
		}

		community.config.CommunityDescription = description

		if community.config.EventsData != nil {
			eventsDescription, err := unmarshalCommunityDescriptionMessage(community.config.EventsData.EventsBaseCommunityDescription, community.ControlNode())
			if err != nil {
				m.logger.Error("invalid EventsBaseCommunityDescription", zap.Error(err))
			}
			if eventsDescription != nil && eventsDescription.Clock == community.Clock() {
				community.applyEvents()
			}
		}

		if m.transport != nil && m.transport.WakuVersion() == 2 {
			topic := community.PubsubTopic()
			privKey, err := m.transport.RetrievePubsubTopicKey(topic)
			if err != nil {
				return err
			}
			community.config.PubsubTopicPrivateKey = privKey
		}

		return nil
	}

	return recordBundleToCommunity(
		r,
		m.identity,
		m.installationID,
		m.logger,
		m.timesource,
		descriptionEncryptor,
		m.mediaServer,
		initializer,
	)
}

func (m *Manager) GetByID(id []byte) (*Community, error) {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}
	return community, nil
}

func (m *Manager) GetByIDString(idString string) (*Community, error) {
	id, err := types.DecodeHex(idString)
	if err != nil {
		return nil, err
	}
	return m.GetByID(id)
}

func (m *Manager) GetCommunityShard(communityID types.HexBytes) (*shard.Shard, error) {
	return m.persistence.GetCommunityShard(communityID)
}

func (m *Manager) SaveCommunityShard(communityID types.HexBytes, shard *shard.Shard, clock uint64) error {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	return m.persistence.SaveCommunityShard(communityID, shard, clock)
}

func (m *Manager) DeleteCommunityShard(communityID types.HexBytes) error {
	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	return m.persistence.DeleteCommunityShard(communityID)
}

func (m *Manager) SaveRequestToJoinRevealedAddresses(requestID types.HexBytes, revealedAccounts []*protobuf.RevealedAccount) error {
	return m.persistence.SaveRequestToJoinRevealedAddresses(requestID, revealedAccounts)
}

func (m *Manager) RemoveRequestToJoinRevealedAddresses(requestID types.HexBytes) error {
	return m.persistence.RemoveRequestToJoinRevealedAddresses(requestID)
}

func (m *Manager) SaveRequestToJoinAndCommunity(requestToJoin *RequestToJoin, community *Community) (*Community, *RequestToJoin, error) {
	if err := m.persistence.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, nil, err
	}
	community.config.RequestedToJoinAt = uint64(time.Now().Unix())
	community.AddRequestToJoin(requestToJoin)

	// Save revealed addresses to our own table so that we can retrieve them later when editing
	if err := m.SaveRequestToJoinRevealedAddresses(requestToJoin.ID, requestToJoin.RevealedAccounts); err != nil {
		return nil, nil, err
	}

	return community, requestToJoin, nil
}

func (m *Manager) CreateRequestToJoin(request *requests.RequestToJoinCommunity, customizationColor multiaccountscommon.CustomizationColor) *RequestToJoin {
	clock := uint64(time.Now().Unix())
	requestToJoin := &RequestToJoin{
		PublicKey:            common.PubkeyToHex(&m.identity.PublicKey),
		Clock:                clock,
		ENSName:              request.ENSName,
		CommunityID:          request.CommunityID,
		State:                RequestToJoinStatePending,
		Our:                  true,
		RevealedAccounts:     make([]*protobuf.RevealedAccount, 0),
		CustomizationColor:   customizationColor,
		ShareFutureAddresses: request.ShareFutureAddresses,
	}

	requestToJoin.CalculateID()

	addSignature := len(request.Signatures) == len(request.AddressesToReveal)
	for i := range request.AddressesToReveal {
		revealedAcc := &protobuf.RevealedAccount{
			Address:          request.AddressesToReveal[i],
			IsAirdropAddress: types.HexToAddress(request.AddressesToReveal[i]) == types.HexToAddress(request.AirdropAddress),
		}

		if addSignature {
			revealedAcc.Signature = request.Signatures[i]
		}

		requestToJoin.RevealedAccounts = append(requestToJoin.RevealedAccounts, revealedAcc)
	}

	return requestToJoin
}

func (m *Manager) SaveRequestToJoin(request *RequestToJoin) error {
	return m.persistence.SaveRequestToJoin(request)
}

func (m *Manager) CanceledRequestsToJoinForUser(pk *ecdsa.PublicKey) ([]*RequestToJoin, error) {
	return m.persistence.CanceledRequestsToJoinForUser(common.PubkeyToHex(pk))
}

func (m *Manager) PendingRequestsToJoin() ([]*RequestToJoin, error) {
	return m.persistence.PendingRequestsToJoin()
}

func (m *Manager) PendingRequestsToJoinForUser(pk *ecdsa.PublicKey) ([]*RequestToJoin, error) {
	return m.persistence.RequestsToJoinForUserByState(common.PubkeyToHex(pk), RequestToJoinStatePending)
}

func (m *Manager) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching pending invitations", zap.String("community-id", id.String()))
	return m.persistence.PendingRequestsToJoinForCommunity(id)
}

func (m *Manager) DeclinedRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching declined invitations", zap.String("community-id", id.String()))
	return m.persistence.DeclinedRequestsToJoinForCommunity(id)
}

func (m *Manager) CanceledRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching canceled invitations", zap.String("community-id", id.String()))
	return m.persistence.CanceledRequestsToJoinForCommunity(id)
}

func (m *Manager) AcceptedRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching canceled invitations", zap.String("community-id", id.String()))
	return m.persistence.AcceptedRequestsToJoinForCommunity(id)
}

func (m *Manager) AcceptedPendingRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	return m.persistence.AcceptedPendingRequestsToJoinForCommunity(id)
}

func (m *Manager) DeclinedPendingRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	return m.persistence.DeclinedPendingRequestsToJoinForCommunity(id)
}

func (m *Manager) AllNonApprovedCommunitiesRequestsToJoin() ([]*RequestToJoin, error) {
	m.logger.Info("fetching all non-approved invitations for all communities")
	return m.persistence.AllNonApprovedCommunitiesRequestsToJoin()
}

func (m *Manager) RequestsToJoinForCommunityAwaitingAddresses(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching ownership changed invitations", zap.String("community-id", id.String()))
	return m.persistence.RequestsToJoinForCommunityAwaitingAddresses(id)
}

func (m *Manager) CanPost(pk *ecdsa.PublicKey, communityID string, chatID string, messageType protobuf.ApplicationMetadataMessage_Type) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}
	return community.CanPost(pk, chatID, messageType)
}

func (m *Manager) IsEncrypted(communityID string) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}

	return community.Encrypted(), nil
}

func (m *Manager) IsChannelEncrypted(communityID string, chatID string) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}

	channelID := strings.TrimPrefix(chatID, communityID)
	return community.ChannelEncrypted(channelID), nil
}

func (m *Manager) ShouldHandleSyncCommunity(community *protobuf.SyncInstallationCommunity) (bool, error) {
	return m.persistence.ShouldHandleSyncCommunity(community)
}

func (m *Manager) ShouldHandleSyncCommunitySettings(communitySettings *protobuf.SyncCommunitySettings) (bool, error) {
	return m.persistence.ShouldHandleSyncCommunitySettings(communitySettings)
}

func (m *Manager) HandleSyncCommunitySettings(syncCommunitySettings *protobuf.SyncCommunitySettings) (*CommunitySettings, error) {
	id, err := types.DecodeHex(syncCommunitySettings.CommunityId)
	if err != nil {
		return nil, err
	}

	settings, err := m.persistence.GetCommunitySettingsByID(id)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		settings = &CommunitySettings{
			CommunityID:                  syncCommunitySettings.CommunityId,
			HistoryArchiveSupportEnabled: syncCommunitySettings.HistoryArchiveSupportEnabled,
			Clock:                        syncCommunitySettings.Clock,
		}
	}

	if syncCommunitySettings.Clock > settings.Clock {
		settings.CommunityID = syncCommunitySettings.CommunityId
		settings.HistoryArchiveSupportEnabled = syncCommunitySettings.HistoryArchiveSupportEnabled
		settings.Clock = syncCommunitySettings.Clock
	}

	err = m.persistence.SaveCommunitySettings(*settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (m *Manager) SetSyncClock(id []byte, clock uint64) error {
	return m.persistence.SetSyncClock(id, clock)
}

func (m *Manager) SetPrivateKey(id []byte, privKey *ecdsa.PrivateKey) error {
	return m.persistence.SetPrivateKey(id, privKey)
}

func (m *Manager) GetSyncedRawCommunity(id []byte) (*RawCommunityRow, error) {
	return m.persistence.getSyncedRawCommunity(id)
}

func (m *Manager) GetCommunitySettingsByID(id types.HexBytes) (*CommunitySettings, error) {
	return m.persistence.GetCommunitySettingsByID(id)
}

func (m *Manager) GetCommunitiesSettings() ([]CommunitySettings, error) {
	return m.persistence.GetCommunitiesSettings()
}

func (m *Manager) SaveCommunitySettings(settings CommunitySettings) error {
	return m.persistence.SaveCommunitySettings(settings)
}

func (m *Manager) CommunitySettingsExist(id types.HexBytes) (bool, error) {
	return m.persistence.CommunitySettingsExist(id)
}

func (m *Manager) DeleteCommunitySettings(id types.HexBytes) error {
	return m.persistence.DeleteCommunitySettings(id)
}

func (m *Manager) UpdateCommunitySettings(settings CommunitySettings) error {
	return m.persistence.UpdateCommunitySettings(settings)
}

func (m *Manager) GetOwnedCommunitiesChatIDs() (map[string]bool, error) {
	ownedCommunities, err := m.Controlled()
	if err != nil {
		return nil, err
	}

	chatIDs := make(map[string]bool)
	for _, c := range ownedCommunities {
		if c.Joined() {
			for _, id := range c.ChatIDs() {
				chatIDs[id] = true
			}
		}
	}
	return chatIDs, nil
}

func (m *Manager) StoreWakuMessage(message *types.Message) error {
	return m.persistence.SaveWakuMessage(message)
}

func (m *Manager) StoreWakuMessages(messages []*types.Message) error {
	return m.persistence.SaveWakuMessages(messages)
}

func (m *Manager) GetLatestWakuMessageTimestamp(topics []types.TopicType) (uint64, error) {
	return m.persistence.GetLatestWakuMessageTimestamp(topics)
}

func (m *Manager) GetCommunityToken(communityID string, chainID int, address string) (*community_token.CommunityToken, error) {
	return m.persistence.GetCommunityToken(communityID, chainID, address)
}

func (m *Manager) GetCommunityTokenByChainAndAddress(chainID int, address string) (*community_token.CommunityToken, error) {
	return m.persistence.GetCommunityTokenByChainAndAddress(chainID, address)
}

func (m *Manager) GetCommunityTokens(communityID string) ([]*community_token.CommunityToken, error) {
	return m.persistence.GetCommunityTokens(communityID)
}

func (m *Manager) GetAllCommunityTokens() ([]*community_token.CommunityToken, error) {
	return m.persistence.GetAllCommunityTokens()
}

func (m *Manager) GetCommunityGrant(communityID string) ([]byte, uint64, error) {
	return m.persistence.GetCommunityGrant(communityID)
}

func (m *Manager) ImageToBase64(uri string) string {
	if uri == "" {
		return ""
	}
	file, err := os.Open(uri)
	if err != nil {
		m.logger.Error(err.Error())
		return ""
	}
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	if err != nil {
		m.logger.Error(err.Error())
		return ""
	}
	base64img, err := images.GetPayloadDataURI(payload)
	if err != nil {
		m.logger.Error(err.Error())
		return ""
	}
	return base64img
}

func (m *Manager) SaveCommunityToken(token *community_token.CommunityToken, croppedImage *images.CroppedImage) (*community_token.CommunityToken, error) {

	_, err := m.GetByIDString(token.CommunityID)
	if err != nil {
		return nil, err
	}

	if croppedImage != nil && croppedImage.ImagePath != "" {
		bytes, err := images.OpenAndAdjustImage(*croppedImage, true)
		if err != nil {
			return nil, err
		}

		base64img, err := images.GetPayloadDataURI(bytes)
		if err != nil {
			return nil, err
		}
		token.Base64Image = base64img
	} else if !images.IsPayloadDataURI(token.Base64Image) {
		// if image is already base64 do not convert (owner and master tokens have already base64 image)
		token.Base64Image = m.ImageToBase64(token.Base64Image)
	}

	return token, m.persistence.AddCommunityToken(token)
}

func (m *Manager) AddCommunityToken(token *community_token.CommunityToken, clock uint64) (*Community, error) {
	if token == nil {
		return nil, errors.New("Token is absent in database")
	}

	communityID, err := types.DecodeHex(token.CommunityID)
	if err != nil {
		return nil, err
	}

	m.communityLock.Lock(communityID)
	defer m.communityLock.Unlock(communityID)

	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	if !community.MemberCanManageToken(&m.identity.PublicKey, token) {
		return nil, ErrInvalidManageTokensPermission
	}

	tokenMetadata := &protobuf.CommunityTokenMetadata{
		ContractAddresses: map[uint64]string{uint64(token.ChainID): token.Address},
		Description:       token.Description,
		Image:             token.Base64Image,
		Symbol:            token.Symbol,
		TokenType:         token.TokenType,
		Name:              token.Name,
		Decimals:          uint32(token.Decimals),
		Version:           token.Version,
	}
	_, err = community.AddCommunityTokensMetadata(tokenMetadata)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() && (token.PrivilegesLevel == community_token.MasterLevel || token.PrivilegesLevel == community_token.OwnerLevel) {
		permissionType := protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER
		if token.PrivilegesLevel == community_token.MasterLevel {
			permissionType = protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER
		}

		contractAddresses := make(map[uint64]string)
		contractAddresses[uint64(token.ChainID)] = token.Address

		tokenCriteria := &protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Type:              protobuf.CommunityTokenType_ERC721,
			Symbol:            token.Symbol,
			Name:              token.Name,
			Amount:            "1",
			AmountInWei:       "1",
			Decimals:          uint64(0),
		}

		request := &requests.CreateCommunityTokenPermission{
			CommunityID:   community.ID(),
			Type:          permissionType,
			TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
			IsPrivate:     true,
			ChatIds:       []string{},
		}

		community, _, err = m.createCommunityTokenPermission(request, community)
		if err != nil {
			return nil, err
		}

		if token.PrivilegesLevel == community_token.OwnerLevel {
			_, err = m.promoteSelfToControlNode(community, clock)
			if err != nil {
				return nil, err
			}
		}
	}

	return community, m.saveAndPublish(community)
}

func (m *Manager) UpdateCommunityTokenState(chainID int, contractAddress string, deployState community_token.DeployState) error {
	return m.persistence.UpdateCommunityTokenState(chainID, contractAddress, deployState)
}

func (m *Manager) UpdateCommunityTokenAddress(chainID int, oldContractAddress string, newContractAddress string) error {
	return m.persistence.UpdateCommunityTokenAddress(chainID, oldContractAddress, newContractAddress)
}

func (m *Manager) UpdateCommunityTokenSupply(chainID int, contractAddress string, supply *bigint.BigInt) error {
	return m.persistence.UpdateCommunityTokenSupply(chainID, contractAddress, supply)
}

func (m *Manager) RemoveCommunityToken(chainID int, contractAddress string) error {
	return m.persistence.RemoveCommunityToken(chainID, contractAddress)
}

func (m *Manager) SetCommunityActiveMembersCount(communityID string, activeMembersCount uint64) error {
	id, err := types.DecodeHex(communityID)
	if err != nil {
		return err
	}

	m.communityLock.Lock(id)
	defer m.communityLock.Unlock(id)

	community, err := m.GetByID(id)
	if err != nil {
		return err
	}

	updated, err := community.SetActiveMembersCount(activeMembersCount)
	if err != nil {
		return err
	}

	if updated {
		if err = m.persistence.SaveCommunity(community); err != nil {
			return err
		}

		m.publish(&Subscription{Community: community})
	}

	return nil
}

func combineAddressesAndChainIDs(addresses []gethcommon.Address, chainIDs []uint64) []*AccountChainIDsCombination {
	combinations := make([]*AccountChainIDsCombination, 0)
	for _, address := range addresses {
		combinations = append(combinations, &AccountChainIDsCombination{
			Address:  address,
			ChainIDs: chainIDs,
		})
	}
	return combinations
}

func revealedAccountsToAccountsAndChainIDsCombination(revealedAccounts []*protobuf.RevealedAccount) []*AccountChainIDsCombination {
	accountsAndChainIDs := make([]*AccountChainIDsCombination, 0)
	for _, revealedAccount := range revealedAccounts {
		accountsAndChainIDs = append(accountsAndChainIDs, &AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress(revealedAccount.Address),
			ChainIDs: revealedAccount.ChainIds,
		})
	}
	return accountsAndChainIDs
}

func (m *Manager) accountsHasPrivilegedPermission(preParsedCommunityPermissionData *PreParsedCommunityPermissionsData, accounts []*AccountChainIDsCombination) bool {
	if preParsedCommunityPermissionData != nil {
		permissionResponse, err := m.PermissionChecker.CheckPermissions(preParsedCommunityPermissionData, accounts, true)
		if err != nil {
			m.logger.Warn("check privileged permission failed: %v", zap.Error(err))
			return false
		}
		return permissionResponse.Satisfied
	}
	return false
}

func (m *Manager) SaveAndPublish(community *Community) error {
	return m.saveAndPublish(community)
}

func (m *Manager) saveAndPublish(community *Community) error {
	err := m.persistence.SaveCommunity(community)
	if err != nil {
		return err
	}

	if community.IsControlNode() {
		m.publish(&Subscription{Community: community})
		return nil
	}

	if community.HasPermissionToSendCommunityEvents() {
		err := m.signEvents(community)
		if err != nil {
			return err
		}
		err = m.persistence.SaveCommunityEvents(community)
		if err != nil {
			return err
		}

		m.publish(&Subscription{CommunityEventsMessage: community.toCommunityEventsMessage()})
		return nil
	}

	return nil
}

func (m *Manager) GetRevealedAddresses(communityID types.HexBytes, memberPk string) ([]*protobuf.RevealedAccount, error) {
	logger := m.logger.Named("GetRevealedAddresses")

	requestID := CalculateRequestID(memberPk, communityID)
	response, err := m.persistence.GetRequestToJoinRevealedAddresses(requestID)

	revealedAddresses := make([]string, len(response))
	for i, acc := range response {
		revealedAddresses[i] = acc.Address
	}
	logger.Debug("Revealed addresses", zap.Any("Addresses:", revealedAddresses))

	return response, err
}

func (m *Manager) handleCommunityTokensMetadata(community *Community) error {
	communityID := community.IDString()
	communityTokens := community.CommunityTokensMetadata()

	if len(communityTokens) == 0 {
		return nil
	}
	for _, tokenMetadata := range communityTokens {
		for chainID, address := range tokenMetadata.ContractAddresses {
			exists, err := m.persistence.HasCommunityToken(communityID, address, int(chainID))
			if err != nil {
				return err
			}
			if !exists {
				// Fetch community token to make sure it's stored in the DB, discard result
				communityToken, err := m.FetchCommunityToken(community, tokenMetadata, chainID, address)
				if err != nil {
					return err
				}

				err = m.persistence.AddCommunityToken(communityToken)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m *Manager) HandleCommunityGrant(community *Community, grant []byte, clock uint64) (uint64, error) {
	_, oldClock, err := m.GetCommunityGrant(community.IDString())
	if err != nil {
		return 0, err
	}

	if oldClock >= clock {
		return 0, ErrGrantOlder
	}

	verifiedGrant, err := community.VerifyGrantSignature(grant)
	if err != nil {
		return 0, err
	}

	if !bytes.Equal(verifiedGrant.MemberId, crypto.CompressPubkey(&m.identity.PublicKey)) {
		return 0, ErrGrantMemberPublicKeyIsDifferent
	}

	return clock - oldClock, m.persistence.SaveCommunityGrant(community.IDString(), grant, clock)
}

func (m *Manager) FetchCommunityToken(community *Community, tokenMetadata *protobuf.CommunityTokenMetadata, chainID uint64, contractAddress string) (*community_token.CommunityToken, error) {
	communityID := community.IDString()

	communityToken := &community_token.CommunityToken{
		CommunityID:        communityID,
		Address:            contractAddress,
		TokenType:          tokenMetadata.TokenType,
		Name:               tokenMetadata.Name,
		Symbol:             tokenMetadata.Symbol,
		Description:        tokenMetadata.Description,
		Transferable:       true,
		RemoteSelfDestruct: false,
		ChainID:            int(chainID),
		DeployState:        community_token.Deployed,
		Base64Image:        tokenMetadata.Image,
		Decimals:           int(tokenMetadata.Decimals),
		Version:            tokenMetadata.Version,
	}

	switch tokenMetadata.TokenType {
	case protobuf.CommunityTokenType_ERC721:
		contractData, err := m.communityTokensService.GetCollectibleContractData(chainID, contractAddress)
		if err != nil {
			return nil, err
		}

		communityToken.Supply = contractData.TotalSupply
		communityToken.Transferable = contractData.Transferable
		communityToken.RemoteSelfDestruct = contractData.RemoteBurnable
		communityToken.InfiniteSupply = contractData.InfiniteSupply

	case protobuf.CommunityTokenType_ERC20:
		contractData, err := m.communityTokensService.GetAssetContractData(chainID, contractAddress)
		if err != nil {
			return nil, err
		}

		communityToken.Supply = contractData.TotalSupply
		communityToken.InfiniteSupply = contractData.InfiniteSupply
	}

	communityToken.PrivilegesLevel = getPrivilegesLevel(chainID, contractAddress, community.TokenPermissions())

	return communityToken, nil
}

func getPrivilegesLevel(chainID uint64, tokenAddress string, tokenPermissions map[string]*CommunityTokenPermission) community_token.PrivilegesLevel {
	for _, permission := range tokenPermissions {
		if permission.Type == protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER || permission.Type == protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER {
			for _, tokenCriteria := range permission.TokenCriteria {
				value, exist := tokenCriteria.ContractAddresses[chainID]
				if exist && value == tokenAddress {
					if permission.Type == protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER {
						return community_token.OwnerLevel
					}
					return community_token.MasterLevel
				}
			}
		}
	}
	return community_token.CommunityLevel
}

func (m *Manager) ValidateCommunityPrivilegedUserSyncMessage(message *protobuf.CommunityPrivilegedUserSyncMessage) error {
	if message == nil {
		return errors.New("invalid CommunityPrivilegedUserSyncMessage message")
	}

	if message.CommunityId == nil || len(message.CommunityId) == 0 {
		return errors.New("invalid CommunityId in CommunityPrivilegedUserSyncMessage message")
	}

	switch message.Type {
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN:
		fallthrough
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_REJECT_REQUEST_TO_JOIN:
		if message.RequestToJoin == nil || len(message.RequestToJoin) == 0 {
			return errors.New("invalid request to join in CommunityPrivilegedUserSyncMessage message")
		}

		for _, requestToJoinProto := range message.RequestToJoin {
			if len(requestToJoinProto.CommunityId) == 0 {
				return errors.New("no communityId in request to join in CommunityPrivilegedUserSyncMessage message")
			}
		}
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN:
		if message.SyncRequestsToJoin == nil || len(message.SyncRequestsToJoin) == 0 {
			return errors.New("invalid sync requests to join in CommunityPrivilegedUserSyncMessage message")
		}
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_MEMBER_EDIT_SHARED_ADDRESSES:
		if message.SyncEditSharedAddresses == nil || len(message.CommunityId) == 0 ||
			len(message.SyncEditSharedAddresses.PublicKey) == 0 || message.SyncEditSharedAddresses.EditSharedAddress == nil {
			return errors.New("invalid edit shared adresses in CommunityPrivilegedUserSyncMessage message")
		}
	}

	return nil
}

func (m *Manager) createCommunityTokenPermission(request *requests.CreateCommunityTokenPermission, community *Community) (*Community, *CommunityChanges, error) {
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	tokenPermission := request.ToCommunityTokenPermission()
	tokenPermission.Id = uuid.New().String()
	changes, err := community.UpsertTokenPermission(&tokenPermission)
	if err != nil {
		return nil, nil, err
	}

	return community, changes, nil

}

func (m *Manager) RemoveUsersWithoutRevealedAccounts(community *Community, clock uint64) (*CommunityChanges, error) {
	membersAccounts, err := m.persistence.GetCommunityRequestsToJoinRevealedAddresses(community.ID())
	if err != nil {
		return nil, err
	}

	myPk := common.PubkeyToHex(&m.identity.PublicKey)
	membersToRemove := []string{}
	for pk := range community.Members() {
		if myPk == pk {
			continue
		}
		if _, exists := membersAccounts[pk]; !exists {
			membersToRemove = append(membersToRemove, pk)
		}
	}

	if len(membersToRemove) > 0 {
		community.SetResendAccountsClock(clock)
	}

	return community.RemoveMembersFromOrg(membersToRemove), nil
}

func (m *Manager) PromoteSelfToControlNode(community *Community, clock uint64) (*CommunityChanges, error) {
	if community == nil {
		return nil, ErrOrgNotFound
	}

	m.communityLock.Lock(community.ID())
	defer m.communityLock.Unlock(community.ID())

	ownerChanged, err := m.promoteSelfToControlNode(community, clock)
	if err != nil {
		return nil, err
	}

	if ownerChanged {
		return community.RemoveAllUsersFromOrg(), m.saveAndPublish(community)
	}

	// if control node device was changed, check that we own all members revealed accounts
	// members without revealed accounts will be soft kicked
	changes, err := m.RemoveUsersWithoutRevealedAccounts(community, clock)
	if err != nil {
		return nil, err
	}

	return changes, m.saveAndPublish(community)
}

func (m *Manager) promoteSelfToControlNode(community *Community, clock uint64) (bool, error) {
	ownerChanged := false
	community.setPrivateKey(m.identity)
	if !community.ControlNode().Equal(&m.identity.PublicKey) {
		ownerChanged = true
		community.setControlNode(&m.identity.PublicKey)
	}

	// Mark this device as the control node
	syncControlNode := &protobuf.SyncCommunityControlNode{
		Clock:          clock,
		InstallationId: m.installationID,
	}

	err := m.SaveSyncControlNode(community.ID(), syncControlNode)
	if err != nil {
		return false, err
	}
	community.config.ControlDevice = true

	if exists := community.HasMember(&m.identity.PublicKey); !exists {
		ownerRole := []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_OWNER}
		_, err = community.AddMember(&m.identity.PublicKey, ownerRole, community.Clock())
		if err != nil {
			return false, err
		}

		for channelID := range community.Chats() {
			_, err = community.AddMemberToChat(channelID, &m.identity.PublicKey, ownerRole, protobuf.CommunityMember_CHANNEL_ROLE_POSTER)
			if err != nil {
				return false, err
			}
		}
	} else {
		_, err = community.AddRoleToMember(&m.identity.PublicKey, protobuf.CommunityMember_ROLE_OWNER)
	}

	if err != nil {
		return false, err
	}

	err = m.handleCommunityEvents(community)
	if err != nil {
		return false, err
	}

	community.increaseClock()

	return ownerChanged, nil
}

func (m *Manager) handleCommunityEventsAndMetadata(community *Community, eventsMessage *CommunityEventsMessage,
	lastlyAppliedEvents map[string]uint64) (*CommunityResponse, error) {
	err := community.processEvents(eventsMessage, lastlyAppliedEvents)
	if err != nil {
		return nil, err
	}

	additionalCommunityResponse, err := m.handleAdditionalAdminChanges(community)
	if err != nil {
		return nil, err
	}

	if err = m.handleCommunityTokensMetadata(community); err != nil {
		return nil, err
	}

	return additionalCommunityResponse, err
}

func (m *Manager) handleCommunityEvents(community *Community) error {
	if community.config.EventsData == nil {
		return nil
	}

	lastlyAppliedEvents, err := m.persistence.GetAppliedCommunityEvents(community.ID())
	if err != nil {
		return err
	}

	_, err = m.handleCommunityEventsAndMetadata(community, community.toCommunityEventsMessage(), lastlyAppliedEvents)
	if err != nil {
		return err
	}

	appliedEvents := map[string]uint64{}
	if community.config.EventsData != nil {
		for _, event := range community.config.EventsData.Events {
			appliedEvents[event.EventTypeID()] = event.CommunityEventClock
		}
	}

	community.config.EventsData = nil // clear events, they are already applied
	community.increaseClock()

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return err
	}

	err = m.persistence.UpsertAppliedCommunityEvents(community.ID(), appliedEvents)
	if err != nil {
		return err
	}

	m.publish(&Subscription{Community: community})

	return nil
}

func (m *Manager) ShareRequestsToJoinWithPrivilegedMembers(community *Community, privilegedMembers map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey) error {
	if len(privilegedMembers) == 0 {
		return nil
	}

	requestsToJoin, err := m.GetCommunityRequestsToJoinWithRevealedAddresses(community.ID())
	if err != nil {
		return err
	}

	var syncRequestsWithoutRevealedAccounts []*protobuf.SyncCommunityRequestsToJoin
	var syncRequestsWithRevealedAccounts []*protobuf.SyncCommunityRequestsToJoin
	for _, request := range requestsToJoin {
		// if shared request to join is not approved by control node - do not send revealed accounts.
		// revealed accounts will be sent as soon as control node accepts request to join
		if request.State != RequestToJoinStateAccepted {
			request.RevealedAccounts = []*protobuf.RevealedAccount{}
		}
		syncRequestsWithRevealedAccounts = append(syncRequestsWithRevealedAccounts, request.ToSyncProtobuf())
		requestProtoWithoutAccounts := request.ToSyncProtobuf()
		requestProtoWithoutAccounts.RevealedAccounts = []*protobuf.RevealedAccount{}
		syncRequestsWithoutRevealedAccounts = append(syncRequestsWithoutRevealedAccounts, requestProtoWithoutAccounts)
	}

	syncMsgWithoutRevealedAccounts := &protobuf.CommunityPrivilegedUserSyncMessage{
		Type:               protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN,
		CommunityId:        community.ID(),
		SyncRequestsToJoin: syncRequestsWithoutRevealedAccounts,
	}

	syncMsgWitRevealedAccounts := &protobuf.CommunityPrivilegedUserSyncMessage{
		Type:               protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN,
		CommunityId:        community.ID(),
		SyncRequestsToJoin: syncRequestsWithRevealedAccounts,
	}

	subscriptionMsg := &CommunityPrivilegedMemberSyncMessage{}

	for role, members := range privilegedMembers {
		if len(members) == 0 {
			continue
		}

		subscriptionMsg.Receivers = members

		switch role {
		case protobuf.CommunityMember_ROLE_ADMIN:
			subscriptionMsg.CommunityPrivilegedUserSyncMessage = syncMsgWithoutRevealedAccounts
		case protobuf.CommunityMember_ROLE_OWNER:
			continue
		case protobuf.CommunityMember_ROLE_TOKEN_MASTER:
			subscriptionMsg.CommunityPrivilegedUserSyncMessage = syncMsgWitRevealedAccounts
		}

		m.publish(&Subscription{CommunityPrivilegedMemberSyncMessage: subscriptionMsg})
	}

	return nil
}

func (m *Manager) shareAcceptedRequestToJoinWithPrivilegedMembers(community *Community, requestToJoin *RequestToJoin) error {
	pk, err := common.HexToPubkey(requestToJoin.PublicKey)
	if err != nil {
		return err
	}

	acceptedRequestsToJoinWithoutRevealedAccounts := make(map[string]*protobuf.CommunityRequestToJoin)
	acceptedRequestsToJoinWithRevealedAccounts := make(map[string]*protobuf.CommunityRequestToJoin)

	acceptedRequestsToJoinWithRevealedAccounts[requestToJoin.PublicKey] = requestToJoin.ToCommunityRequestToJoinProtobuf()

	revealedAccounts := requestToJoin.RevealedAccounts
	requestToJoin.RevealedAccounts = make([]*protobuf.RevealedAccount, 0)
	acceptedRequestsToJoinWithoutRevealedAccounts[requestToJoin.PublicKey] = requestToJoin.ToCommunityRequestToJoinProtobuf()
	// Set back the revealed accounts
	requestToJoin.RevealedAccounts = revealedAccounts

	msgWithRevealedAccounts := &protobuf.CommunityPrivilegedUserSyncMessage{
		Type:          protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN,
		CommunityId:   community.ID(),
		RequestToJoin: acceptedRequestsToJoinWithRevealedAccounts,
	}

	msgWithoutRevealedAccounts := &protobuf.CommunityPrivilegedUserSyncMessage{
		Type:          protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN,
		CommunityId:   community.ID(),
		RequestToJoin: acceptedRequestsToJoinWithoutRevealedAccounts,
	}

	// do not sent to ourself and to the accepted user
	skipMembers := make(map[string]struct{})
	skipMembers[common.PubkeyToHex(&m.identity.PublicKey)] = struct{}{}
	skipMembers[common.PubkeyToHex(pk)] = struct{}{}

	subscriptionMsg := &CommunityPrivilegedMemberSyncMessage{}

	fileredPrivilegedMembers := community.GetFilteredPrivilegedMembers(skipMembers)
	for role, members := range fileredPrivilegedMembers {
		if len(members) == 0 {
			continue
		}

		subscriptionMsg.Receivers = members

		switch role {
		case protobuf.CommunityMember_ROLE_ADMIN:
			subscriptionMsg.CommunityPrivilegedUserSyncMessage = msgWithoutRevealedAccounts
		case protobuf.CommunityMember_ROLE_OWNER:
			fallthrough
		case protobuf.CommunityMember_ROLE_TOKEN_MASTER:
			subscriptionMsg.CommunityPrivilegedUserSyncMessage = msgWithRevealedAccounts
		}

		m.publish(&Subscription{CommunityPrivilegedMemberSyncMessage: subscriptionMsg})
	}

	return nil
}

func (m *Manager) GetCommunityRequestsToJoinWithRevealedAddresses(communityID types.HexBytes) ([]*RequestToJoin, error) {
	return m.persistence.GetCommunityRequestsToJoinWithRevealedAddresses(communityID)
}

func (m *Manager) SaveCommunity(community *Community) error {
	return m.persistence.SaveCommunity(community)
}

func (m *Manager) CreateCommunityTokenDeploymentSignature(ctx context.Context, chainID uint64, addressFrom string, communityID string) ([]byte, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return nil, err
	}
	if !community.IsControlNode() {
		return nil, ErrNotControlNode
	}
	digest, err := m.communityTokensService.DeploymentSignatureDigest(chainID, addressFrom, communityID)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(digest, community.PrivateKey())
}

func (m *Manager) GetSyncControlNode(id types.HexBytes) (*protobuf.SyncCommunityControlNode, error) {
	return m.persistence.GetSyncControlNode(id)
}

func (m *Manager) SaveSyncControlNode(id types.HexBytes, syncControlNode *protobuf.SyncCommunityControlNode) error {
	return m.persistence.SaveSyncControlNode(id, syncControlNode.Clock, syncControlNode.InstallationId)
}

func (m *Manager) SetSyncControlNode(id types.HexBytes, syncControlNode *protobuf.SyncCommunityControlNode) error {
	existingSyncControlNode, err := m.GetSyncControlNode(id)
	if err != nil {
		return err
	}

	if existingSyncControlNode == nil || existingSyncControlNode.Clock < syncControlNode.Clock {
		return m.SaveSyncControlNode(id, syncControlNode)
	}

	return nil
}

func (m *Manager) GetCommunityRequestToJoinWithRevealedAddresses(pubKey string, communityID types.HexBytes) (*RequestToJoin, error) {
	return m.persistence.GetCommunityRequestToJoinWithRevealedAddresses(pubKey, communityID)
}

func (m *Manager) SafeGetSignerPubKey(chainID uint64, communityID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	return m.ownerVerifier.SafeGetSignerPubKey(ctx, chainID, communityID)
}

func (m *Manager) GetCuratedCommunities() (*CuratedCommunities, error) {
	return m.persistence.GetCuratedCommunities()
}

func (m *Manager) SetCuratedCommunities(communities *CuratedCommunities) error {
	return m.persistence.SetCuratedCommunities(communities)
}

func (m *Manager) encryptCommunityDescriptionImpl(groupID []byte, d *protobuf.CommunityDescription) (string, []byte, error) {
	payload, err := proto.Marshal(d)
	if err != nil {
		return "", nil, err
	}

	encryptedPayload, ratchet, newSeqNo, err := m.encryptor.EncryptWithHashRatchet(groupID, payload)
	if err == encryption.ErrNoEncryptionKey {
		_, err := m.encryptor.GenerateHashRatchetKey(groupID)
		if err != nil {
			return "", nil, err
		}
		encryptedPayload, ratchet, newSeqNo, err = m.encryptor.EncryptWithHashRatchet(groupID, payload)
		if err != nil {
			return "", nil, err
		}

	} else if err != nil {
		return "", nil, err
	}

	keyID, err := ratchet.GetKeyID()
	if err != nil {
		return "", nil, err
	}

	m.logger.Debug("encrypting community description",
		zap.Any("community", d),
		zap.String("groupID", types.Bytes2Hex(groupID)),
		zap.String("keyID", types.Bytes2Hex(keyID)))

	keyIDSeqNo := fmt.Sprintf("%s%d", hex.EncodeToString(keyID), newSeqNo)

	return keyIDSeqNo, encryptedPayload, nil
}

func (m *Manager) encryptCommunityDescription(community *Community, d *protobuf.CommunityDescription) (string, []byte, error) {
	return m.encryptCommunityDescriptionImpl(community.ID(), d)
}

func (m *Manager) encryptCommunityDescriptionChannel(community *Community, channelID string, d *protobuf.CommunityDescription) (string, []byte, error) {
	return m.encryptCommunityDescriptionImpl([]byte(community.IDString()+channelID), d)
}

// TODO: add collectiblesManager to messenger intance
func (m *Manager) GetCollectiblesManager() CollectiblesManager {
	return m.collectiblesManager
}

type DecryptCommunityResponse struct {
	Decrypted   bool
	Description *protobuf.CommunityDescription
	KeyID       []byte
	GroupID     []byte
}

func (m *Manager) decryptCommunityDescription(keyIDSeqNo string, d []byte) (*DecryptCommunityResponse, error) {
	const hashHexLength = 64
	if len(keyIDSeqNo) <= hashHexLength {
		return nil, errors.New("invalid keyIDSeqNo")
	}

	keyID, err := hex.DecodeString(keyIDSeqNo[:hashHexLength])
	if err != nil {
		return nil, err
	}

	seqNo, err := strconv.ParseUint(keyIDSeqNo[hashHexLength:], 10, 32)
	if err != nil {
		return nil, err
	}

	decryptedPayload, err := m.encryptor.DecryptWithHashRatchet(keyID, uint32(seqNo), d)
	if err == encryption.ErrNoRatchetKey {
		return &DecryptCommunityResponse{
			KeyID: keyID,
		}, err

	}
	if err != nil {
		return nil, err
	}

	var description protobuf.CommunityDescription
	err = proto.Unmarshal(decryptedPayload, &description)
	if err != nil {
		return nil, err
	}

	decryptCommunityResponse := &DecryptCommunityResponse{
		Decrypted:   true,
		KeyID:       keyID,
		Description: &description,
	}
	return decryptCommunityResponse, nil
}

// GetPersistence returns the instantiated *Persistence used by the Manager
func (m *Manager) GetPersistence() *Persistence {
	return m.persistence
}

func ToLinkPreveiwThumbnail(image images.IdentityImage) (*common.LinkPreviewThumbnail, error) {
	thumbnail := &common.LinkPreviewThumbnail{}

	if image.IsEmpty() {
		return nil, nil
	}

	width, height, err := images.GetImageDimensions(image.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to get image dimensions: %w", err)
	}

	dataURI, err := image.GetDataURI()
	if err != nil {
		return nil, fmt.Errorf("failed to get data uri: %w", err)
	}

	thumbnail.Width = width
	thumbnail.Height = height
	thumbnail.DataURI = dataURI
	return thumbnail, nil
}

func (c *Community) ToStatusLinkPreview() (*common.StatusCommunityLinkPreview, error) {
	communityLinkPreview := &common.StatusCommunityLinkPreview{}
	if image, ok := c.Images()[images.SmallDimName]; ok {
		thumbnail, err := ToLinkPreveiwThumbnail(images.IdentityImage{Payload: image.Payload})
		if err != nil {
			c.config.Logger.Warn("unfurling status link: failed to set community thumbnail", zap.Error(err))
		}
		communityLinkPreview.Icon = *thumbnail
	}

	if image, ok := c.Images()[images.BannerIdentityName]; ok {
		thumbnail, err := ToLinkPreveiwThumbnail(images.IdentityImage{Payload: image.Payload})
		if err != nil {
			c.config.Logger.Warn("unfurling status link: failed to set community thumbnail", zap.Error(err))
		}
		communityLinkPreview.Banner = *thumbnail
	}

	communityLinkPreview.CommunityID = c.IDString()
	communityLinkPreview.DisplayName = c.Name()
	communityLinkPreview.Description = c.DescriptionText()
	communityLinkPreview.MembersCount = uint32(c.MembersCount())
	communityLinkPreview.Color = c.Color()

	return communityLinkPreview, nil
}

func (m *Manager) determineChannelsForHRKeysRequest(c *Community, now int64) ([]string, error) {
	result := []string{}

	channelsWithMissingKeys := func() map[string]struct{} {
		r := map[string]struct{}{}
		for id := range c.Chats() {
			if c.HasMissingEncryptionKey(id) {
				r[id] = struct{}{}
			}
		}
		return r
	}()

	if len(channelsWithMissingKeys) == 0 {
		return result, nil
	}

	requests, err := m.persistence.GetEncryptionKeyRequests(c.ID(), channelsWithMissingKeys)
	if err != nil {
		return nil, err
	}

	for channelID := range channelsWithMissingKeys {
		request, ok := requests[channelID]
		if !ok {
			// If there's no prior request, ask for encryption key now
			result = append(result, channelID)
			continue
		}

		// Exponential backoff formula: initial delay * 2^(requestCount - 1)
		initialDelay := int64(10 * 60 * 1000) // 10 minutes in milliseconds
		backoffDuration := initialDelay * (1 << (request.requestedCount - 1))
		nextRequestTime := request.requestedAt + backoffDuration

		if now >= nextRequestTime {
			result = append(result, channelID)
		}
	}

	return result, nil
}

type CommunityWithChannelIDs struct {
	Community  *Community
	ChannelIDs []string
}

// DetermineChannelsForHRKeysRequest identifies channels in a community that
// should ask for encryption keys based on their current state and past request records,
// as determined by exponential backoff.
func (m *Manager) DetermineChannelsForHRKeysRequest() ([]*CommunityWithChannelIDs, error) {
	communities, err := m.Joined()
	if err != nil {
		return nil, err
	}

	result := []*CommunityWithChannelIDs{}
	now := time.Now().UnixMilli()

	for _, c := range communities {
		if c.IsControlNode() {
			continue
		}

		channelsToRequest, err := m.determineChannelsForHRKeysRequest(c, now)
		if err != nil {
			return nil, err
		}

		if len(channelsToRequest) > 0 {
			result = append(result, &CommunityWithChannelIDs{
				Community:  c,
				ChannelIDs: channelsToRequest,
			})
		}
	}

	return result, nil
}

func (m *Manager) updateEncryptionKeysRequests(communityID types.HexBytes, channelIDs []string, now int64) error {
	return m.persistence.UpdateAndPruneEncryptionKeyRequests(communityID, channelIDs, now)
}

func (m *Manager) UpdateEncryptionKeysRequests(communityID types.HexBytes, channelIDs []string) error {
	return m.updateEncryptionKeysRequests(communityID, channelIDs, time.Now().UnixMilli())
}

func unmarshalCommunityDescriptionMessage(signedDescription []byte, signerPubkey *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	metadata := &protobuf.ApplicationMetadataMessage{}

	err := proto.Unmarshal(signedDescription, metadata)
	if err != nil {
		return nil, err
	}

	if metadata.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, ErrInvalidMessage
	}

	signer, err := utils.RecoverKey(metadata)
	if err != nil {
		return nil, err
	}

	if signer == nil {
		return nil, errors.New("CommunityDescription does not contain the control node signature")
	}

	if !signer.Equal(signerPubkey) {
		return nil, errors.New("CommunityDescription was not signed by an owner")
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(metadata.Payload, description)
	if err != nil {
		return nil, err
	}

	return description, nil
}
