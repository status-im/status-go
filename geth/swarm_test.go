package geth_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/status-im/status-go/geth"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestFileUpload(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	address1, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Error("Test failed: could not create account")
		return
	}

	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}

	file := "../data/testfile"
	fi, err := os.Stat(file)
	if err != nil {
		t.Error(err)
		return
	}

	entry, err := UploadFile(file, fi)
	if err != nil {
		t.Error(err)
		return
	}

	mroot := Manifest{[]ManifestEntry{entry}}
	hash, err := UploadManifest(mroot)
	if err != nil {
		t.Error(err)
		return
	}

	if hash != "4b4d2180d328a0ba551ddba46427f005530a0b160c61965abe6b18b15a389c39" {
		t.Errorf("Hash is not equal 4b4d2180d328a0ba551ddba46427f005530a0b160c61965abe6b18b15a389c39 got(%s)", hash)
		return
	}

	resp, err := GetFile("4b4d2180d328a0ba551ddba46427f005530a0b160c61965abe6b18b15a389c39")
	if err != nil {
		t.Error(err)
		return
	}

	manifestText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}

	manifest := Manifest{}
	json.Unmarshal(manifestText, &manifest)

	resp, err = GetFile(manifest.Entries[0].Hash)
	if err != nil {
		t.Error(err)
		return
	}

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
		return
	}

	if string(text) != "swarm test" {
		t.Errorf("Received file is not equal swarm test: got(%s)", string(text))
	}
}

func TestSwarmNonUnlockWhenRestart(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	_, err = geth.NodeManagerInstance().SwarmService()
	if err != nil {
		t.Errorf("swarm service not running: %v", err)
	}

	address1, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Error("Test failed: could not create account")
		return
	}

	address2, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Error("Test failed: could not create account")
		return
	}

	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Error(err)
		return
	}

	err = geth.SelectAccount(address2, newAccountPassword)
	if err != nil {
		t.Error(err)
		return
	}

	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Error(err)
		return
	}
}

// manifest is the JSON representation of a swarm manifest.
type ManifestEntry struct {
	Hash        string `json:"hash,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Path        string `json:"path,omitempty"`
}

// manifest is the JSON representation of a swarm manifest.
type Manifest struct {
	Entries []ManifestEntry `json:"entries,omitempty"`
}

// upload file to local running swarm
func UploadFile(file string, fi os.FileInfo) (ManifestEntry, error) {
	hash, err := uploadFileContent(file, fi)
	m := ManifestEntry{
		Hash:        hash,
		ContentType: mime.TypeByExtension(filepath.Ext(fi.Name())),
	}
	return m, err
}

func uploadFileContent(file string, fi os.FileInfo) (string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	return postRaw("application/octet-stream", fi.Size(), fd)
}

// upload manifest to local running swarm
func UploadManifest(m Manifest) (string, error) {
	jsm, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return postRaw("application/json", int64(len(jsm)), ioutil.NopCloser(bytes.NewReader(jsm)))
}

func postRaw(mimetype string, size int64, body io.ReadCloser) (string, error) {
	req, err := http.NewRequest("POST", "http://127.0.0.1:8500/bzzr:/", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", mimetype)
	req.ContentLength = size
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	content, err := ioutil.ReadAll(resp.Body)
	return string(content), err
}

// get file from local running swarm
func GetFile(hash string) (resp *http.Response, err error) {
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:8500/bzzr:/%s", hash))

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Insuccess status for file download request - %s", resp.Status)
	}

	return
}
