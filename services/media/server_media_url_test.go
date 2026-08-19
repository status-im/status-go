package media

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/httpserver"
	"github.com/status-im/status-go/protocol/common"
)

func TestServerURLSuite(t *testing.T) {
	suite.Run(t, new(ServerURLSuite))
}

type ServerURLSuite struct {
	suite.Suite

	baseURL string
	server  *Server
}

func (s *ServerURLSuite) SetupTest() {
	port := gofakeit.Number(1000, 65535)
	ip := gofakeit.IPv4Address()
	s.baseURL = fmt.Sprintf("http://%s:%d", ip, port)

	srv := httpserver.NewServer(nil, nil)
	srv.SetURLStateForTest(&net.TCPAddr{IP: net.ParseIP(ip), Port: port}, 0, nil)
	s.server = &Server{Server: srv}
}

func (s *ServerURLSuite) TestServer_MakeBaseURL() {
	s.Require().Equal(s.baseURL, s.server.MakeBaseURL().String())
}

func (s *ServerURLSuite) TestServer_MakeImageServerURL() {
	s.Require().Equal(s.baseURL+"/messages/", s.server.MakeImageServerURL())
}

func (s *ServerURLSuite) TestServer_MakeImageURL() {
	s.Require().Equal(
		s.baseURL+"/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"))
}

func (s *ServerURLSuite) TestServer_MakeImageURL_WhenStopped_UsesCachedPort() {
	s.server.SetURLStateForTest(nil, 8543, &httpserver.Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
	})

	s.Require().Equal(
		"http://127.0.0.1:8543/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"),
	)
}

func (s *ServerURLSuite) TestServer_MakeImageURL_WhenStopped_UsesConfiguredPort() {
	s.server.SetURLStateForTest(nil, 0, &httpserver.Config{
		AddrPort: netip.MustParseAddrPort("127.0.0.1:9020"),
	})

	s.Require().Equal(
		"http://127.0.0.1:9020/messages/images?messageId=0x10aded70ffee",
		s.server.MakeImageURL("0x10aded70ffee"),
	)
}

func (s *ServerURLSuite) TestServer_MakeLinkPreviewThumbnailURL() {
	s.Require().Equal(
		s.baseURL+"/link-preview/thumbnail?message-id=99&url=https%3A%2F%2Fgithub.com",
		s.server.MakeLinkPreviewThumbnailURL("99", "https://github.com"))
}

func (s *ServerURLSuite) TestServer_MakeStatusLinkPreviewThumbnailURL() {
	s.Require().Equal(
		s.baseURL+"/status-link-preview/thumbnail?image-id=contact-icon&message-id=99&url=https%3A%2F%2Fstatus.app",
		s.server.MakeStatusLinkPreviewThumbnailURL("99", "https://status.app", common.MediaServerContactIcon))
}

func (s *ServerURLSuite) TestServer_MakeAudioURL() {
	s.Require().Equal(
		s.baseURL+"/messages/audio?messageId=0xde1e7ebee71e",
		s.server.MakeAudioURL("0xde1e7ebee71e"))
}

func (s *ServerURLSuite) TestServer_MakeStickerURL() {
	s.Require().Equal(
		s.baseURL+"/ipfs?hash=0xdeadbeef4ac0",
		s.server.MakeStickerURL("0xdeadbeef4ac0"))
}

func (s *ServerURLSuite) TestServer_MakeContactImageURL() {
	s.Require().Equal(
		s.baseURL+"/contactImages?clock=1&imageName=Test&publicKey=0x1",
		s.server.MakeContactImageURL("0x1", "Test", uint64(1)))
}

func TestMediaServerURLsRemainAbsoluteAfterBackground(t *testing.T) {
	mediaServer, err := NewServer(nil, nil, nil, nil, WithMediaServerAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("failed to create media server: %v", err)
	}

	if err := mediaServer.Start(); err != nil {
		t.Fatalf("failed to start media server: %v", err)
	}
	t.Cleanup(func() {
		_ = mediaServer.Stop()
	})

	expectedBase := mediaServer.MakeBaseURL().String()
	if expectedBase == "" {
		t.Fatalf("expected non-empty base URL after Start")
	}

	expectedParsed, err := url.Parse(expectedBase)
	if err != nil {
		t.Fatalf("failed to parse expected base URL %q: %v", expectedBase, err)
	}
	if expectedParsed.Scheme == "" || expectedParsed.Host == "" {
		t.Fatalf("expected absolute base URL, got %q", expectedBase)
	}

	if mediaServer.CachedPort() == 0 {
		t.Fatalf("expected cachedPort to be set after Start")
	}
	if expectedParsed.Host != net.JoinHostPort("localhost", fmt.Sprintf("%d", mediaServer.CachedPort())) &&
		expectedParsed.Host != net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", mediaServer.CachedPort())) {
		t.Fatalf("expected host+cachedPort, got host %q and cachedPort %d", expectedParsed.Host, mediaServer.CachedPort())
	}

	mediaServer.ToBackground()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !mediaServer.IsRunning() && mediaServer.ListeningAddr() == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if mediaServer.IsRunning() {
		t.Fatalf("expected media server to stop after ToBackground")
	}
	if mediaServer.ListeningAddr() != nil {
		t.Fatalf("expected media server address to be nil after ToBackground, got %v", mediaServer.ListeningAddr())
	}

	imageURL := mediaServer.MakeImageURL("xyz")
	imageServerURL := mediaServer.MakeImageServerURL()

	if strings.HasPrefix(imageURL, "/messages/") {
		t.Fatalf("expected absolute image URL, got relative %q", imageURL)
	}
	if strings.HasPrefix(imageServerURL, "/messages/") {
		t.Fatalf("expected absolute image server URL, got relative %q", imageServerURL)
	}

	parsedImageURL, err := url.Parse(imageURL)
	if err != nil {
		t.Fatalf("failed to parse image URL %q: %v", imageURL, err)
	}
	if parsedImageURL.Scheme == "" || parsedImageURL.Host == "" {
		t.Fatalf("expected absolute image URL, got %q", imageURL)
	}

	parsedImageServerURL, err := url.Parse(imageServerURL)
	if err != nil {
		t.Fatalf("failed to parse image server URL %q: %v", imageServerURL, err)
	}
	if parsedImageServerURL.Scheme == "" || parsedImageServerURL.Host == "" {
		t.Fatalf("expected absolute image server URL, got %q", imageServerURL)
	}

	if !strings.HasPrefix(imageURL, expectedBase+"/messages/images?") {
		t.Fatalf("expected image URL to keep base %q, got %q", expectedBase, imageURL)
	}
	if !strings.HasPrefix(imageServerURL, expectedBase+"/messages/") {
		t.Fatalf("expected image server URL to keep base %q, got %q", expectedBase, imageServerURL)
	}
}
