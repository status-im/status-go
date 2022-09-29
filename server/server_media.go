package server

import (
	"database/sql"
	"net/url"

	"github.com/status-im/status-go/signal"

	"github.com/status-im/status-go/multiaccounts"

	"github.com/status-im/status-go/ipfs"
)

type MediaServer struct {
	Server

	db              *sql.DB
	downloader      *ipfs.Downloader
	multiaccountsDB *multiaccounts.Database
}

// NewMediaServer returns a *MediaServer
func NewMediaServer(db *sql.DB, downloader *ipfs.Downloader, multiaccountsDB *multiaccounts.Database) (*MediaServer, error) {
	err := generateTLSCert()
	if err != nil {
		return nil, err
	}

	s := &MediaServer{
		Server:          NewServer(globalCertificate, localhost, signal.SendMediaServerStarted),
		db:              db,
		downloader:      downloader,
		multiaccountsDB: multiaccountsDB,
	}
	s.SetHandlers(HandlerPatternMap{
		imagesPath:             handleImage(s.db, s.logger),
		audioPath:              handleAudio(s.db, s.logger),
		identiconsPath:         handleIdenticon(s.logger),
		ipfsPath:               handleIPFS(s.downloader, s.logger),
		accountImagesPath:      handleAccountImages(s.multiaccountsDB, s.logger),
		contactImagesPath:      handleContactImages(s.db, s.logger),
		discordAuthorsPath:     handleDiscordAuthorAvatar(s.db, s.logger),
		discordAttachmentsPath: handleDiscordAttachment(s.db, s.logger),
	})

	return s, nil
}

func (s *MediaServer) MakeImageServerURL() string {
	u := s.MakeBaseURL()
	u.Path = basePath + "/"
	return u.String()
}

func (s *MediaServer) MakeIdenticonURL(from string) string {
	u := s.MakeBaseURL()
	u.Path = identiconsPath
	u.RawQuery = url.Values{"publicKey": {from}}.Encode()

	return u.String()
}

func (s *MediaServer) MakeImageURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = imagesPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *MediaServer) MakeDiscordAuthorAvatarURL(authorID string) string {
	u := s.MakeBaseURL()
	u.Path = discordAuthorsPath
	u.RawQuery = url.Values{"authorId": {authorID}}.Encode()

	return u.String()
}

func (s *MediaServer) MakeDiscordAttachmentURL(messageID string, id string) string {
	u := s.MakeBaseURL()
	u.Path = discordAttachmentsPath
	u.RawQuery = url.Values{"messageId": {messageID}, "attachmentId": {id}}.Encode()

	return u.String()
}

func (s *MediaServer) MakeAudioURL(id string) string {
	u := s.MakeBaseURL()
	u.Path = audioPath
	u.RawQuery = url.Values{"messageId": {id}}.Encode()

	return u.String()
}

func (s *MediaServer) MakeStickerURL(stickerHash string) string {
	u := s.MakeBaseURL()
	u.Path = ipfsPath
	u.RawQuery = url.Values{"hash": {stickerHash}}.Encode()

	return u.String()
}
