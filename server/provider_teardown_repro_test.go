package server

import (
	"net/netip"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/ipfs"
	"github.com/status-im/status-go/protocol/protobuf"
)

// Valid ENS/sticker contenthash; decodeStringHash succeeds so Get() reaches inputTaskChan.
const validIPFSContentHash = "e3010170122050efc0a3e661339f31e1e44b3d15a1bf4e501c965a0523f57b701667fa90ccca"

func newReproMediaServer() *MediaServer {
	return &MediaServer{
		Server: NewServer(zap.NewNop(), &Config{
			AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
		}),
	}
}

// Repro: StatusNode.Stop snapshots-then-stops the downloader under an in-flight
// handleIPFS. Get() sends on a closed inputTaskChan and panics.
//
//	n.mediaServer.SetDataProviders(nil, nil, nil)
//	n.downloader.Stop()
func TestHandleIPFSDoesNotPanicWhenDownloaderStoppedUnderInflightRequest(t *testing.T) {
	d := ipfs.NewDownloader(t.TempDir())
	s := newReproMediaServer()
	s.SetDataProviders(nil, nil, d)

	// Same two-step read handleIPFS does: copy the pointer, then use it.
	downloader := s.ipfsDownloader()
	require.NotNil(t, downloader)

	s.SetDataProviders(nil, nil, nil)
	d.Stop()

	require.NotPanics(t, func() {
		_, _ = downloader.Get(validIPFSContentHash, false)
	})
}

// Repro: community reader function pointers are written without providersMu
// while HTTP handlers already read them. Media server starts before messenger
// calls SetCommunityImageReader. Run with -race.
func TestCommunityImageReaderRacesHandlers(t *testing.T) {
	s := newReproMediaServer()

	png := []byte{0x89, 0x50, 0x4E, 0x47}
	makeReader := func(id int) func(string) (map[string]*protobuf.IdentityImage, error) {
		return func(string) (map[string]*protobuf.IdentityImage, error) {
			_ = id
			return map[string]*protobuf.IdentityImage{
				"thumb": {Payload: png},
			}, nil
		}
	}
	s.SetCommunityImageReader(makeReader(0))

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = s.getCommunityImage("c1")
			}
		})
	}

	wg.Go(func() {
		for i := range 10000 {
			s.SetCommunityImageReader(makeReader(i))
		}
		close(stop)
	})

	wg.Wait()
}
