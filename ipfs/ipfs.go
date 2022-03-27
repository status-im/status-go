package ipfs

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/wealdtech/go-multicodec"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

const infuraAPIURL = "https://ipfs.infura.io:5001/api/v0/cat?arg="
const maxRequestsPerSecond = 3

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
	ipfsDir         string
	rateLimiterChan chan taskRequest
	inputTaskChan   chan taskRequest
	client          *http.Client

	quit chan struct{}
}

func NewDownloader(rootDir string) *Downloader {
	ipfsDir := filepath.Clean(filepath.Join(rootDir, "./ipfs"))
	if err := os.MkdirAll(ipfsDir, 0700); err != nil {
		panic("could not create IPFSDir")
	}

	d := &Downloader{
		ipfsDir:         ipfsDir,
		rateLimiterChan: make(chan taskRequest, maxRequestsPerSecond),
		inputTaskChan:   make(chan taskRequest, 1000),
		client: &http.Client{
			Timeout: time.Second * 5,
		},

		quit: make(chan struct{}, 1),
	}

	go d.taskDispatcher()
	go d.worker()

	return d
}

func (d *Downloader) Stop() {
	close(d.rateLimiterChan)
	close(d.inputTaskChan)
}

func (d *Downloader) worker() {
	for request := range d.rateLimiterChan {
		resp, err := d.download(request.cid, request.download)
		request.doneChan <- taskResponse{
			err:      err,
			response: resp,
		}
	}
}

func (d *Downloader) taskDispatcher() {
	rate := time.Tick(time.Second / maxRequestsPerSecond)
	for range rate {
		request := <-d.inputTaskChan
		d.rateLimiterChan <- request
	}
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

	return thisCID.StringOfBase(multibase.Base32)
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
	d.inputTaskChan <- taskRequest{
		cid:      cid,
		download: download,
		doneChan: doneChan,
	}

	done := <-doneChan
	close(doneChan)

	return done.response, done.err
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

	req, err := http.NewRequest(http.MethodPost, infuraAPIURL+cid, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("failed to close the stickerpack request body", "err", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Error("could not load data for", "cid", cid, "code", resp.StatusCode)
		return nil, errors.New("could not load ipfs data")
	}

	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if download {
		err = os.WriteFile(path, fileContent, 0700)
		if err != nil {
			return nil, err
		}
	}

	return fileContent, nil
}
