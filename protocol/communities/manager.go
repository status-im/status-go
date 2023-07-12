package communities

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/signal"
)

var defaultAnnounceList = [][]string{
	{"udp://tracker.opentrackr.org:1337/announce"},
	{"udp://tracker.openbittorrent.com:6969/announce"},
}
var pieceLength = 100 * 1024

const maxArchiveSizeInBytes = 30000000

var memberPermissionsCheckInterval = 1 * time.Hour

// errors
var (
	ErrTorrentTimedout                 = errors.New("torrent has timed out")
	ErrCommunityRequestAlreadyRejected = errors.New("that user was already rejected from the community")
)

type Manager struct {
	persistence                    *Persistence
	encryptor                      *encryption.Protocol
	ensSubscription                chan []*ens.VerificationRecord
	subscriptions                  []chan *Subscription
	ensVerifier                    *ens.Verifier
	identity                       *ecdsa.PrivateKey
	accountsManager                account.Manager
	tokenManager                   TokenManager
	logger                         *zap.Logger
	stdoutLogger                   *zap.Logger
	transport                      *transport.Transport
	quit                           chan struct{}
	openseaClientBuilder           openseaClientBuilder
	torrentConfig                  *params.TorrentConfig
	torrentClient                  *torrent.Client
	walletConfig                   *params.WalletConfig
	historyArchiveTasksWaitGroup   sync.WaitGroup
	historyArchiveTasks            sync.Map // stores `chan struct{}`
	periodicMemberPermissionsTasks sync.Map // stores `chan struct{}`
	torrentTasks                   map[string]metainfo.Hash
	historyArchiveDownloadTasks    map[string]*HistoryArchiveDownloadTask
	stopped                        bool
}

type openseaClient interface {
	FetchAllAssetsByOwnerAndContractAddress(owner gethcommon.Address, contractAddresses []gethcommon.Address, cursor string, limit int) (*opensea.AssetContainer, error)
}

type openseaClientBuilder interface {
	NewOpenseaClient(uint64, string, *event.Feed) (openseaClient, error)
}

type defaultOpenseaBuilder struct {
}

func (b *defaultOpenseaBuilder) NewOpenseaClient(chainID uint64, apiKey string, feed *event.Feed) (openseaClient, error) {
	return opensea.NewOpenseaClient(chainID, apiKey, nil)
}

type HistoryArchiveDownloadTask struct {
	CancelChan chan struct{}
	Waiter     sync.WaitGroup
	m          sync.RWMutex
	Cancelled  bool
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

type managerOptions struct {
	accountsManager      account.Manager
	tokenManager         TokenManager
	walletConfig         *params.WalletConfig
	openseaClientBuilder openseaClientBuilder
}

type TokenManager interface {
	GetBalancesByChain(ctx context.Context, accounts, tokens []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error)
	GetAllChainIDs() ([]uint64, error)
}

type DefaultTokenManager struct {
	tokenManager *token.Manager
}

func NewDefaultTokenManager(tm *token.Manager) *DefaultTokenManager {
	return &DefaultTokenManager{tokenManager: tm}
}

type BalancesByChain = map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big

func (m *DefaultTokenManager) GetAllChainIDs() ([]uint64, error) {
	networks, err := m.tokenManager.RPCClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}
	return chainIDs, nil
}

func (m *DefaultTokenManager) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error) {
	clients, err := m.tokenManager.RPCClient.EthClients(chainIDs)
	if err != nil {
		return nil, err
	}

	resp, err := m.tokenManager.GetBalancesByChain(context.Background(), clients, accounts, tokenAddresses)
	return resp, err
}

type ManagerOption func(*managerOptions)

func WithAccountManager(accountsManager account.Manager) ManagerOption {
	return func(opts *managerOptions) {
		opts.accountsManager = accountsManager
	}
}

func WithOpenseaClientBuilder(builder openseaClientBuilder) ManagerOption {
	return func(opts *managerOptions) {
		opts.openseaClientBuilder = builder
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

func NewManager(identity *ecdsa.PrivateKey, db *sql.DB, encryptor *encryption.Protocol, logger *zap.Logger, verifier *ens.Verifier, transport *transport.Transport, torrentConfig *params.TorrentConfig, opts ...ManagerOption) (*Manager, error) {
	if identity == nil {
		return nil, errors.New("empty identity")
	}

	var err error
	if logger == nil {
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	stdoutLogger, err := zap.NewDevelopment()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create archive logger")
	}

	managerConfig := managerOptions{}
	for _, opt := range opts {
		opt(&managerConfig)
	}

	manager := &Manager{
		logger:                      logger,
		stdoutLogger:                stdoutLogger,
		encryptor:                   encryptor,
		identity:                    identity,
		quit:                        make(chan struct{}),
		transport:                   transport,
		torrentConfig:               torrentConfig,
		torrentTasks:                make(map[string]metainfo.Hash),
		historyArchiveDownloadTasks: make(map[string]*HistoryArchiveDownloadTask),
		persistence: &Persistence{
			logger: logger,
			db:     db,
		},
	}

	if managerConfig.accountsManager != nil {
		manager.accountsManager = managerConfig.accountsManager
	}

	if managerConfig.tokenManager != nil {
		manager.tokenManager = managerConfig.tokenManager
	}

	if managerConfig.walletConfig != nil {
		manager.walletConfig = managerConfig.walletConfig
	}

	if managerConfig.openseaClientBuilder != nil {
		manager.openseaClientBuilder = managerConfig.openseaClientBuilder
	} else {
		manager.openseaClientBuilder = &defaultOpenseaBuilder{}
	}

	if verifier != nil {

		sub := verifier.Subscribe()
		manager.ensSubscription = sub
		manager.ensVerifier = verifier
	}

	return manager, nil
}

func (m *Manager) LogStdout(msg string, fields ...zap.Field) {
	m.stdoutLogger.Info(msg, fields...)
	m.logger.Debug(msg, fields...)
}

type archiveMDSlice []*archiveMetadata

type archiveMetadata struct {
	hash string
	from uint64
}

func (md archiveMDSlice) Len() int {
	return len(md)
}

func (md archiveMDSlice) Swap(i, j int) {
	md[i], md[j] = md[j], md[i]
}

func (md archiveMDSlice) Less(i, j int) bool {
	return md[i].from > md[j].from
}

type Subscription struct {
	Community                                *Community
	Invitations                              []*protobuf.CommunityInvitation
	CreatingHistoryArchivesSignal            *signal.CreatingHistoryArchivesSignal
	HistoryArchivesCreatedSignal             *signal.HistoryArchivesCreatedSignal
	NoHistoryArchivesCreatedSignal           *signal.NoHistoryArchivesCreatedSignal
	HistoryArchivesSeedingSignal             *signal.HistoryArchivesSeedingSignal
	HistoryArchivesUnseededSignal            *signal.HistoryArchivesUnseededSignal
	HistoryArchiveDownloadedSignal           *signal.HistoryArchiveDownloadedSignal
	DownloadingHistoryArchivesStartedSignal  *signal.DownloadingHistoryArchivesStartedSignal
	DownloadingHistoryArchivesFinishedSignal *signal.DownloadingHistoryArchivesFinishedSignal
	ImportingHistoryArchiveMessagesSignal    *signal.ImportingHistoryArchiveMessagesSignal
	CommunityAdminEvent                      *protobuf.CommunityAdminEvent
	MemberPermissionsCheckedSignal           *MemberPermissionsCheckedSignal
}

type MemberPermissionsCheckedSignal struct {
}

type CommunityResponse struct {
	Community *Community        `json:"community"`
	Changes   *CommunityChanges `json:"changes"`
}

func (m *Manager) Subscribe() chan *Subscription {
	subscription := make(chan *Subscription, 100)
	m.subscriptions = append(m.subscriptions, subscription)
	return subscription
}

func (m *Manager) Start() error {
	m.stopped = false
	if m.ensVerifier != nil {
		m.runENSVerificationLoop()
	}

	if m.torrentConfig != nil && m.torrentConfig.Enabled {
		err := m.StartTorrentClient()
		if err != nil {
			m.LogStdout("couldn't start torrent client", zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) runENSVerificationLoop() {
	go func() {
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

func (m *Manager) Stop() error {
	m.stopped = true
	close(m.quit)
	for _, c := range m.subscriptions {
		close(c)
	}
	m.StopTorrentClient()
	return nil
}

func (m *Manager) SetTorrentConfig(config *params.TorrentConfig) {
	m.torrentConfig = config
}

// getTCPandUDPport will return the same port number given if != 0,
// otherwise, it will attempt to find a free random tcp and udp port using
// the same number for both protocols
func (m *Manager) getTCPandUDPport(portNumber int) (int, error) {
	if portNumber != 0 {
		return portNumber, nil
	}

	// Find free port
	for i := 0; i < 10; i++ {
		tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", "0"))
		if err != nil {
			m.logger.Warn("unable to resolve tcp addr: %v", zap.Error(err))
			continue
		}

		tcpListener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			tcpListener.Close()
			m.logger.Warn("unable to listen on addr", zap.Stringer("addr", tcpAddr), zap.Error(err))
			continue
		}

		port := tcpListener.Addr().(*net.TCPAddr).Port
		tcpListener.Close()

		udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)))
		if err != nil {
			m.logger.Warn("unable to resolve udp addr: %v", zap.Error(err))
			continue
		}

		udpListener, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			udpListener.Close()
			m.logger.Warn("unable to listen on addr", zap.Stringer("addr", udpAddr), zap.Error(err))
			continue
		}

		udpListener.Close()

		return port, nil
	}

	return 0, fmt.Errorf("no free port found")
}

func (m *Manager) StartTorrentClient() error {
	if m.torrentConfig == nil {
		return fmt.Errorf("can't start torrent client: missing torrentConfig")
	}

	if m.TorrentClientStarted() {
		return nil
	}

	port, err := m.getTCPandUDPport(m.torrentConfig.Port)
	if err != nil {
		return err
	}

	config := torrent.NewDefaultClientConfig()
	config.SetListenAddr(":" + fmt.Sprint(port))
	config.Seed = true

	config.DataDir = m.torrentConfig.DataDir

	if _, err := os.Stat(m.torrentConfig.DataDir); os.IsNotExist(err) {
		err := os.MkdirAll(m.torrentConfig.DataDir, 0700)
		if err != nil {
			return err
		}
	}

	m.logger.Info("Starting torrent client", zap.Any("port", port))
	// Instantiating the client will make it bootstrap and listen eagerly,
	// so no go routine is needed here
	client, err := torrent.NewClient(config)
	if err != nil {
		return err
	}
	m.torrentClient = client
	return nil
}

func (m *Manager) StopTorrentClient() []error {
	if m.TorrentClientStarted() {
		m.StopHistoryArchiveTasksIntervals()
		m.logger.Info("Stopping torrent client")
		errs := m.torrentClient.Close()
		if len(errs) > 0 {
			return errs
		}
		m.torrentClient = nil
	}
	return make([]error, 0)
}

func (m *Manager) TorrentClientStarted() bool {
	return m.torrentClient != nil
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

type KnownCommunitiesResponse struct {
	ContractCommunities         []string              `json:"contractCommunities"`
	ContractFeaturedCommunities []string              `json:"contractFeaturedCommunities"`
	Descriptions                map[string]*Community `json:"communities"`
	UnknownCommunities          []string              `json:"unknownCommunities"`
}

func (m *Manager) GetStoredDescriptionForCommunities(communityIDs []types.HexBytes) (response *KnownCommunitiesResponse, err error) {
	response = &KnownCommunitiesResponse{
		Descriptions: make(map[string]*Community),
	}

	for i := range communityIDs {
		communityID := communityIDs[i].String()
		var community *Community
		community, err = m.GetByID(communityIDs[i])
		if err != nil {
			return
		}

		response.ContractCommunities = append(response.ContractCommunities, communityID)

		if community != nil {
			response.Descriptions[community.IDString()] = community
		} else {
			response.UnknownCommunities = append(response.UnknownCommunities, communityID)
		}
	}

	return
}

func (m *Manager) Joined() ([]*Community, error) {
	return m.persistence.JoinedCommunities(&m.identity.PublicKey)
}

func (m *Manager) Spectated() ([]*Community, error) {
	return m.persistence.SpectatedCommunities(&m.identity.PublicKey)
}

func (m *Manager) JoinedAndPendingCommunitiesWithRequests() ([]*Community, error) {
	return m.persistence.JoinedAndPendingCommunitiesWithRequests(&m.identity.PublicKey)
}

func (m *Manager) DeletedCommunities() ([]*Community, error) {
	return m.persistence.DeletedCommunities(&m.identity.PublicKey)
}

func (m *Manager) Created() ([]*Community, error) {
	return m.persistence.CreatedCommunities(&m.identity.PublicKey)
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

	config := Config{
		ID:                   &key.PublicKey,
		PrivateKey:           key,
		Logger:               m.logger,
		Joined:               true,
		MemberIdentity:       &m.identity.PublicKey,
		CommunityDescription: description,
	}
	community, err := New(config)
	if err != nil {
		return nil, err
	}

	// We join any community we create
	community.Join()

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if publish {
		m.publish(&Subscription{Community: community})
	}

	return community, nil
}

func (m *Manager) CreateCommunityTokenPermission(request *requests.CreateCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	tokenPermission := request.ToCommunityTokenPermission()
	tokenPermission.Id = uuid.New().String()

	changes, err := community.AddTokenPermission(&tokenPermission)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityBecomeMemberTokenPermissionChangeAdminEvent()})
	}

	if community.IsOwner() {
		// check existing member permission once, then check periodically
		go func() {
			err := m.checkMemberPermissions(community, false)
			if err != nil {
				m.logger.Debug("failed to check member permissions", zap.Error(err))
			}

			m.CheckMemberPermissionsPeriodically(community.ID())
		}()
	}

	return community, changes, nil
}

func (m *Manager) EditCommunityTokenPermission(request *requests.EditCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	tokenPermission := request.ToCommunityTokenPermission()

	changes, err := community.UpdateTokenPermission(tokenPermission.Id, &tokenPermission)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() && tokenPermission.Type == protobuf.CommunityTokenPermission_BECOME_MEMBER {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityBecomeMemberTokenPermissionChangeAdminEvent()})
	}

	// check if members still fulfill the token criteria of all
	// BECOME_MEMBER permissions and kick them if necessary
	//
	// We do this in a separate routine to not block
	// this function
	go func() {
		err := m.checkMemberPermissions(community, false)
		if err != nil {
			m.logger.Debug("failed to check member permissions", zap.Error(err))
		}
	}()

	return community, changes, nil
}

func (m *Manager) checkMemberPermissions(community *Community, removeAdmins bool) error {
	becomeMemberPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)

	adminPermissions := len(becomeAdminPermissions) > 0
	memberPermissions := len(becomeMemberPermissions) > 0

	if !adminPermissions && !memberPermissions && !removeAdmins {
		return nil
	}

	for memberKey, member := range community.Members() {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return err
		}

		if memberKey == common.PubkeyToHex(&m.identity.PublicKey) || community.IsMemberOwner(memberPubKey) {
			continue
		}

		isAdmin := community.IsMemberAdmin(memberPubKey)
		memberHasWallet := len(member.RevealedAccounts) > 0

		// Check if user was not treated as an admin without wallet in open community
		// or user treated as a member without wallet in closed community
		if (!memberHasWallet && isAdmin) || (memberPermissions && !memberHasWallet) {
			_, err = community.RemoveUserFromOrg(memberPubKey)
			if err != nil {
				return err
			}
			continue
		}

		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(member.RevealedAccounts)

		// Check if user is still an admin or can become an admin and do update of member role
		removeAdminRole := false
		if adminPermissions {
			permissionResponse, err := m.checkPermissionToJoin(becomeAdminPermissions, accountsAndChainIDs, true)
			if err != nil {
				return err
			} else if permissionResponse.Satisfied && !isAdmin {
				_, err = community.AddRoleToMember(memberPubKey, protobuf.CommunityMember_ROLE_ADMIN)
				if err != nil {
					return err
				}
				isAdmin = true
			} else if !permissionResponse.Satisfied && isAdmin {
				removeAdminRole = true
				isAdmin = false
			}
		}

		// Remove admin role if user does not pass admin permissions or we do not have admin permissions but have an admin role
		if removeAdminRole || isAdmin && removeAdmins {
			_, err = community.RemoveRoleFromMember(memberPubKey, protobuf.CommunityMember_ROLE_ADMIN)
			if err != nil {
				return err
			}
			isAdmin = false
		}

		// Skip further validation if user has admin permissions or we do not have member permissions
		if isAdmin || !memberPermissions {
			continue
		}

		permissionResponse, err := m.checkPermissionToJoin(becomeMemberPermissions, accountsAndChainIDs, true)
		if err != nil {
			return err
		}

		if !permissionResponse.Satisfied {
			_, err = community.RemoveUserFromOrg(memberPubKey)
			if err != nil {
				return err
			}
		}
	}

	m.publish(&Subscription{
		Community:                      community,
		MemberPermissionsCheckedSignal: &MemberPermissionsCheckedSignal{},
	})
	return nil
}

func (m *Manager) CheckMemberPermissionsPeriodically(communityID types.HexBytes) {

	if _, exists := m.periodicMemberPermissionsTasks.Load(communityID.String()); exists {
		return
	}

	cancel := make(chan struct{})
	m.periodicMemberPermissionsTasks.Store(communityID.String(), cancel)

	ticker := time.NewTicker(memberPermissionsCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			community, err := m.GetByID(communityID)
			if err != nil {
				m.logger.Debug("can't validate member permissions, community was not found", zap.Error(err))
				m.periodicMemberPermissionsTasks.Delete(communityID.String())
			}

			err = m.checkMemberPermissions(community, true)
			if err != nil {
				m.logger.Debug("failed to check member permissions", zap.Error(err))
			}
		case <-cancel:
			m.periodicMemberPermissionsTasks.Delete(communityID.String())
			return
		}
	}
}

func (m *Manager) DeleteCommunityTokenPermission(request *requests.DeleteCommunityTokenPermission) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	changes, err := community.DeleteTokenPermission(request.PermissionID)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityBecomeMemberTokenPermissionDeleteAdminEvent()})
	}

	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)

	// Make sure that we remove admins roles if we remove admin permissions
	err = m.checkMemberPermissions(community, len(becomeAdminPermissions) == 0)
	if err != nil {
		return nil, nil, err
	}

	// Check if there's stil BECOME_ADMIN and BECOME_MEMBER permissions,
	// if not we can stop checking token criteria on-chain
	if cancel, exists := m.periodicMemberPermissionsTasks.Load(community.IDString()); exists &&
		len(community.TokenPermissions()) == 0 {
		close(cancel.(chan struct{})) // Need to cast to the chan
	}

	return community, changes, nil
}

func (m *Manager) DeleteCommunity(id types.HexBytes) error {
	err := m.persistence.DeleteCommunity(id)
	if err != nil {
		return err
	}
	return m.persistence.DeleteCommunitySettings(id)
}

// EditCommunity takes a description, updates the community with the description,
// saves it and returns it
func (m *Manager) EditCommunity(request *requests.EditCommunity) (*Community, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	if !community.IsOwnerOrAdmin() {
		return nil, ErrNotEnoughPermissions
	}

	newDescription, err := request.ToCommunityDescription()
	if err != nil {
		return nil, fmt.Errorf("Can't create community description: %v", err)
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

	// Edit the community values
	community.Edit(newDescription)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityEditAdminEvent()})
	}

	return community, nil
}

func (m *Manager) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !community.IsOwner() {
		return nil, ErrNotOwner
	}

	return community.config.PrivateKey, nil
}

func (m *Manager) ImportCommunity(key *ecdsa.PrivateKey) (*Community, error) {
	communityID := crypto.CompressPubkey(&key.PublicKey)

	community, err := m.persistence.GetByID(&m.identity.PublicKey, communityID)
	if err != nil {
		return nil, err
	}

	if community == nil {
		description := &protobuf.CommunityDescription{
			Permissions: &protobuf.CommunityPermissions{},
		}

		config := Config{
			ID:                   &key.PublicKey,
			PrivateKey:           key,
			Logger:               m.logger,
			Joined:               true,
			MemberIdentity:       &m.identity.PublicKey,
			CommunityDescription: description,
		}
		community, err = New(config)
		if err != nil {
			return nil, err
		}
	} else {
		community.config.PrivateKey = key
	}

	community.Join()
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) CreateChat(communityID types.HexBytes, chat *protobuf.CommunityChat, publish bool, thirdPartyID string) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}
	chatID := uuid.New().String()
	if thirdPartyID != "" {
		chatID = chatID + thirdPartyID
	}

	changes, err := community.CreateChat(chatID, chat)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if publish {
		if community.IsOwner() {
			m.publish(&Subscription{Community: community})
		} else if community.IsAdmin() {
			m.publish(&Subscription{CommunityAdminEvent: community.ToCreateChannelAdminEvent(chatID, chat)})
		}
	}

	return community, changes, nil
}

func (m *Manager) EditChat(communityID types.HexBytes, chatID string, chat *protobuf.CommunityChat) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(chatID, communityID.String()) {
		chatID = strings.TrimPrefix(chatID, communityID.String())
	}

	changes, err := community.EditChat(chatID, chat)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToEditChannelAdminEvent(chatID, chat)})
	}

	return community, changes, nil
}

func (m *Manager) DeleteChat(communityID types.HexBytes, chatID string) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(chatID, communityID.String()) {
		chatID = strings.TrimPrefix(chatID, communityID.String())
	}
	changes, err := community.DeleteChat(chatID)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToDeleteChannelAdminEvent(chatID)})
	}

	return community, changes, nil
}

func (m *Manager) CreateCategory(request *requests.CreateCommunityCategory, publish bool) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
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

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCreateCategoryAdminEvent(categoryID, request.CategoryName, request.ChatIDs)})
	}

	return community, changes, nil
}

func (m *Manager) EditCategory(request *requests.EditCommunityCategory) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
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

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToEditCategoryAdminEvent(request.CategoryID, request.CategoryName, request.ChatIDs)})
	}

	return community, changes, nil
}

func (m *Manager) EditChatFirstMessageTimestamp(communityID types.HexBytes, chatID string, timestamp uint32) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
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

func (m *Manager) ReorderCategories(request *requests.ReorderCommunityCategories) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	changes, err := community.ReorderCategories(request.CategoryID, request.Position)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToReorderCategoryAdminEvent(request.CategoryID, request.Position)})
	}

	return community, changes, nil
}

func (m *Manager) ReorderChat(request *requests.ReorderCommunityChat) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	// Remove communityID prefix from chatID if exists
	if strings.HasPrefix(request.ChatID, request.CommunityID.String()) {
		request.ChatID = strings.TrimPrefix(request.ChatID, request.CommunityID.String())
	}

	changes, err := community.ReorderChat(request.CategoryID, request.ChatID, request.Position)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToReorderChannelAdminEvent(request.CategoryID, request.ChatID, request.Position)})
	}

	return community, changes, nil
}

func (m *Manager) DeleteCategory(request *requests.DeleteCommunityCategory) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}

	changes, err := community.DeleteCategory(request.CategoryID)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToDeleteCategoryAdminEvent(request.CategoryID)})
	}

	return community, changes, nil
}

func (m *Manager) HandleCommunityDescriptionMessage(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, payload []byte) (*CommunityResponse, error) {
	if signer == nil {
		return nil, errors.New("signer can't be nil")
	}

	id := crypto.CompressPubkey(signer)

	community, err := m.persistence.GetByID(&m.identity.PublicKey, id)
	if err != nil {
		return nil, err
	}

	if community == nil {
		config := Config{
			CommunityDescription:          description,
			Logger:                        m.logger,
			MarshaledCommunityDescription: payload,
			MemberIdentity:                &m.identity.PublicKey,
			ID:                            signer,
		}

		community, err = New(config)
		if err != nil {
			return nil, err
		}
	}

	if !common.IsPubKeyEqual(community.PublicKey(), signer) {
		return nil, ErrNotAuthorized
	}

	return m.handleCommunityDescriptionMessageCommon(community, description, payload)
}

func (m *Manager) handleCommunityDescriptionMessageCommon(community *Community, description *protobuf.CommunityDescription, payload []byte) (*CommunityResponse, error) {
	changes, err := community.UpdateCommunityDescription(description, payload)
	if err != nil {
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

	// If the community require membership, we set whether we should leave/join the community after a state change
	if community.InvitationOnly() || community.OnRequest() || community.AcceptRequestToJoinAutomatically() {
		if changes.HasNewMember(pkString) {
			hasPendingRequest, err := m.persistence.HasPendingRequestsToJoinForUserAndCommunity(pkString, changes.Community.ID())
			if err != nil {
				return nil, err
			}
			// If there's any pending request, we should join the community
			// automatically
			changes.ShouldMemberJoin = hasPendingRequest
		}

		if changes.HasMemberLeft(pkString) {
			// If we joined previously the community, we should leave it
			changes.ShouldMemberLeave = community.Joined()
		}
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	// We mark our requests as completed, though maybe we should mark
	// any request for any user that has been added as completed
	if err := m.markRequestToJoin(&m.identity.PublicKey, community); err != nil {
		return nil, err
	}
	// Check if there's a change and we should be joining

	return &CommunityResponse{
		Community: community,
		Changes:   changes,
	}, nil
}

func (m *Manager) HandleCommunityAdminEvent(signer *ecdsa.PublicKey, adminEvent *protobuf.CommunityAdminEvent, payload []byte) (*CommunityResponse, error) {
	if signer == nil {
		return nil, errors.New("signer can't be nil")
	}

	community, err := m.persistence.GetByID(&m.identity.PublicKey, adminEvent.CommunityId)
	if err != nil {
		return nil, err
	}

	if community == nil {
		return nil, ErrOrgNotFound
	}

	if !community.IsMemberAdmin(signer) {
		return nil, errors.New("user is not an admin")
	}

	patchedCommDescr, err := community.PatchCommunityDescriptionByAdminEvent(adminEvent)
	if err != nil {
		return nil, err
	}

	marshaledCommDescr, err := proto.Marshal(patchedCommDescr)
	if err != nil {
		return nil, err
	}

	// Some admin changes do not live on a `Community` and need to be handled by
	// `Manager` here
	err = m.handleAdditionalAdminChanges(community, adminEvent)
	if err != nil {
		return nil, err
	}

	rawMessage, err := protocol.WrapMessageV1(marshaledCommDescr, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, community.PrivateKey())
	if err != nil {
		return nil, err
	}

	return m.handleCommunityDescriptionMessageCommon(community, patchedCommDescr, rawMessage)
}

func (m *Manager) handleAdditionalAdminChanges(community *Community, adminEvent *protobuf.CommunityAdminEvent) error {

	saveOrUpdateRequestToJoin := func(signer string, request *protobuf.CommunityRequestToJoin, state RequestToJoinState) error {
		requestToJoin := &RequestToJoin{
			PublicKey:        signer,
			Clock:            request.Clock,
			ENSName:          request.EnsName,
			CommunityID:      request.CommunityId,
			State:            state,
			RevealedAccounts: request.RevealedAccounts,
		}

		requestToJoin.CalculateID()

		existingRequestToJoin, err := m.persistence.GetRequestToJoin(requestToJoin.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if existingRequestToJoin != nil {
			// node already knows about this request to join, so let's compare clocks
			// and update it if necessary
			if existingRequestToJoin.Clock <= requestToJoin.Clock {
				pk, err := common.HexToPubkey(existingRequestToJoin.PublicKey)
				if err != nil {
					return err
				}
				err = m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), state)
				if err != nil {
					return err
				}
			}
		} else {
			err := m.persistence.SaveRequestToJoin(requestToJoin)
			if err != nil {
				return err
			}
		}
		return nil
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		for signer, request := range adminEvent.AcceptedRequestsToJoin {
			err := saveOrUpdateRequestToJoin(signer, request, RequestToJoinStateAccepted)
			if err != nil {
				return err
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		for signer, request := range adminEvent.RejectedRequestsToJoin {
			err := saveOrUpdateRequestToJoin(signer, request, RequestToJoinStateDeclined)
			if err != nil {
				return err
			}
		}

	default:
	}
	return nil
}

// TODO: This is not fully implemented, we want to save the grant passed at
// this stage and make sure it's used when publishing.
func (m *Manager) HandleCommunityInvitation(signer *ecdsa.PublicKey, invitation *protobuf.CommunityInvitation, payload []byte) (*CommunityResponse, error) {
	m.logger.Debug("Handling wrapped community description message")

	community, err := m.HandleWrappedCommunityDescriptionMessage(payload)
	if err != nil {
		return nil, err
	}

	// Save grant

	return community, nil
}

// markRequestToJoin marks all the pending requests to join as completed
// if we are members
func (m *Manager) markRequestToJoin(pk *ecdsa.PublicKey, community *Community) error {
	if community.HasMember(pk) {
		return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateAccepted)
	}
	return nil
}

func (m *Manager) markRequestToJoinAsCanceled(pk *ecdsa.PublicKey, community *Community) error {
	return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateCanceled)
}

func (m *Manager) DeletePendingRequestToJoin(request *RequestToJoin) error {
	community, err := m.GetByID(request.CommunityID)
	if err != nil {
		return err
	}

	err = m.persistence.DeletePendingRequestToJoin(request.ID)
	if err != nil {
		return err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return err
	}

	return nil
}

// UpdateClockInRequestToJoin method is used for testing
func (m *Manager) UpdateClockInRequestToJoin(id types.HexBytes, clock uint64) error {
	return m.persistence.UpdateClockInRequestToJoin(id, clock)
}

func (m *Manager) SetMuted(id types.HexBytes, muted bool, mutedTill time.Time) error {
	return m.persistence.SetMuted(id, muted, mutedTill)
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

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	return dbRequest, community, nil
}

func (m *Manager) CheckPermissionToJoin(id []byte, addresses []gethcommon.Address) (*CheckPermissionToJoinResponse, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)
	becomeMemberPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)

	permissionsToJoin := append(becomeAdminPermissions, becomeMemberPermissions...)

	allChainIDs, err := m.tokenManager.GetAllChainIDs()
	if err != nil {
		return nil, err
	}

	accountsAndChainIDs := combineAddressesAndChainIDs(addresses, allChainIDs)

	if len(permissionsToJoin) == 0 {
		// There are no permissions to join on this community at the moment,
		// so we reveal all accounts + all chain IDs
		response := &CheckPermissionsResponse{
			Satisfied:         true,
			Permissions:       make(map[string]*PermissionTokenCriteriaResult),
			ValidCombinations: accountsAndChainIDs,
		}
		return response, nil
	}
	return m.checkPermissionToJoin(permissionsToJoin, accountsAndChainIDs, false)
}

func (m *Manager) AcceptRequestToJoin(request *requests.AcceptRequestToJoinCommunity) (*Community, error) {
	dbRequest, err := m.persistence.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return nil, err
	}

	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)
	becomeMemberPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)

	revealedAccounts, err := m.persistence.GetRequestToJoinRevealedAddresses(dbRequest.ID)
	if err != nil {
		return nil, err
	}

	memberRole := protobuf.CommunityMember_ROLE_NONE

	if len(becomeMemberPermissions) > 0 || len(becomeAdminPermissions) > 0 {

		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(revealedAccounts)

		// admin token permissions required to became an admin must not cancel request to join
		// if requirements were not met
		hasPermission := m.accountsHasAdminPermission(becomeAdminPermissions, accountsAndChainIDs)

		if hasPermission {
			memberRole = protobuf.CommunityMember_ROLE_ADMIN
		} else if len(becomeMemberPermissions) > 0 {
			permissionResponse, err := m.checkPermissionToJoin(becomeMemberPermissions, accountsAndChainIDs, true)
			if err != nil {
				return nil, err
			}

			hasPermission := permissionResponse.Satisfied

			if !hasPermission {
				return community, ErrNoPermissionToJoin
			}
		}
	}

	pk, err := common.HexToPubkey(dbRequest.PublicKey)
	if err != nil {
		return nil, err
	}

	role := []protobuf.CommunityMember_Roles{}
	if memberRole != protobuf.CommunityMember_ROLE_NONE {
		role = []protobuf.CommunityMember_Roles{memberRole}
	}

	changes, err := community.AddMember(pk, role)
	if err != nil {
		return nil, err
	}

	_, err = community.AddMemberRevealedAccounts(dbRequest.PublicKey, revealedAccounts, dbRequest.Clock)
	if err != nil {
		return nil, err
	}

	if err := m.markRequestToJoin(pk, community); err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		acceptedRequestsToJoin := make(map[string]*protobuf.CommunityRequestToJoin)
		acceptedRequestsToJoin[dbRequest.PublicKey] = dbRequest.ToCommunityRequestToJoinProtobuf()

		adminChanges := &CommunityAdminEventChanges{
			CommunityChanges:       changes,
			AcceptedRequestsToJoin: acceptedRequestsToJoin,
		}

		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityRequestToJoinAcceptAdminEvent(adminChanges)})
	}

	return community, nil
}

func (m *Manager) GetRequestToJoin(ID types.HexBytes) (*RequestToJoin, error) {
	return m.persistence.GetRequestToJoin(ID)
}

func (m *Manager) DeclineRequestToJoin(request *requests.DeclineRequestToJoinCommunity) error {
	dbRequest, err := m.persistence.GetRequestToJoin(request.ID)
	if err != nil {
		return err
	}

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return err
	}

	err = m.persistence.SetRequestToJoinState(dbRequest.PublicKey, dbRequest.CommunityID, RequestToJoinStateDeclined)
	if err != nil {
		return err
	}

	// typically, community's clock is increased implicitly when making changes
	// to it, however in this scenario there are no changes in the community, yet
	// we need to increase the clock to ensure the admin event is processed by other
	// nodes.
	community.increaseClock()

	rejectedRequestsToJoin := make(map[string]*protobuf.CommunityRequestToJoin)
	rejectedRequestsToJoin[dbRequest.PublicKey] = dbRequest.ToCommunityRequestToJoinProtobuf()

	adminChanges := &CommunityAdminEventChanges{
		CommunityChanges:       community.emptyCommunityChanges(),
		RejectedRequestsToJoin: rejectedRequestsToJoin,
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToCommunityRequestToJoinRejectAdminEvent(adminChanges)})
	}
	return nil
}

func (m *Manager) isUserRejectedFromCommunity(signer *ecdsa.PublicKey, community *Community, requestClock uint64) (bool, error) {
	declinedRequestsToJoin, err := m.persistence.DeclinedRequestsToJoinForCommunity(community.ID())
	if err != nil {
		return false, err
	}

	for _, req := range declinedRequestsToJoin {
		if req.PublicKey == common.PubkeyToHex(signer) {
			dbRequestTimeOutClock, err := AddTimeoutToRequestToJoinClock(req.Clock)
			if err != nil {
				return false, err
			}

			if requestClock < dbRequestTimeOutClock {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *Manager) HandleCommunityCancelRequestToJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityCancelRequestToJoin) (*RequestToJoin, error) {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, request.CommunityId)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	isUserRejected, err := m.isUserRejectedFromCommunity(signer, community, request.Clock)
	if err != nil {
		return nil, err
	}
	if isUserRejected {
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

	return requestToJoin, nil
}

func (m *Manager) HandleCommunityRequestToJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestToJoin) (*RequestToJoin, error) {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, request.CommunityId)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	isUserRejected, err := m.isUserRejectedFromCommunity(signer, community, request.Clock)
	if err != nil {
		return nil, err
	}
	if isUserRejected {
		return nil, ErrCommunityRequestAlreadyRejected
	}

	// Banned member can't request to join community
	if community.isBanned(signer) {
		return nil, ErrCantRequestAccess
	}

	if err := community.ValidateRequestToJoin(signer, request); err != nil {
		return nil, err
	}

	requestToJoin := &RequestToJoin{
		PublicKey:        common.PubkeyToHex(signer),
		Clock:            request.Clock,
		ENSName:          request.EnsName,
		CommunityID:      request.CommunityId,
		State:            RequestToJoinStatePending,
		RevealedAccounts: request.RevealedAccounts,
	}

	requestToJoin.CalculateID()

	if err := m.persistence.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, err
	}

	if len(request.RevealedAccounts) > 0 {
		// verify if revealed addresses indeed belong to requester
		for _, revealedAccount := range request.RevealedAccounts {
			recoverParams := account.RecoverParams{
				Message:   types.EncodeHex(crypto.Keccak256(crypto.CompressPubkey(signer), community.ID(), requestToJoin.ID)),
				Signature: types.EncodeHex(revealedAccount.Signature),
			}

			matching, err := m.accountsManager.CanRecover(recoverParams, types.HexToAddress(revealedAccount.Address))
			if err != nil {
				return nil, err
			}
			if !matching {
				// if ownership of only one wallet address cannot be verified,
				// we mark the request as cancelled and stop
				err = m.markRequestToJoinAsCanceled(signer, community)
				if err != nil {
					return nil, err
				}
				requestToJoin.State = RequestToJoinStateDeclined
				return requestToJoin, nil
			}
		}

		// Save revealed addresses + signatures so they can later be added
		// to the community member list when the request is accepted
		err = m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoin)
		if err != nil {
			return nil, err
		}
	}

	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)
	becomeMemberPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)

	// If user is already a member, then accept request automatically
	// It may happen when member removes itself from community and then tries to rejoin
	// More specifically, CommunityRequestToLeave may be delivered later than CommunityRequestToJoin, or not delivered at all
	acceptAutomatically := community.AcceptRequestToJoinAutomatically() || community.HasMember(signer)
	hasPermission := false

	// if we have admin or member permission and user revealed the address - process verification
	// if user does not reveal the address and we have member permission only - we decline this request
	// if user does not reveal the address and we have admin permission only - user allowed to join as a member
	// in non private community
	if (len(becomeMemberPermissions) > 0 || len(becomeAdminPermissions) > 0) && len(request.RevealedAccounts) > 0 {
		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(request.RevealedAccounts)

		// admin token permissions required to become an admin must not cancel request to join
		// if requirements were not met
		hasPermission = m.accountsHasAdminPermission(becomeAdminPermissions, accountsAndChainIDs)

		// if user does not have admin permissions, check on member permissions
		if !hasPermission && len(becomeMemberPermissions) > 0 {
			permissionResponse, err := m.checkPermissionToJoin(becomeMemberPermissions, accountsAndChainIDs, true)
			if err != nil {
				return nil, err
			}

			hasPermission = permissionResponse.Satisfied

			if !hasPermission {
				err = m.markRequestToJoinAsCanceled(signer, community)
				if err != nil {
					return nil, err
				}
				requestToJoin.State = RequestToJoinStateDeclined
				return requestToJoin, nil
			}
		}

		// Save revealed addresses + signatures so they can later be added
		// to the community member list when the request is accepted
		err = m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoin)
		if err != nil {
			return nil, err
		}
	} else if len(becomeMemberPermissions) > 0 && len(request.RevealedAccounts) == 0 {
		// we have member token permissions but requester hasn't revealed
		// any addresses
		err = m.markRequestToJoinAsCanceled(signer, community)
		if err != nil {
			return nil, err
		}
		requestToJoin.State = RequestToJoinStateDeclined
		return requestToJoin, nil
	}

	if (len(becomeMemberPermissions) == 0 || hasPermission) && acceptAutomatically {
		err = m.markRequestToJoin(signer, community)
		if err != nil {
			return nil, err
		}
		requestToJoin.State = RequestToJoinStateAccepted
	}

	return requestToJoin, nil
}

func (m *Manager) HandleCommunityEditSharedAddresses(signer *ecdsa.PublicKey, request *protobuf.CommunityEditRevealedAccounts) error {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, request.CommunityId)
	if err != nil {
		return err
	}
	if community == nil {
		return ErrOrgNotFound
	}
	if err := community.ValidateEditSharedAddresses(signer, request); err != nil {
		return err
	}

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

	_, err = community.AddMemberRevealedAccounts(common.PubkeyToHex(signer), request.RevealedAccounts, request.Clock)
	if err != nil {
		return err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	}

	return nil
}

type CheckPermissionsResponse struct {
	Satisfied         bool                                      `json:"satisfied"`
	Permissions       map[string]*PermissionTokenCriteriaResult `json:"permissions"`
	ValidCombinations []*AccountChainIDsCombination             `json:"validCombinations"`
}

type CheckPermissionToJoinResponse = CheckPermissionsResponse

type PermissionTokenCriteriaResult struct {
	Criteria []bool `json:"criteria"`
}

type AccountChainIDsCombination struct {
	Address  gethcommon.Address `json:"address"`
	ChainIDs []uint64           `json:"chainIds"`
}

func (c *CheckPermissionToJoinResponse) calculateSatisfied() {
	if len(c.Permissions) == 0 {
		c.Satisfied = true
		return
	}

	c.Satisfied = false
	for _, p := range c.Permissions {
		satisfied := true
		for _, criteria := range p.Criteria {
			if !criteria {
				satisfied = false
				break
			}
		}
		if satisfied {
			c.Satisfied = true
		}
	}
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

func (m *Manager) checkPermissionToJoin(permissions []*protobuf.CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckPermissionToJoinResponse, error) {
	return m.checkPermissions(permissions, accountsAndChainIDs, shortcircuit)
}

// checkPermissions will retrieve balances and check whether the user has
// permission to join the community, if shortcircuit is true, it will stop as soon
// as we know the answer
func (m *Manager) checkPermissions(permissions []*protobuf.CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckPermissionsResponse, error) {

	response := &CheckPermissionsResponse{
		Satisfied:         false,
		Permissions:       make(map[string]*PermissionTokenCriteriaResult),
		ValidCombinations: make([]*AccountChainIDsCombination, 0),
	}

	erc20TokenRequirements, erc721TokenRequirements, _ := ExtractTokenCriteria(permissions)

	erc20ChainIDsMap := make(map[uint64]bool)
	erc721ChainIDsMap := make(map[uint64]bool)

	erc20TokenAddresses := make([]gethcommon.Address, 0)
	accounts := make([]gethcommon.Address, 0)

	for _, accountAndChainIDs := range accountsAndChainIDs {
		accounts = append(accounts, accountAndChainIDs.Address)
	}

	// figure out chain IDs we're interested in
	for chainID, tokens := range erc20TokenRequirements {
		erc20ChainIDsMap[chainID] = true
		for contractAddress := range tokens {
			erc20TokenAddresses = append(erc20TokenAddresses, gethcommon.HexToAddress(contractAddress))
		}
	}

	for chainID := range erc721TokenRequirements {
		erc721ChainIDsMap[chainID] = true
	}

	chainIDsForERC20 := calculateChainIDsSet(accountsAndChainIDs, erc20ChainIDsMap)
	chainIDsForERC721 := calculateChainIDsSet(accountsAndChainIDs, erc721ChainIDsMap)

	// if there are no chain IDs that match token criteria chain IDs
	// we aren't able to check balances on selected networks
	if len(erc20ChainIDsMap) > 0 && len(chainIDsForERC20) == 0 {
		return response, nil
	}

	ownedERC20TokenBalances := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, 0)
	if len(chainIDsForERC20) > 0 {
		// this only returns balances for the networks we're actually interested in
		balances, err := m.tokenManager.GetBalancesByChain(context.Background(), accounts, erc20TokenAddresses, chainIDsForERC20)
		if err != nil {
			return nil, err
		}
		ownedERC20TokenBalances = balances
	}

	ownedERC721Tokens := make(CollectiblesByChain)
	if len(chainIDsForERC721) > 0 {
		collectibles, err := m.GetOwnedERC721Tokens(accounts, erc721TokenRequirements, chainIDsForERC721)
		if err != nil {
			return nil, err
		}
		ownedERC721Tokens = collectibles
	}

	accountsChainIDsCombinations := make(map[gethcommon.Address]map[uint64]bool)

	for _, tokenPermission := range permissions {

		permissionRequirementsMet := true
		response.Permissions[tokenPermission.Id] = &PermissionTokenCriteriaResult{}

		// There can be multiple token requirements per permission.
		// If only one is not met, the entire permission is marked
		// as not fulfilled
		for _, tokenRequirement := range tokenPermission.TokenCriteria {

			tokenRequirementMet := false

			if tokenRequirement.Type == protobuf.CommunityTokenType_ERC721 {
				if len(ownedERC721Tokens) == 0 {
					response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, false)
					continue
				}

			chainIDLoopERC721:
				for chainID, address := range tokenRequirement.ContractAddresses {
					if _, exists := ownedERC721Tokens[chainID]; !exists || len(ownedERC721Tokens[chainID]) == 0 {
						continue chainIDLoopERC721
					}

					for account := range ownedERC721Tokens[chainID] {
						if _, exists := ownedERC721Tokens[chainID][account]; !exists {
							continue
						}

						if _, exists := ownedERC721Tokens[chainID][account][strings.ToLower(address)]; exists {

							if _, exists := accountsChainIDsCombinations[account]; !exists {
								accountsChainIDsCombinations[account] = make(map[uint64]bool)
							}

							if len(tokenRequirement.TokenIds) == 0 {
								// no specific tokenId of this collection is needed
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][chainID] = true
								break chainIDLoopERC721
							}

						tokenIDsLoop:
							for _, tokenID := range tokenRequirement.TokenIds {
								tokenIDBigInt := new(big.Int).SetUint64(tokenID)

								for _, asset := range ownedERC721Tokens[chainID][account][strings.ToLower(address)] {
									if asset.TokenID.Cmp(tokenIDBigInt) == 0 {
										tokenRequirementMet = true
										accountsChainIDsCombinations[account][chainID] = true
										break tokenIDsLoop
									}
								}
							}
						}
					}
				}
			} else if tokenRequirement.Type == protobuf.CommunityTokenType_ERC20 {
				if len(ownedERC20TokenBalances) == 0 {
					response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, false)
					continue
				}

				accumulatedBalance := new(big.Float)

			chainIDLoopERC20:
				for chainID, address := range tokenRequirement.ContractAddresses {
					if _, exists := ownedERC20TokenBalances[chainID]; !exists || len(ownedERC20TokenBalances[chainID]) == 0 {
						continue chainIDLoopERC20
					}
					contractAddress := gethcommon.HexToAddress(address)
					for account := range ownedERC20TokenBalances[chainID] {
						if _, exists := ownedERC20TokenBalances[chainID][account][contractAddress]; !exists {
							continue
						}

						value := ownedERC20TokenBalances[chainID][account][contractAddress]

						accountChainBalance := new(big.Float).Quo(
							new(big.Float).SetInt(value.ToInt()),
							big.NewFloat(math.Pow(10, float64(tokenRequirement.Decimals))),
						)

						if _, exists := accountsChainIDsCombinations[account]; !exists {
							accountsChainIDsCombinations[account] = make(map[uint64]bool)
						}

						if accountChainBalance.Cmp(big.NewFloat(0)) > 0 {
							// account has balance > 0 on this chain for this token, so let's add it the chain IDs
							accountsChainIDsCombinations[account][chainID] = true
						}

						// check if adding current chain account balance to accumulated balance
						// satisfies required amount
						prevBalance := accumulatedBalance
						accumulatedBalance.Add(prevBalance, accountChainBalance)

						requiredAmount, err := strconv.ParseFloat(tokenRequirement.Amount, 32)
						if err != nil {
							return nil, err
						}

						if accumulatedBalance.Cmp(big.NewFloat(requiredAmount)) != -1 {
							tokenRequirementMet = true
							if shortcircuit {
								break chainIDLoopERC20
							}
						}
					}
				}

			} else if tokenRequirement.Type == protobuf.CommunityTokenType_ENS {

				for _, account := range accounts {
					ownedENSNames, err := m.getOwnedENS([]gethcommon.Address{account})
					if err != nil {
						return nil, err
					}

					if _, exists := accountsChainIDsCombinations[account]; !exists {
						accountsChainIDsCombinations[account] = make(map[uint64]bool)
					}

					if !strings.HasPrefix(tokenRequirement.EnsPattern, "*.") {
						for _, ownedENS := range ownedENSNames {
							if ownedENS == tokenRequirement.EnsPattern {
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][walletcommon.EthereumMainnet] = true
							}
						}
					} else {
						parentName := tokenRequirement.EnsPattern[2:]
						for _, ownedENS := range ownedENSNames {
							if strings.HasSuffix(ownedENS, parentName) {
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][walletcommon.EthereumMainnet] = true
							}
						}
					}
				}
			}
			if !tokenRequirementMet {
				permissionRequirementsMet = false
			}
			response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, tokenRequirementMet)
		}
		// multiple permissions are treated as logical OR, meaning
		// if only one of them is fulfilled, the user gets permission
		// to join and we can stop early
		if shortcircuit && permissionRequirementsMet {
			break
		}
	}

	// attach valid account and chainID combinations to response
	for account, chainIDs := range accountsChainIDsCombinations {
		combination := &AccountChainIDsCombination{
			Address: account,
		}
		for chainID := range chainIDs {
			combination.ChainIDs = append(combination.ChainIDs, chainID)
		}
		response.ValidCombinations = append(response.ValidCombinations, combination)
	}

	response.calculateSatisfied()

	return response, nil
}

type CollectiblesByChain = map[uint64]map[gethcommon.Address]map[string][]opensea.Asset

func (m *Manager) GetOwnedERC721Tokens(walletAddresses []gethcommon.Address, tokenRequirements map[uint64]map[string]*protobuf.TokenCriteria, chainIDs []uint64) (CollectiblesByChain, error) {

	if m.walletConfig == nil || m.walletConfig.OpenseaAPIKey == "" {
		return nil, errors.New("no opensea client")
	}

	ownedERC721Tokens := make(map[uint64]map[gethcommon.Address]map[string][]opensea.Asset)

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

		client, err := m.openseaClientBuilder.NewOpenseaClient(chainID, m.walletConfig.OpenseaAPIKey, nil)
		if err != nil {
			return nil, err
		}

		contractAddresses := make([]gethcommon.Address, 0)
		for contractAddress := range erc721Tokens {
			contractAddresses = append(contractAddresses, gethcommon.HexToAddress(contractAddress))
		}

		if _, exists := ownedERC721Tokens[chainID]; !exists {
			ownedERC721Tokens[chainID] = make(map[gethcommon.Address]map[string][]opensea.Asset)
		}

		for _, owner := range walletAddresses {
			assets, err := client.FetchAllAssetsByOwnerAndContractAddress(owner, contractAddresses, "", 5)
			if err != nil {
				m.logger.Info("couldn't fetch owner assets", zap.Error(err))
				return nil, err
			}

			if len(assets.Assets) == 0 {
				continue
			}

			if _, exists := ownedERC721Tokens[chainID][owner]; !exists {
				ownedERC721Tokens[chainID][owner] = make(map[string][]opensea.Asset, 0)
			}

			for _, asset := range assets.Assets {
				if _, exists := ownedERC721Tokens[chainID][owner][asset.Contract.Address]; !exists {
					ownedERC721Tokens[chainID][owner][asset.Contract.Address] = make([]opensea.Asset, 0)
				}
				ownedERC721Tokens[chainID][owner][asset.Contract.Address] = append(ownedERC721Tokens[chainID][owner][asset.Contract.Address], asset)
			}
		}
	}
	return ownedERC721Tokens, nil
}

func (m *Manager) getOwnedENS(addresses []gethcommon.Address) ([]string, error) {
	ownedENS := make([]string, 0)
	if m.ensVerifier == nil {
		m.logger.Warn("no ensVerifier configured for communities manager")
		return ownedENS, nil
	}
	for _, address := range addresses {
		name, err := m.ensVerifier.ReverseResolve(address)
		if err != nil && err.Error() != "not a resolver" {
			return ownedENS, err
		}
		if name != "" {
			ownedENS = append(ownedENS, name)
		}
	}
	return ownedENS, nil
}

func (m *Manager) CheckChannelPermissions(communityID types.HexBytes, chatID string, addresses []gethcommon.Address) (*CheckChannelPermissionsResponse, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	if chatID == "" {
		return nil, errors.New(fmt.Sprintf("couldn't check channel permissions, invalid chat id: %s", chatID))
	}

	viewOnlyPermissions := community.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL)
	viewAndPostPermissions := community.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL)

	allChainIDs, err := m.tokenManager.GetAllChainIDs()
	if err != nil {
		return nil, err
	}
	accountsAndChainIDs := combineAddressesAndChainIDs(addresses, allChainIDs)

	response, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountsAndChainIDs, false)
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

func (m *Manager) checkChannelPermissions(viewOnlyPermissions []*protobuf.CommunityTokenPermission, viewAndPostPermissions []*protobuf.CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckChannelPermissionsResponse, error) {

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

	viewOnlyPermissionsResponse, err := m.checkPermissions(viewOnlyPermissions, accountsAndChainIDs, shortcircuit)
	if err != nil {
		return nil, err
	}

	viewAndPostPermissionsResponse, err := m.checkPermissions(viewAndPostPermissions, accountsAndChainIDs, shortcircuit)
	if err != nil {
		return nil, err
	}

	hasViewOnlyPermissions := len(viewOnlyPermissions) > 0
	hasViewAndPostPermissions := len(viewAndPostPermissions) > 0

	if (hasViewAndPostPermissions && !hasViewOnlyPermissions) || (hasViewOnlyPermissions && hasViewAndPostPermissions && viewAndPostPermissionsResponse.Satisfied) {
		response.ViewOnlyPermissions.Satisfied = viewAndPostPermissionsResponse.Satisfied
	} else {
		response.ViewOnlyPermissions.Satisfied = viewOnlyPermissionsResponse.Satisfied
	}
	response.ViewOnlyPermissions.Permissions = viewOnlyPermissionsResponse.Permissions

	if (hasViewOnlyPermissions && !viewOnlyPermissionsResponse.Satisfied) ||
		(hasViewOnlyPermissions && !hasViewAndPostPermissions) {
		response.ViewAndPostPermissions.Satisfied = false
	} else {
		response.ViewAndPostPermissions.Satisfied = viewAndPostPermissionsResponse.Satisfied
	}
	response.ViewAndPostPermissions.Permissions = viewAndPostPermissionsResponse.Permissions

	return response, nil
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

	response := &CheckAllChannelsPermissionsResponse{
		Channels: make(map[string]*CheckChannelPermissionsResponse),
	}

	for channelID := range channels {
		viewOnlyPermissions := community.ChannelTokenPermissionsByType(community.IDString()+channelID, protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL)
		viewAndPostPermissions := community.ChannelTokenPermissionsByType(community.IDString()+channelID, protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL)

		checkChannelPermissionsResponse, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountsAndChainIDs, false)
		if err != nil {
			return nil, err
		}
		err = m.persistence.SaveCheckChannelPermissionResponse(community.IDString(), community.IDString()+channelID, checkChannelPermissionsResponse)
		if err != nil {
			return nil, err
		}
		response.Channels[community.IDString()+channelID] = checkChannelPermissionsResponse
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
	pkString := common.PubkeyToHex(&m.identity.PublicKey)

	community, err := m.persistence.GetByID(&m.identity.PublicKey, request.CommunityId)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	communityDescriptionBytes, err := proto.Marshal(request.Community)
	if err != nil {
		return nil, err
	}

	// We need to wrap `request.Community` in an `ApplicationMetadataMessage`
	// of type `CommunityDescription` because `UpdateCommunityDescription` expects this.
	//
	// This is merely for marsheling/unmarsheling, hence we attaching a `Signature`
	// is not needed.
	metadataMessage := &protobuf.ApplicationMetadataMessage{
		Payload: communityDescriptionBytes,
		Type:    protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
	}

	appMetadataMsg, err := proto.Marshal(metadataMessage)
	if err != nil {
		return nil, err
	}

	if !common.IsPubKeyEqual(community.PublicKey(), signer) && !community.IsMemberAdmin(signer) {
		return nil, ErrNotAuthorized
	}

	_, err = community.UpdateCommunityDescription(request.Community, appMetadataMsg)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if request.Accepted {
		err = m.markRequestToJoin(&m.identity.PublicKey, community)
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

func (m *Manager) HandleWrappedCommunityDescriptionMessage(payload []byte) (*CommunityResponse, error) {
	m.logger.Debug("Handling wrapped community description message")

	applicationMetadataMessage := &protobuf.ApplicationMetadataMessage{}
	err := proto.Unmarshal(payload, applicationMetadataMessage)
	if err != nil {
		return nil, err
	}
	if applicationMetadataMessage.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, ErrInvalidMessage
	}
	signer, err := applicationMetadataMessage.RecoverKey()
	if err != nil {
		return nil, err
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(applicationMetadataMessage.Payload, description)
	if err != nil {
		return nil, err
	}

	return m.HandleCommunityDescriptionMessage(signer, description, payload)
}

func (m *Manager) JoinCommunity(id types.HexBytes, forceJoin bool) (*Community, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
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
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
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

func (m *Manager) GetRequestToJoinIDByPkAndCommunityID(pk *ecdsa.PublicKey, communityID []byte) ([]byte, error) {
	return m.persistence.GetRequestToJoinIDByPkAndCommunityID(common.PubkeyToHex(pk), communityID)
}

func (m *Manager) UpdateCommunityDescriptionMagnetlinkMessageClock(communityID types.HexBytes, clock uint64) error {
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
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	community.RemoveOurselvesFromOrg(&m.identity.PublicKey)
	community.Leave()

	if err = m.persistence.SaveCommunity(community); err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) inviteUsersToCommunity(community *Community, pks []*ecdsa.PublicKey) (*Community, error) {
	var invitations []*protobuf.CommunityInvitation
	for _, pk := range pks {
		invitation, err := community.InviteUserToOrg(pk)
		if err != nil {
			return nil, err
		}
		// We mark the user request (if any) as completed
		if err := m.markRequestToJoin(pk, community); err != nil {
			return nil, err
		}

		invitations = append(invitations, invitation)
	}

	err := m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community, Invitations: invitations})

	return community, nil
}

func (m *Manager) InviteUsersToCommunity(communityID types.HexBytes, pks []*ecdsa.PublicKey) (*Community, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	return m.inviteUsersToCommunity(community, pks)
}

func (m *Manager) AddMemberOwnerToCommunity(communityID types.HexBytes, pk *ecdsa.PublicKey) (*Community, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.AddMember(pk, []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_OWNER})
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
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.RemoveUserFromOrg(pk)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToKickCommunityMemberAdminEvent(common.PubkeyToHex(pk))})
	}

	return community, nil
}

func (m *Manager) UnbanUserFromCommunity(request *requests.UnbanUserFromCommunity) (*Community, error) {
	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.UnbanUserFromCommunity(publicKey)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToUnbanCommunityMemberAdminEvent(request.User.String())})
	}

	return community, nil
}

func (m *Manager) AddRoleToMember(request *requests.AddRoleToMember) (*Community, error) {
	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
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
	id := request.CommunityID
	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
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
	id := request.CommunityID

	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.BanUserFromCommunity(publicKey)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	if community.IsOwner() {
		m.publish(&Subscription{Community: community})
	} else if community.IsAdmin() {
		m.publish(&Subscription{CommunityAdminEvent: community.ToBanCommunityMemberAdminEvent(request.User.String())})
	}

	return community, nil
}

func (m *Manager) GetByID(id []byte) (*Community, error) {
	return m.persistence.GetByID(&m.identity.PublicKey, id)
}

func (m *Manager) GetByIDString(idString string) (*Community, error) {
	id, err := types.DecodeHex(idString)
	if err != nil {
		return nil, err
	}
	return m.GetByID(id)
}

func (m *Manager) RequestToJoin(requester *ecdsa.PublicKey, request *requests.RequestToJoinCommunity) (*Community, *RequestToJoin, error) {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	// We don't allow requesting access if already joined
	if community.Joined() {
		return nil, nil, ErrAlreadyJoined
	}

	clock := uint64(time.Now().Unix())
	requestToJoin := &RequestToJoin{
		PublicKey:        common.PubkeyToHex(requester),
		Clock:            clock,
		ENSName:          request.ENSName,
		CommunityID:      request.CommunityID,
		State:            RequestToJoinStatePending,
		Our:              true,
		RevealedAccounts: make([]*protobuf.RevealedAccount, 0),
	}

	requestToJoin.CalculateID()

	if err := m.persistence.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, nil, err
	}
	community.config.RequestedToJoinAt = uint64(time.Now().Unix())
	community.AddRequestToJoin(requestToJoin)

	return community, requestToJoin, nil
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
	return m.persistence.PendingRequestsToJoinForUser(common.PubkeyToHex(pk))
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

func (m *Manager) CanPost(pk *ecdsa.PublicKey, communityID string, chatID string, grant []byte) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}
	if community == nil {
		return false, nil
	}
	return community.CanPost(pk, chatID, grant)
}

func (m *Manager) IsEncrypted(communityID string) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}

	return community.Encrypted(), nil

}
func (m *Manager) ShouldHandleSyncCommunity(community *protobuf.SyncCommunity) (bool, error) {
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

func (m *Manager) GetAdminCommunitiesChatIDs() (map[string]bool, error) {
	adminCommunities, err := m.Created()
	if err != nil {
		return nil, err
	}

	chatIDs := make(map[string]bool)
	for _, c := range adminCommunities {
		if c.Joined() {
			for _, id := range c.ChatIDs() {
				chatIDs[id] = true
			}
		}
	}
	return chatIDs, nil
}

func (m *Manager) IsAdminCommunityByID(communityID types.HexBytes) (bool, error) {
	pubKey, err := crypto.DecompressPubkey(communityID)
	if err != nil {
		return false, err
	}
	return m.IsAdminCommunity(pubKey)
}

func (m *Manager) IsAdminCommunity(pubKey *ecdsa.PublicKey) (bool, error) {
	adminCommunities, err := m.Created()
	if err != nil {
		return false, err
	}

	for _, c := range adminCommunities {
		if c.PrivateKey().PublicKey.Equal(pubKey) {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) IsJoinedCommunity(pubKey *ecdsa.PublicKey) (bool, error) {
	community, err := m.GetByID(crypto.CompressPubkey(pubKey))
	if err != nil {
		return false, err
	}

	return community != nil && community.Joined(), nil
}

func (m *Manager) GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error) {
	chatIDs, err := m.persistence.GetCommunityChatIDs(communityID)
	if err != nil {
		return nil, err
	}

	filters := []*transport.Filter{}
	for _, cid := range chatIDs {
		filters = append(filters, m.transport.FilterByChatID(cid))
	}
	return filters, nil
}

func (m *Manager) GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error) {
	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		return nil, err
	}

	topics := []types.TopicType{}
	for _, filter := range filters {
		topics = append(topics, filter.Topic)
	}

	return topics, nil
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

func (m *Manager) GetOldestWakuMessageTimestamp(topics []types.TopicType) (uint64, error) {
	return m.persistence.GetOldestWakuMessageTimestamp(topics)
}

func (m *Manager) GetLastMessageArchiveEndDate(communityID types.HexBytes) (uint64, error) {
	return m.persistence.GetLastMessageArchiveEndDate(communityID)
}

func (m *Manager) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		m.LogStdout("failed to get community chats filters", zap.Error(err))
		return 0, err
	}

	if len(filters) == 0 {
		// If we don't have chat filters, we likely don't have any chats
		// associated to this community, which means there's nothing more
		// to do here
		return 0, nil
	}

	topics := []types.TopicType{}

	for _, filter := range filters {
		topics = append(topics, filter.Topic)
	}

	lastArchiveEndDateTimestamp, err := m.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		m.LogStdout("failed to get last archive end date", zap.Error(err))
		return 0, err
	}

	if lastArchiveEndDateTimestamp == 0 {
		// If we don't have a tracked last message archive end date, it
		// means we haven't created an archive before, which means
		// the next thing to look at is the oldest waku message timestamp for
		// this community
		lastArchiveEndDateTimestamp, err = m.GetOldestWakuMessageTimestamp(topics)
		if err != nil {
			m.LogStdout("failed to get oldest waku message timestamp", zap.Error(err))
			return 0, err
		}
		if lastArchiveEndDateTimestamp == 0 {
			// This means there's no waku message stored for this community so far
			// (even after requesting possibly missed messages), so no messages exist yet that can be archived
			m.LogStdout("can't find valid `lastArchiveEndTimestamp`")
			return 0, nil
		}
	}

	return lastArchiveEndDateTimestamp, nil
}

func (m *Manager) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	m.UnseedHistoryArchiveTorrent(communityID)
	_, err := m.CreateHistoryArchiveTorrentFromDB(communityID, topics, startDate, endDate, partition, encrypt)
	if err != nil {
		return err
	}
	return m.SeedHistoryArchiveTorrent(communityID)
}

func (m *Manager) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
	id := community.IDString()
	if _, exists := m.historyArchiveTasks.Load(id); exists {
		m.LogStdout("history archive tasks interval already in progres", zap.String("id", id))
		return
	}

	cancel := make(chan struct{})
	m.historyArchiveTasks.Store(id, cancel)
	m.historyArchiveTasksWaitGroup.Add(1)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.LogStdout("starting history archive tasks interval", zap.String("id", id))
	for {
		select {
		case <-ticker.C:
			m.LogStdout("starting archive task...", zap.String("id", id))
			lastArchiveEndDateTimestamp, err := m.GetHistoryArchivePartitionStartTimestamp(community.ID())
			if err != nil {
				m.LogStdout("failed to get last archive end date", zap.Error(err))
				continue
			}

			if lastArchiveEndDateTimestamp == 0 {
				// This means there are no waku messages for this community,
				// so nothing to do here
				m.LogStdout("couldn't determine archive start date - skipping")
				continue
			}

			topics, err := m.GetCommunityChatsTopics(community.ID())
			if err != nil {
				m.LogStdout("failed to get community chat topics ", zap.Error(err))
				continue
			}

			ts := time.Now().Unix()
			to := time.Unix(ts, 0)
			lastArchiveEndDate := time.Unix(int64(lastArchiveEndDateTimestamp), 0)

			err = m.CreateAndSeedHistoryArchive(community.ID(), topics, lastArchiveEndDate, to, interval, community.Encrypted())
			if err != nil {
				m.LogStdout("failed to create and seed history archive", zap.Error(err))
				continue
			}
		case <-cancel:
			m.UnseedHistoryArchiveTorrent(community.ID())
			m.historyArchiveTasks.Delete(id)
			m.historyArchiveTasksWaitGroup.Done()
			return
		}
	}
}

func (m *Manager) StopHistoryArchiveTasksIntervals() {
	m.historyArchiveTasks.Range(func(_, task interface{}) bool {
		close(task.(chan struct{})) // Need to cast to the chan
		return true
	})
	// Stoping archive interval tasks is async, so we need
	// to wait for all of them to be closed before we shutdown
	// the torrent client
	m.historyArchiveTasksWaitGroup.Wait()
}

func (m *Manager) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {
	task, exists := m.historyArchiveTasks.Load(communityID.String())
	if exists {
		m.logger.Info("Stopping history archive tasks interval", zap.Any("id", communityID.String()))
		close(task.(chan struct{})) // Need to cast to the chan
	}
}

type EncodedArchiveData struct {
	padding int
	bytes   []byte
}

func (m *Manager) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.CreateHistoryArchiveTorrent(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *Manager) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	return m.CreateHistoryArchiveTorrent(communityID, make([]*types.Message, 0), topics, startDate, endDate, partition, encrypt)
}
func (m *Manager) CreateHistoryArchiveTorrent(communityID types.HexBytes, msgs []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	loadFromDB := len(msgs) == 0

	from := startDate
	to := from.Add(partition)
	if to.After(endDate) {
		to = endDate
	}

	archiveDir := m.torrentConfig.DataDir + "/" + communityID.String()
	torrentDir := m.torrentConfig.TorrentDir
	indexPath := archiveDir + "/index"
	dataPath := archiveDir + "/data"

	wakuMessageArchiveIndexProto := &protobuf.WakuMessageArchiveIndex{}
	wakuMessageArchiveIndex := make(map[string]*protobuf.WakuMessageArchiveIndexMetadata)
	archiveIDs := make([]string, 0)

	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		err := os.MkdirAll(archiveDir, 0700)
		if err != nil {
			return archiveIDs, err
		}
	}
	if _, err := os.Stat(torrentDir); os.IsNotExist(err) {
		err := os.MkdirAll(torrentDir, 0700)
		if err != nil {
			return archiveIDs, err
		}
	}

	_, err := os.Stat(indexPath)
	if err == nil {
		wakuMessageArchiveIndexProto, err = m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
		if err != nil {
			return archiveIDs, err
		}
	}

	var offset uint64 = 0

	for hash, metadata := range wakuMessageArchiveIndexProto.Archives {
		offset = offset + metadata.Size
		wakuMessageArchiveIndex[hash] = metadata
	}

	var encodedArchives []*EncodedArchiveData
	topicsAsByteArrays := topicsAsByteArrays(topics)

	m.publish(&Subscription{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.LogStdout("creating archives",
		zap.Any("startDate", startDate),
		zap.Any("endDate", endDate),
		zap.Duration("partition", partition),
	)
	for {
		if from.Equal(endDate) || from.After(endDate) {
			break
		}
		m.LogStdout("creating message archive",
			zap.Any("from", from),
			zap.Any("to", to),
		)

		var messages []types.Message
		if loadFromDB {
			messages, err = m.persistence.GetWakuMessagesByFilterTopic(topics, uint64(from.Unix()), uint64(to.Unix()))
			if err != nil {
				return archiveIDs, err
			}
		} else {
			for _, msg := range msgs {
				if int64(msg.Timestamp) >= from.Unix() && int64(msg.Timestamp) < to.Unix() {
					messages = append(messages, *msg)
				}
			}

		}

		if len(messages) == 0 {
			// No need to create an archive with zero messages
			m.LogStdout("no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]types.Message, 0)
		currentChunkSize := 0
		currentChunk := make([]types.Message, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			if msgSize > maxArchiveSizeInBytes {
				// we drop messages this big
				continue
			}

			if currentChunkSize+msgSize > maxArchiveSizeInBytes {
				messageChunks = append(messageChunks, currentChunk)
				currentChunk = make([]types.Message, 0)
				currentChunkSize = 0
			}
			currentChunk = append(currentChunk, msg)
			currentChunkSize = currentChunkSize + msgSize
		}
		messageChunks = append(messageChunks, currentChunk)

		for _, messages := range messageChunks {
			wakuMessageArchive := m.createWakuMessageArchive(from, to, messages, topicsAsByteArrays)
			encodedArchive, err := proto.Marshal(wakuMessageArchive)
			if err != nil {
				return archiveIDs, err
			}

			if encrypt {
				messageSpec, err := m.encryptor.BuildHashRatchetMessage(communityID, encodedArchive)
				if err != nil {
					return archiveIDs, err
				}

				encodedArchive, err = proto.Marshal(messageSpec.Message)
				if err != nil {
					return archiveIDs, err
				}
			}

			rawSize := len(encodedArchive)
			padding := 0
			size := 0

			if rawSize > pieceLength {
				size = rawSize + pieceLength - (rawSize % pieceLength)
				padding = size - rawSize
			} else {
				padding = pieceLength - rawSize
				size = rawSize + padding
			}

			wakuMessageArchiveIndexMetadata := &protobuf.WakuMessageArchiveIndexMetadata{
				Metadata: wakuMessageArchive.Metadata,
				Offset:   offset,
				Size:     uint64(size),
				Padding:  uint64(padding),
			}

			wakuMessageArchiveIndexMetadataBytes, err := proto.Marshal(wakuMessageArchiveIndexMetadata)
			if err != nil {
				return archiveIDs, err
			}

			archiveID := crypto.Keccak256Hash(wakuMessageArchiveIndexMetadataBytes).String()
			archiveIDs = append(archiveIDs, archiveID)
			wakuMessageArchiveIndex[archiveID] = wakuMessageArchiveIndexMetadata
			encodedArchives = append(encodedArchives, &EncodedArchiveData{bytes: encodedArchive, padding: padding})
			offset = offset + uint64(rawSize) + uint64(padding)
		}

		from = to
		to = to.Add(partition)
		if to.After(endDate) {
			to = endDate
		}
	}

	if len(encodedArchives) > 0 {

		dataBytes := make([]byte, 0)

		for _, encodedArchiveData := range encodedArchives {
			dataBytes = append(dataBytes, encodedArchiveData.bytes...)
			dataBytes = append(dataBytes, make([]byte, encodedArchiveData.padding)...)
		}

		wakuMessageArchiveIndexProto.Archives = wakuMessageArchiveIndex
		indexBytes, err := proto.Marshal(wakuMessageArchiveIndexProto)
		if err != nil {
			return archiveIDs, err
		}

		if encrypt {
			messageSpec, err := m.encryptor.BuildHashRatchetMessage(communityID, indexBytes)
			if err != nil {
				return archiveIDs, err
			}
			indexBytes, err = proto.Marshal(messageSpec.Message)
			if err != nil {
				return archiveIDs, err
			}
		}

		err = os.WriteFile(indexPath, indexBytes, 0644) // nolint: gosec
		if err != nil {
			return archiveIDs, err
		}

		file, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return archiveIDs, err
		}
		defer file.Close()

		_, err = file.Write(dataBytes)
		if err != nil {
			return archiveIDs, err
		}

		metaInfo := metainfo.MetaInfo{
			AnnounceList: defaultAnnounceList,
		}
		metaInfo.SetDefaults()
		metaInfo.CreatedBy = common.PubkeyToHex(&m.identity.PublicKey)

		info := metainfo.Info{
			PieceLength: int64(pieceLength),
		}

		err = info.BuildFromFilePath(archiveDir)
		if err != nil {
			return archiveIDs, err
		}

		metaInfo.InfoBytes, err = bencode.Marshal(info)
		if err != nil {
			return archiveIDs, err
		}

		metaInfoBytes, err := bencode.Marshal(metaInfo)
		if err != nil {
			return archiveIDs, err
		}

		err = os.WriteFile(m.torrentFile(communityID.String()), metaInfoBytes, 0644) // nolint: gosec
		if err != nil {
			return archiveIDs, err
		}

		m.LogStdout("torrent created", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.publish(&Subscription{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.LogStdout("no archives created")
		m.publish(&Subscription{
			NoHistoryArchivesCreatedSignal: &signal.NoHistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	}

	lastMessageArchiveEndDate, err := m.persistence.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		return archiveIDs, err
	}

	if lastMessageArchiveEndDate > 0 {
		err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	} else {
		err = m.persistence.SaveLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	}
	if err != nil {
		return archiveIDs, err
	}
	return archiveIDs, nil
}

func (m *Manager) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	m.UnseedHistoryArchiveTorrent(communityID)

	id := communityID.String()
	torrentFile := m.torrentFile(id)

	metaInfo, err := metainfo.LoadFromFile(torrentFile)
	if err != nil {
		return err
	}

	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return err
	}

	hash := metaInfo.HashInfoBytes()
	m.torrentTasks[id] = hash

	if err != nil {
		return err
	}

	torrent, err := m.torrentClient.AddTorrent(metaInfo)
	if err != nil {
		return err
	}
	torrent.DownloadAll()

	m.publish(&Subscription{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
		},
	})

	magnetLink := metaInfo.Magnet(nil, &info).String()

	m.LogStdout("seeding torrent", zap.String("id", id), zap.String("magnetLink", magnetLink))
	return nil
}

func (m *Manager) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {
	id := communityID.String()

	hash, exists := m.torrentTasks[id]

	if exists {
		torrent, ok := m.torrentClient.Torrent(hash)
		if ok {
			m.logger.Debug("Unseeding and dropping torrent for community: ", zap.Any("id", id))
			torrent.Drop()
			delete(m.torrentTasks, id)

			m.publish(&Subscription{
				HistoryArchivesUnseededSignal: &signal.HistoryArchivesUnseededSignal{
					CommunityID: id,
				},
			})
		}
	}
}

func (m *Manager) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	id := communityID.String()
	hash := m.torrentTasks[id]
	torrent, ok := m.torrentClient.Torrent(hash)
	return ok && torrent.Seeding()
}

func (m *Manager) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return m.historyArchiveDownloadTasks[communityID]
}

func (m *Manager) DeleteHistoryArchiveDownloadTask(communityID string) {
	delete(m.historyArchiveDownloadTasks, communityID)
}

func (m *Manager) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
	m.historyArchiveDownloadTasks[communityID] = task
}

type HistoryArchiveDownloadTaskInfo struct {
	TotalDownloadedArchivesCount int
	TotalArchivesCount           int
	Cancelled                    bool
}

func (m *Manager) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {

	id := communityID.String()

	ml, err := metainfo.ParseMagnetUri(magnetlink)
	if err != nil {
		return nil, err
	}

	m.logger.Debug("adding torrent via magnetlink for community", zap.String("id", id), zap.String("magnetlink", magnetlink))
	torrent, err := m.torrentClient.AddMagnet(magnetlink)
	if err != nil {
		return nil, err
	}

	downloadTaskInfo := &HistoryArchiveDownloadTaskInfo{
		TotalDownloadedArchivesCount: 0,
		TotalArchivesCount:           0,
		Cancelled:                    false,
	}

	m.torrentTasks[id] = ml.InfoHash
	timeout := time.After(20 * time.Second)

	m.LogStdout("fetching torrent info", zap.String("magnetlink", magnetlink))
	select {
	case <-timeout:
		return nil, ErrTorrentTimedout
	case <-cancelTask:
		m.LogStdout("cancelled fetching torrent info")
		downloadTaskInfo.Cancelled = true
		return downloadTaskInfo, nil
	case <-torrent.GotInfo():

		files := torrent.Files()

		i, ok := findIndexFile(files)
		if !ok {
			// We're dealing with a malformed torrent, so don't do anything
			return nil, errors.New("malformed torrent data")
		}

		indexFile := files[i]
		indexFile.Download()

		m.LogStdout("downloading history archive index")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-cancelTask:
				m.LogStdout("cancelled downloading archive index")
				downloadTaskInfo.Cancelled = true
				return downloadTaskInfo, nil
			case <-ticker.C:
				if indexFile.BytesCompleted() == indexFile.Length() {

					index, err := m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
					if err != nil {
						return nil, err
					}

					existingArchiveIDs, err := m.persistence.GetDownloadedMessageArchiveIDs(communityID)
					if err != nil {
						return nil, err
					}

					if len(existingArchiveIDs) == len(index.Archives) {
						m.LogStdout("download cancelled, no new archives")
						return downloadTaskInfo, nil
					}

					downloadTaskInfo.TotalDownloadedArchivesCount = len(existingArchiveIDs)
					downloadTaskInfo.TotalArchivesCount = len(index.Archives)

					archiveHashes := make(archiveMDSlice, 0, downloadTaskInfo.TotalArchivesCount)

					for hash, metadata := range index.Archives {
						archiveHashes = append(archiveHashes, &archiveMetadata{hash: hash, from: metadata.Metadata.From})
					}

					sort.Sort(sort.Reverse(archiveHashes))

					m.publish(&Subscription{
						DownloadingHistoryArchivesStartedSignal: &signal.DownloadingHistoryArchivesStartedSignal{
							CommunityID: communityID.String(),
						},
					})

					for _, hd := range archiveHashes {

						hash := hd.hash
						hasArchive := false

						for _, existingHash := range existingArchiveIDs {
							if existingHash == hash {
								hasArchive = true
								break
							}
						}
						if hasArchive {
							continue
						}

						metadata := index.Archives[hash]
						startIndex := int(metadata.Offset) / pieceLength
						endIndex := startIndex + int(metadata.Size)/pieceLength

						downloadMsg := fmt.Sprintf("downloading data for message archive (%d/%d)", downloadTaskInfo.TotalDownloadedArchivesCount+1, downloadTaskInfo.TotalArchivesCount)
						m.LogStdout(downloadMsg, zap.String("hash", hash))
						m.LogStdout("pieces (start, end)", zap.Any("startIndex", startIndex), zap.Any("endIndex", endIndex-1))
						torrent.DownloadPieces(startIndex, endIndex)

						piecesCompleted := make(map[int]bool)
						for i = startIndex; i < endIndex; i++ {
							piecesCompleted[i] = false
						}

						psc := torrent.SubscribePieceStateChanges()
						downloadTicker := time.NewTicker(1 * time.Second)
						defer downloadTicker.Stop()

					downloadLoop:
						for {
							select {
							case <-downloadTicker.C:
								done := true
								for i = startIndex; i < endIndex; i++ {
									piecesCompleted[i] = torrent.PieceState(i).Complete
									if !piecesCompleted[i] {
										done = false
									}
								}
								if done {
									psc.Close()
									break downloadLoop
								}
							case <-cancelTask:
								m.LogStdout("downloading archive data interrupted")
								downloadTaskInfo.Cancelled = true
								return downloadTaskInfo, nil
							}
						}
						downloadTaskInfo.TotalDownloadedArchivesCount++
						err = m.persistence.SaveMessageArchiveID(communityID, hash)
						if err != nil {
							m.LogStdout("couldn't save message archive ID", zap.Error(err))
							continue
						}
						m.publish(&Subscription{
							HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
								CommunityID: communityID.String(),
								From:        int(metadata.Metadata.From),
								To:          int(metadata.Metadata.To),
							},
						})
					}
					m.publish(&Subscription{
						HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
							CommunityID: communityID.String(),
						},
					})
					m.LogStdout("finished downloading archives")
					return downloadTaskInfo, nil
				}
			}
		}
	}
}

func (m *Manager) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return m.persistence.GetMessageArchiveIDsToImport(communityID)
}

func (m *Manager) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	id := communityID.String()

	index, err := m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
	if err != nil {
		return nil, err
	}

	dataFile, err := os.Open(m.archiveDataFile(id))
	if err != nil {
		return nil, err
	}
	defer dataFile.Close()

	m.LogStdout("extracting messages from history archive", zap.String("archive id", archiveID))
	metadata := index.Archives[archiveID]

	_, err = dataFile.Seek(int64(metadata.Offset), 0)
	if err != nil {
		m.LogStdout("failed to seek archive data file", zap.Error(err))
		return nil, err
	}

	data := make([]byte, metadata.Size-metadata.Padding)
	m.LogStdout("loading history archive data into memory", zap.Float64("data_size_MB", float64(metadata.Size-metadata.Padding)/1024.0/1024.0))
	_, err = dataFile.Read(data)
	if err != nil {
		m.LogStdout("failed failed to read archive data", zap.Error(err))
		return nil, err
	}

	archive := &protobuf.WakuMessageArchive{}

	err = proto.Unmarshal(data, archive)
	if err != nil {
		// The archive data might eb encrypted so we try to decrypt instead first
		var protocolMessage encryption.ProtocolMessage
		err := proto.Unmarshal(data, &protocolMessage)
		if err != nil {
			m.LogStdout("failed to unmarshal protocol message", zap.Error(err))
			return nil, err
		}

		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			m.logger.Debug("failed to decompress community pubkey", zap.Error(err))
			return nil, err
		}
		decryptedBytes, err := m.encryptor.HandleMessage(m.identity, pk, &protocolMessage, make([]byte, 0))
		if err != nil {
			m.LogStdout("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}
		err = proto.Unmarshal(decryptedBytes.DecryptedMessage, archive)
		if err != nil {
			m.LogStdout("failed to unmarshal message archive data", zap.Error(err))
			return nil, err
		}
	}
	return archive.Messages, nil
}

func (m *Manager) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return m.persistence.SetMessageArchiveIDImported(communityID, hash, imported)
}

func (m *Manager) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	id := communityID.String()
	torrentFile := m.torrentFile(id)

	metaInfo, err := metainfo.LoadFromFile(torrentFile)
	if err != nil {
		return "", err
	}

	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return "", err
	}

	return metaInfo.Magnet(nil, &info).String(), nil
}

func (m *Manager) createWakuMessageArchive(from time.Time, to time.Time, messages []types.Message, topics [][]byte) *protobuf.WakuMessageArchive {
	var wakuMessages []*protobuf.WakuMessage

	for _, msg := range messages {
		topic := types.TopicTypeToByteArray(msg.Topic)
		wakuMessage := &protobuf.WakuMessage{
			Sig:          msg.Sig,
			Timestamp:    uint64(msg.Timestamp),
			Topic:        topic,
			Payload:      msg.Payload,
			Padding:      msg.Padding,
			Hash:         msg.Hash,
			ThirdPartyId: msg.ThirdPartyID,
		}
		wakuMessages = append(wakuMessages, wakuMessage)
	}

	metadata := protobuf.WakuMessageArchiveMetadata{
		From:         uint64(from.Unix()),
		To:           uint64(to.Unix()),
		ContentTopic: topics,
	}

	wakuMessageArchive := &protobuf.WakuMessageArchive{
		Metadata: &metadata,
		Messages: wakuMessages,
	}
	return wakuMessageArchive
}

func (m *Manager) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	wakuMessageArchiveIndexProto := &protobuf.WakuMessageArchiveIndex{}

	indexPath := m.archiveIndexFile(communityID.String())
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(indexData, wakuMessageArchiveIndexProto)
	if err != nil {
		return nil, err
	}

	if len(wakuMessageArchiveIndexProto.Archives) == 0 && len(indexData) > 0 {
		// This means we're dealing with an encrypted index file, so we have to decrypt it first
		var protocolMessage encryption.ProtocolMessage
		err := proto.Unmarshal(indexData, &protocolMessage)
		if err != nil {
			return nil, err
		}
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return nil, err
		}
		decryptedBytes, err := m.encryptor.HandleMessage(myKey, pk, &protocolMessage, make([]byte, 0))
		if err != nil {
			return nil, err
		}
		err = proto.Unmarshal(decryptedBytes.DecryptedMessage, wakuMessageArchiveIndexProto)
		if err != nil {
			return nil, err
		}
	}

	return wakuMessageArchiveIndexProto, nil
}

func (m *Manager) TorrentFileExists(communityID string) bool {
	_, err := os.Stat(m.torrentFile(communityID))
	return err == nil
}

func (m *Manager) torrentFile(communityID string) string {
	return m.torrentConfig.TorrentDir + "/" + communityID + ".torrent"
}

func (m *Manager) archiveIndexFile(communityID string) string {
	return m.torrentConfig.DataDir + "/" + communityID + "/index"
}

func (m *Manager) archiveDataFile(communityID string) string {
	return m.torrentConfig.DataDir + "/" + communityID + "/data"
}

func topicsAsByteArrays(topics []types.TopicType) [][]byte {
	var topicsAsByteArrays [][]byte
	for _, t := range topics {
		topic := types.TopicTypeToByteArray(t)
		topicsAsByteArrays = append(topicsAsByteArrays, topic)
	}
	return topicsAsByteArrays
}

func findIndexFile(files []*torrent.File) (index int, ok bool) {
	for i, f := range files {
		if f.DisplayPath() == "index" {
			return i, true
		}
	}
	return 0, false
}

func (m *Manager) GetCommunityTokens(communityID string) ([]*CommunityToken, error) {
	return m.persistence.GetCommunityTokens(communityID)
}

func (m *Manager) GetAllCommunityTokens() ([]*CommunityToken, error) {
	return m.persistence.GetAllCommunityTokens()
}

func (m *Manager) ImageToBase64(uri string) string {
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

func (m *Manager) AddCommunityToken(token *CommunityToken, croppedImage *images.CroppedImage) (*CommunityToken, error) {

	community, err := m.GetByIDString(token.CommunityID)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
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
	} else {
		token.Base64Image = m.ImageToBase64(token.Base64Image)
	}

	tokenMetadata := &protobuf.CommunityTokenMetadata{
		ContractAddresses: map[uint64]string{uint64(token.ChainID): token.Address},
		Description:       token.Description,
		Image:             token.Base64Image,
		Symbol:            token.Symbol,
		TokenType:         token.TokenType,
		Name:              token.Name,
		Decimals:          uint32(token.Decimals),
	}
	_, err = community.AddCommunityTokensMetadata(tokenMetadata)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return token, m.persistence.AddCommunityToken(token)
}

func (m *Manager) UpdateCommunityTokenState(chainID int, contractAddress string, deployState DeployState) error {
	return m.persistence.UpdateCommunityTokenState(chainID, contractAddress, deployState)
}

func (m *Manager) UpdateCommunityTokenSupply(chainID int, contractAddress string, supply *bigint.BigInt) error {
	return m.persistence.UpdateCommunityTokenSupply(chainID, contractAddress, supply)
}

func (m *Manager) SetCommunityActiveMembersCount(communityID string, activeMembersCount uint64) error {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return err
	}
	if community == nil {
		return ErrOrgNotFound
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

// UpdateCommunity takes a Community persists it and republishes it.
// The clock is incremented meaning even a no change update will be republished by the admin, and parsed by the member.
func (m *Manager) UpdateCommunity(c *Community) error {
	c.increaseClock()

	err := m.persistence.SaveCommunity(c)
	if err != nil {
		return err
	}

	m.publish(&Subscription{Community: c})
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

func (m *Manager) accountsHasAdminPermission(becomeAdminPermissions []*protobuf.CommunityTokenPermission, accounts []*AccountChainIDsCombination) bool {
	if len(becomeAdminPermissions) > 0 {
		permissionResponse, err := m.checkPermissionToJoin(becomeAdminPermissions, accounts, true)
		if err != nil {
			m.logger.Warn("check admin permission failed: %v", zap.Error(err))
			return false
		}
		return permissionResponse.Satisfied
	}
	return false
}
