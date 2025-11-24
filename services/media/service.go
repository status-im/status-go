package media

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"

	errorspkg "github.com/pkg/errors"

	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/signal"
)

type config struct {
	logger *zap.Logger

	// disableTLS controls whether the media server uses HTTP instead of HTTPS.
	// Set to true to avoid TLS certificate issues with react-native-fast-image
	// on Android, which has limitations with dynamic certificate updates.
	// Pls check doc/use-status-backend-server.md in status-mobile for more details
	disableTLS bool

	// address is the address to bind the media server to. Defaults to "localhost:0".
	address string

	// AdvertizeHost and AdvertizePort define a different host/port to be advertized.
	// Check server.Config for more details.
	advertizeHost string
	advertizePort int
}

type Service struct {
	server.Server

	logger                      *zap.Logger
	db                          *sql.DB
	downloader                  *ipfs.Downloader
	multiaccountsDB             *multiaccounts.Database
	walletDB                    *sql.DB
	communityImagesReader       func(communityID string) (map[string]*protobuf.IdentityImage, error)
	communityTokenReader        func(communityID string) ([]*protobuf.CommunityTokenMetadata, error)
	communityImageVersionReader func(communityID string) uint32
}

func initMediaCertificate(disableTLS bool) (*tls.Certificate, error) {
	if disableTLS {
		return nil, nil
	}

	cert, _, err := server.GenerateMediaTLSCert()
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// NewService returns a *Service
func NewService(db *sql.DB, downloader *ipfs.Downloader, multiaccountsDB *multiaccounts.Database, walletDB *sql.DB, opts ...Option) (*Service, error) {
	cfg := &config{
		logger:        zap.NewNop(),
		disableTLS:    false,
		address:       "127.0.0.1:0",
		advertizeHost: "",
		advertizePort: 0,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	s := &Service{
		logger:          cfg.logger,
		db:              db,
		downloader:      downloader,
		multiaccountsDB: multiaccountsDB,
		walletDB:        walletDB,
	}

	addrPort, err := netip.ParseAddrPort(cfg.address)
	if err != nil {
		return nil, errorspkg.Wrap(err, "failed to parse media server address")
	}

	cert, err := initMediaCertificate(cfg.disableTLS)
	if err != nil {
		return nil, err
	}

	s.Server = server.NewServer(
		s.logger.Named("server"),
		&server.Config{
			Cert:          cert,
			AddrPort:      addrPort,
			AdvertizeHost: cfg.advertizeHost,
			AdvertizePort: cfg.advertizePort,
		},
	)

	s.SetHandlers(HandlerPatternMap{
		accountImagesPath:                   s.handleAccountImages,
		accountInitialsPath:                 s.handleAccountInitials,
		audioPath:                           s.handleAudio,
		contactImagesPath:                   s.handleContactImages,
		discordAttachmentsPath:              s.handleDiscordAttachment,
		discordAuthorsPath:                  s.handleDiscordAuthorAvatar,
		imagesPath:                          s.handleImage,
		ipfsPath:                            s.handleIPFS,
		LinkPreviewThumbnailPath:            s.handleLinkPreviewThumbnail,
		LinkPreviewFaviconPath:              s.handleLinkPreviewFavicon,
		StatusLinkPreviewThumbnailPath:      s.handleStatusLinkPreviewThumbnail,
		communityTokenImagesPath:            s.handleCommunityTokenImages,
		communityDescriptionImagesPath:      s.handleCommunityDescriptionImages,
		communityDescriptionTokenImagesPath: s.handleCommunityDescriptionTokenImages,
		walletCommunityImagesPath:           s.handleWalletCommunityImages,
		walletCollectionImagesPath:          s.handleWalletCollectionImages,
		walletCollectibleImagesPath:         s.handleWalletCollectibleImages,
		healthPath:                          s.handleHealth,
	})

	return s, nil
}
func (s *Service) SetDataProviders(db *sql.DB, walletDB *sql.DB, downloader *ipfs.Downloader) {
	s.db = db
	s.walletDB = walletDB
	s.downloader = downloader
}

func (s *Service) Start() error {
	err := s.Server.Start()
	if err != nil {
		s.logger.Error("failed to start media server", zap.Error(err))
		return err
	}

	port := s.Server.GetPort()
	s.logger.Info("media server started",
		zap.Int("listeningPort", port),
		zap.String("advertisingAddress", s.Server.GetAddrPort()))
	signal.SendMediaServerStarted(port)

	return nil
}

func (s *Service) Stop() error {
	return s.Server.Stop()
}

func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "media",
			Version:   "1.0",
			Service:   &API{},
		},
	}
}

func (s *Service) SetCommunityImageVersionReader(getFunc func(communityID string) uint32) {
	s.communityImageVersionReader = getFunc
}

func (s *Service) getCommunityImageVersion(communityID string) uint32 {
	if s.communityImageVersionReader == nil {
		return 0
	}
	return s.communityImageVersionReader(communityID)
}

func (s *Service) SetCommunityImageReader(getFunc func(communityID string) (map[string]*protobuf.IdentityImage, error)) {
	s.communityImagesReader = getFunc
}

func (s *Service) getCommunityImage(communityID string) (map[string]*protobuf.IdentityImage, error) {
	if s.communityImagesReader == nil {
		return nil, errors.New("community image reader not set")
	}
	return s.communityImagesReader(communityID)
}

func (s *Service) SetCommunityTokensReader(getFunc func(communityID string) ([]*protobuf.CommunityTokenMetadata, error)) {
	s.communityTokenReader = getFunc
}

func (s *Service) getCommunityTokens(communityID string) ([]*protobuf.CommunityTokenMetadata, error) {
	if s.communityTokenReader == nil {
		return nil, errors.New("community token reader not set")
	}
	return s.communityTokenReader(communityID)
}

func (s *Service) MakeImageServerURL() string {
	u := s.MakeBaseURL()
	u.Path = basePath + "/"
	return u.String()
}

func (s *Service) MakeImageURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = imagesPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *Service) MakeLinkPreviewThumbnailURL(msgID string, previewURL string) string {
	u := s.MakeBaseURL()
	u.Path = LinkPreviewThumbnailPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}}.Encode()
	return u.String()
}

func (s *Service) MakeStatusLinkPreviewThumbnailURL(msgID string, previewURL string, imageID common.MediaServerImageID) string {
	u := s.MakeBaseURL()
	u.Path = StatusLinkPreviewThumbnailPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}, "image-id": {string(imageID)}}.Encode()
	return u.String()
}

func (s *Service) MakeLinkPreviewFaviconURL(msgID string, previewURL string) string {
	u := s.MakeBaseURL()
	u.Path = LinkPreviewFaviconPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}}.Encode()
	return u.String()
}

func (s *Service) MakeDiscordAuthorAvatarURL(authorID string) string {
	u := s.MakeBaseURL()
	u.Path = discordAuthorsPath
	u.RawQuery = url.Values{"authorId": {authorID}}.Encode()

	return u.String()
}

func (s *Service) MakeDiscordAttachmentURL(messageID string, id string) string {
	u := s.MakeBaseURL()
	u.Path = discordAttachmentsPath
	u.RawQuery = url.Values{"messageId": {messageID}, "attachmentId": {id}}.Encode()

	return u.String()
}

func (s *Service) MakeAudioURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = audioPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *Service) MakeStickerURL(stickerHash string) string {
	u := s.MakeBaseURL()
	u.Path = ipfsPath
	u.RawQuery = url.Values{"hash": {stickerHash}}.Encode()

	return u.String()
}

func (s *Service) MakeContactImageURL(publicKey string, imageType string, imageClock uint64) string {
	u := s.MakeBaseURL()
	u.Path = contactImagesPath
	u.RawQuery = url.Values{"publicKey": {publicKey}, "imageName": {imageType}, "clock": {fmt.Sprint(imageClock)}}.Encode()

	return u.String()
}

func (s *Service) MakeAccountImageURL(keyUid string, imageType string, imageClock uint64) string {
	u := s.MakeBaseURL()
	u.Path = accountImagesPath
	u.RawQuery = url.Values{"keyUid": {keyUid}, "imageName": {imageType}, "clock": {fmt.Sprint(imageClock)}}.Encode()

	return u.String()
}

func (s *Service) MakeCommunityTokenImagesURL(communityID string, chainID uint64, symbol string) string {
	u := s.MakeBaseURL()
	u.Path = communityTokenImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"chainID":     {strconv.FormatUint(chainID, 10)},
		"symbol":      {symbol},
	}.Encode()

	return u.String()
}

func (s *Service) MakeCommunityImageURL(communityID, name string) string {
	u := s.MakeBaseURL()
	u.Path = communityDescriptionImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"name":        {name},
		"version":     {fmt.Sprintf("%d", (s.getCommunityImageVersion(communityID)))},
	}.Encode()

	return u.String()
}

func (s *Service) MakeCommunityDescriptionTokenImageURL(communityID, symbol string) string {
	u := s.MakeBaseURL()
	u.Path = communityDescriptionTokenImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"symbol":      {symbol},
	}.Encode()

	return u.String()
}

func (s *Service) MakeWalletCommunityImagesURL(communityID string) string {
	u := s.MakeBaseURL()
	u.Path = walletCommunityImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
	}.Encode()

	return u.String()
}

func (s *Service) MakeWalletCollectionImagesURL(contractID thirdparty.ContractID) string {
	u := s.MakeBaseURL()
	u.Path = walletCollectionImagesPath
	u.RawQuery = url.Values{
		"chainID":         {contractID.ChainID.String()},
		"contractAddress": {contractID.Address.Hex()},
	}.Encode()

	return u.String()
}

func (s *Service) MakeWalletCollectibleImagesURL(collectibleID thirdparty.CollectibleUniqueID) string {
	u := s.MakeBaseURL()
	u.Path = walletCollectibleImagesPath
	u.RawQuery = url.Values{
		"chainID":         {collectibleID.ContractID.ChainID.String()},
		"contractAddress": {collectibleID.ContractID.Address.Hex()},
		"tokenID":         {collectibleID.TokenID.String()},
	}.Encode()

	return u.String()
}
