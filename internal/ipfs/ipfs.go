package ipfs

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/wealdtech/go-multicodec"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/pausable"
)

const maxRequestsPerSecond = 3

// ErrDownloaderStopped is returned by Get for requests that cannot be served
// because the downloader is shutting down.
var ErrDownloaderStopped = errors.New("ipfs downloader stopped")

type taskResponse struct {
	err      error
	response []byte
}

type taskRequest struct {
	cid      string
	download bool
	doneChan chan taskResponse
}

type Downloader struct {
	pausable.PauseBroadcaster

	ctx             context.Context
	cancel          func()
	ipfsDir         string
	wg              sync.WaitGroup
	rateLimiterChan chan taskRequest
	inputTaskChan   chan taskRequest
	client          *http.Client

	// stopMu orders wg.Add against wg.Wait: Get takes it for reading before
	// adding, Stop takes it for writing to close quit, so no Add can slip in
	// once Stop has begun waiting.
	stopMu   sync.RWMutex
	stopOnce sync.Once
	quit     chan struct{}
}

func NewDownloader(rootDir string) *Downloader {
	ipfsDir := filepath.Clean(filepath.Join(rootDir, "./ipfs"))
	if err := os.MkdirAll(ipfsDir, 0700); err != nil {
		panic("could not create IPFSDir")
	}

	ctx, cancel := context.WithCancel(context.TODO())

	d := &Downloader{
		ctx:             ctx,
		cancel:          cancel,
		ipfsDir:         ipfsDir,
		rateLimiterChan: make(chan taskRequest, maxRequestsPerSecond),
		inputTaskChan:   make(chan taskRequest, 1000),
		wg:              sync.WaitGroup{},
		client: &http.Client{
			Timeout: time.Second * 5,
		},

		quit: make(chan struct{}),
	}

	// Tracked on the same WaitGroup as Get callers so Stop cannot return while
	// either goroutine is still touching the ipfs dir. wg.Done is deferred after
	// the panic guard so it still runs before LogOnPanic re-raises.
	d.wg.Add(2)
	go func() {
		defer panics.LogOnPanic()
		defer d.wg.Done()
		d.taskDispatcher()
	}()
	go func() {
		defer panics.LogOnPanic()
		defer d.wg.Done()
		d.worker()
	}()

	return d
}

func (d *Downloader) Stop() {
	d.stopOnce.Do(func() {
		d.stopMu.Lock()
		close(d.quit)
		d.stopMu.Unlock()
	})

	d.cancel()

	d.wg.Wait()
	// The task channels are deliberately left open: callers racing with Stop
	// select on quit instead, and a closed channel would turn that race into a
	// send-on-closed-channel panic.
}

func (d *Downloader) worker() {
	for {
		select {
		case <-d.quit:
			return
		case request := <-d.rateLimiterChan:
			resp, err := d.download(request.cid, request.download)
			// doneChan is buffered, so an abandoned request never blocks here.
			request.doneChan <- taskResponse{
				err:      err,
				response: resp,
			}
		}
	}
}

func (d *Downloader) taskDispatcher() {
	sub := d.Subscribe()
	defer sub.Unsubscribe()
	pt := pausable.NewPausableTicker(pausable.PausableTickerConfig{
		Interval: time.Second / maxRequestsPerSecond,
		OnTick: func() {
			select {
			case request := <-d.inputTaskChan:
				// Quit-aware: rateLimiterChan may be full with no worker left to
				// drain it, which would park this goroutine for good.
				select {
				case d.rateLimiterChan <- request:
				case <-d.quit:
				}
			default:
			}
		},
	}, sub.C())
	pt.Run(d.quit)
}

func hashToCid(hash []byte) (string, error) {
	// contract response includes a contenthash, which needs to be decoded to reveal
	// an IPFS identifier. Once decoded, download the content from IPFS. This content
	// is in EDN format, ie https://ipfs.infura.io/ipfs/QmWVVLwVKCwkVNjYJrRzQWREVvEk917PhbHYAUhA1gECTM
	// and it also needs to be decoded in to a nim type

	data, codec, err := multicodec.RemoveCodec(hash)
	if err != nil {
		return "", err
	}

	codecName, err := multicodec.Name(codec)
	if err != nil {
		return "", err
	}

	if codecName != "ipfs-ns" {
		return "", errors.New("codecName is not ipfs-ns")
	}

	thisCID, err := cid.Parse(data)
	if err != nil {
		return "", err
	}

	return thisCID.Hash().B58String(), nil
}

func decodeStringHash(input string) (string, error) {
	hash, err := hexutil.Decode("0x" + input)
	if err != nil {
		return "", err
	}

	cid, err := hashToCid(hash)
	if err != nil {
		return "", err
	}

	return cid, nil
}

// Get checks if an IPFS image exists and returns it from cache
// otherwise downloads it from INFURA's ipfs gateway
func (d *Downloader) Get(hash string, download bool) ([]byte, error) {
	cid, err := decodeStringHash(hash)
	if err != nil {
		return nil, err
	}

	exists, content, err := d.exists(cid)
	if err != nil {
		return nil, err
	}
	if exists {
		return content, nil
	}

	doneChan := make(chan taskResponse, 1)

	// Register under the read lock so Stop cannot begin waiting between the
	// quit check and the Add.
	d.stopMu.RLock()
	select {
	case <-d.quit:
		d.stopMu.RUnlock()
		return nil, ErrDownloaderStopped
	default:
	}
	d.wg.Add(1)
	d.stopMu.RUnlock()
	defer d.wg.Done()

	// Both hand-offs must give up on quit: Stop waits on wg, so a request left
	// queued here would deadlock shutdown.
	select {
	case d.inputTaskChan <- taskRequest{
		cid:      cid,
		download: download,
		doneChan: doneChan,
	}:
	case <-d.quit:
		return nil, ErrDownloaderStopped
	}

	return d.awaitResult(doneChan)
}

func (d *Downloader) awaitResult(doneChan chan taskResponse) ([]byte, error) {
	select {
	case done := <-doneChan:
		return done.response, done.err
	case <-d.quit:
		// select picks uniformly among ready cases, so re-check: a result the
		// worker already delivered beats the shutdown error.
		select {
		case done := <-doneChan:
			return done.response, done.err
		default:
			return nil, ErrDownloaderStopped
		}
	}
}

func (d *Downloader) exists(cid string) (bool, []byte, error) {
	path := filepath.Join(d.ipfsDir, cid)
	_, err := os.Stat(path)
	if err == nil {
		fileContent, err := os.ReadFile(path)
		return true, fileContent, err
	}

	return false, nil, nil
}

func (d *Downloader) download(cid string, download bool) ([]byte, error) {
	path := filepath.Join(d.ipfsDir, cid)

	req, err := http.NewRequest(http.MethodGet, GatewayURL+cid, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(d.ctx)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logutils.ZapLogger().Error("failed to close the stickerpack request body", zap.Error(err))
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logutils.ZapLogger().Error("could not load data for", zap.String("cid", cid), zap.Int("code", resp.StatusCode))
		return nil, errors.New("could not load ipfs data")
	}

	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if download {
		// #nosec G306
		err = os.WriteFile(path, fileContent, 0700)
		if err != nil {
			return nil, err
		}
	}

	return fileContent, nil
}
