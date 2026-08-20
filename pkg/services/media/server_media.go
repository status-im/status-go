package media

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"

	"go.uber.org/zap"

	errorspkg "github.com/pkg/errors"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/httpserver"
	"github.com/status-im/status-go/internal/ipfs"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

type MediaServerOption func(*MediaServerConfig)

func WithMediaServerDisableTLS(disableTLS bool) MediaServerOption {
	return func(s *MediaServerConfig) {
		s.disableTLS = disableTLS
	}
}

func WithMediaServerAddress(address string) MediaServerOption {
	return func(s *MediaServerConfig) {
		s.address = address
	}
}

func WithMediaServerAdvertizeAddress(host string, port int) MediaServerOption {
	return func(s *MediaServerConfig) {
		s.advertizeHost = host
		s.advertizePort = port
	}
}

type MediaServerConfig struct {
	// disableTLS controls whether the media server uses HTTP instead of HTTPS.
	// Set to true to avoid TLS certificate issues with react-native-fast-image
	// on Android, which has limitations with dynamic certificate updates.
	// Pls check doc/use-status-backend-server.md in status-mobile for more details
	disableTLS bool

	// address is the address to bind the media server to. Defaults to "localhost:0".
	address string

	// AdvertizeHost and AdvertizePort define the a different host/port to be advertized.
	// Check httpserver.Config for more details.
	advertizeHost string
	advertizePort int
}

type Server struct {
	*httpserver.Server

	db                          *sql.DB
	downloader                  *ipfs.Downloader
	multiaccountsDB             *multiaccounts.Database
	walletDB                    *sql.DB
	communityImagesReader       func(communityID string) (map[string]*protobuf.IdentityImage, error)
	communityTokenReader        func(communityID string) ([]*protobuf.CommunityTokenMetadata, error)
	communityImageVersionReader func(communityID string) uint32

	config *MediaServerConfig
}

const (
	mediaServerHostEnvVar = "STATUS_GO_MEDIA_SERVER_HOST"
	mediaServerPortEnvVar = "STATUS_GO_MEDIA_SERVER_PORT"
)

func applyMediaServerAddrEnv(config *MediaServerConfig) error {
	envHost, hasHost := os.LookupEnv(mediaServerHostEnvVar)
	hasHost = hasHost && envHost != ""
	envPort, hasPort := os.LookupEnv(mediaServerPortEnvVar)
	hasPort = hasPort && envPort != ""
	if !hasHost && !hasPort {
		return nil
	}

	host, port, err := net.SplitHostPort(config.address)
	if err != nil {
		return errorspkg.Wrap(err, "failed to parse media server address")
	}

	if hasHost {
		if _, err := netip.ParseAddr(envHost); err != nil {
			return fmt.Errorf("invalid %s %q: must be an IP address", mediaServerHostEnvVar, envHost)
		}
		host = envHost
	}

	if hasPort {
		if p, err := strconv.ParseUint(envPort, 10, 16); err != nil || p == 0 {
			return fmt.Errorf("invalid %s %q: must be an integer in range 1-65535", mediaServerPortEnvVar, envPort)
		}
		port = envPort
	}

	config.address = net.JoinHostPort(host, port)
	return nil
}

func initMediaCertificate(disableTLS bool) (*tls.Certificate, error) {
	if disableTLS {
		return nil, nil
	}

	cert, _, err := generateMediaTLSCert()
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// NewServer returns a *Server
func NewServer(db *sql.DB, downloader *ipfs.Downloader, multiaccountsDB *multiaccounts.Database, walletDB *sql.DB, opts ...MediaServerOption) (*Server, error) {

	s := &Server{
		config: &MediaServerConfig{
			disableTLS:    false,
			address:       "127.0.0.1:0",
			advertizeHost: "",
			advertizePort: 0,
		},
		db:              db,
		downloader:      downloader,
		multiaccountsDB: multiaccountsDB,
		walletDB:        walletDB,
	}

	for _, opt := range opts {
		opt(s.config)
	}

	if err := applyMediaServerAddrEnv(s.config); err != nil {
		return nil, err
	}

	addrPort, err := netip.ParseAddrPort(s.config.address)
	if err != nil {
		return nil, errorspkg.Wrap(err, "failed to parse media server address")
	}

	cert, err := initMediaCertificate(s.config.disableTLS)
	if err != nil {
		return nil, err
	}
	s.Server = httpserver.NewServer(
		logutils.ZapLogger().Named("MediaServer"),
		&httpserver.Config{
			Cert:          cert,
			AddrPort:      addrPort,
			AdvertizeHost: s.config.advertizeHost,
			AdvertizePort: s.config.advertizePort,
		},
	)

	s.SetHandlers(httpserver.HandlerPatternMap{
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

func (s *Server) SetDataProviders(db *sql.DB, walletDB *sql.DB, downloader *ipfs.Downloader) {
	s.db = db
	s.walletDB = walletDB
	s.downloader = downloader
}

func (s *Server) Start() error {
	err := s.Server.Start()
	if err != nil {
		s.GetLogger().Error("failed to start media server", zap.Error(err))
		return err
	}

	port := s.Server.GetPort()
	s.GetLogger().Info("media server started",
		zap.String("listeningAddress", s.Server.GetListeningAddrPort()),
		zap.String("advertisingAddress", s.Server.GetAddrPort()))
	signal.SendMediaServerStarted(port)

	return nil
}

func (s *Server) SetCommunityImageVersionReader(getFunc func(communityID string) uint32) {
	s.communityImageVersionReader = getFunc
}

func (s *Server) getCommunityImageVersion(communityID string) uint32 {
	if s.communityImageVersionReader == nil {
		return 0
	}
	return s.communityImageVersionReader(communityID)
}

func (s *Server) SetCommunityImageReader(getFunc func(communityID string) (map[string]*protobuf.IdentityImage, error)) {
	s.communityImagesReader = getFunc
}

func (s *Server) getCommunityImage(communityID string) (map[string]*protobuf.IdentityImage, error) {
	if s.communityImagesReader == nil {
		return nil, errors.New("community image reader not set")
	}
	return s.communityImagesReader(communityID)
}

func (s *Server) SetCommunityTokensReader(getFunc func(communityID string) ([]*protobuf.CommunityTokenMetadata, error)) {
	s.communityTokenReader = getFunc
}

func (s *Server) getCommunityTokens(communityID string) ([]*protobuf.CommunityTokenMetadata, error) {
	if s.communityTokenReader == nil {
		return nil, errors.New("community token reader not set")
	}
	return s.communityTokenReader(communityID)
}

func (s *Server) MakeImageServerURL() string {
	u := s.MakeBaseURL()
	u.Path = basePath + "/"
	return u.String()
}

func (s *Server) MakeImageURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = imagesPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *Server) MakeLinkPreviewThumbnailURL(msgID string, previewURL string) string {
	u := s.MakeBaseURL()
	u.Path = LinkPreviewThumbnailPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}}.Encode()
	return u.String()
}

func (s *Server) MakeStatusLinkPreviewThumbnailURL(msgID string, previewURL string, imageID common.MediaServerImageID) string {
	u := s.MakeBaseURL()
	u.Path = StatusLinkPreviewThumbnailPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}, "image-id": {string(imageID)}}.Encode()
	return u.String()
}

func (s *Server) MakeLinkPreviewFaviconURL(msgID string, previewURL string) string {
	u := s.MakeBaseURL()
	u.Path = LinkPreviewFaviconPath
	u.RawQuery = url.Values{"message-id": {msgID}, "url": {previewURL}}.Encode()
	return u.String()
}

func (s *Server) MakeDiscordAuthorAvatarURL(authorID string) string {
	u := s.MakeBaseURL()
	u.Path = discordAuthorsPath
	u.RawQuery = url.Values{"authorId": {authorID}}.Encode()

	return u.String()
}

func (s *Server) MakeDiscordAttachmentURL(messageID string, id string) string {
	u := s.MakeBaseURL()
	u.Path = discordAttachmentsPath
	u.RawQuery = url.Values{"messageId": {messageID}, "attachmentId": {id}}.Encode()

	return u.String()
}

func (s *Server) MakeAudioURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = audioPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *Server) MakeStickerURL(stickerHash string) string {
	u := s.MakeBaseURL()
	u.Path = ipfsPath
	u.RawQuery = url.Values{"hash": {stickerHash}}.Encode()

	return u.String()
}

func (s *Server) MakeContactImageURL(publicKey string, imageType string, imageClock uint64) string {
	u := s.MakeBaseURL()
	u.Path = contactImagesPath
	u.RawQuery = url.Values{"publicKey": {publicKey}, "imageName": {imageType}, "clock": {fmt.Sprint(imageClock)}}.Encode()

	return u.String()
}

func (s *Server) MakeAccountImageURL(keyUid string, imageType string, imageClock uint64) string {
	u := s.MakeBaseURL()
	u.Path = accountImagesPath
	u.RawQuery = url.Values{"keyUid": {keyUid}, "imageName": {imageType}, "clock": {fmt.Sprint(imageClock)}}.Encode()

	return u.String()
}

func (s *Server) MakeCommunityTokenImagesURL(communityID string, chainID uint64, symbol string) string {
	u := s.MakeBaseURL()
	u.Path = communityTokenImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"chainID":     {strconv.FormatUint(chainID, 10)},
		"symbol":      {symbol},
	}.Encode()

	return u.String()
}

func (s *Server) MakeCommunityImageURL(communityID, name string) string {
	u := s.MakeBaseURL()
	u.Path = communityDescriptionImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"name":        {name},
		"version":     {fmt.Sprintf("%d", (s.getCommunityImageVersion(communityID)))},
	}.Encode()

	return u.String()
}

func (s *Server) MakeCommunityDescriptionTokenImageURL(communityID, symbol string) string {
	u := s.MakeBaseURL()
	u.Path = communityDescriptionTokenImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
		"symbol":      {symbol},
	}.Encode()

	return u.String()
}

func (s *Server) MakeWalletCommunityImagesURL(communityID string) string {
	u := s.MakeBaseURL()
	u.Path = walletCommunityImagesPath
	u.RawQuery = url.Values{
		"communityID": {communityID},
	}.Encode()

	return u.String()
}

func (s *Server) MakeWalletCollectionImagesURL(contractID thirdparty.ContractID) string {
	u := s.MakeBaseURL()
	u.Path = walletCollectionImagesPath
	u.RawQuery = url.Values{
		"chainID":         {contractID.ChainID.String()},
		"contractAddress": {contractID.Address.Hex()},
	}.Encode()

	return u.String()
}

func (s *Server) MakeWalletCollectibleImagesURL(collectibleID thirdparty.CollectibleUniqueID) string {
	u := s.MakeBaseURL()
	u.Path = walletCollectibleImagesPath
	u.RawQuery = url.Values{
		"chainID":         {collectibleID.ContractID.ChainID.String()},
		"contractAddress": {collectibleID.ContractID.Address.Hex()},
		"tokenID":         {collectibleID.TokenID.String()},
	}.Encode()

	return u.String()
}
