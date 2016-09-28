package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
*/
import "C"
import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var muPrepareTestNode sync.Mutex

const (
	TestDataDir         = "../.ethereumtest"
	TestNodeSyncSeconds = 420
)

type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = func(jsonEvent string) { // internal signal handler (used in tests)
	glog.V(logger.Info).Infof("notification received (default notification handler): %s\n", jsonEvent)
}

func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	notificationHandler = fn
}

//export NotifyNode
func NotifyNode(jsonEvent *C.char) {
	notificationHandler(C.GoString(jsonEvent))
}

// export TriggerTestSignal
func TriggerTestSignal() {
	C.StatusServiceSignalEvent(C.CString(`{"answer": 42}`))
}

func CopyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return nil
}

// LoadFromFile is usefull for loading test data, from testdata/filename into a variable
func LoadFromFile(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes())
}

func PrepareTestNode() (err error) {
	muPrepareTestNode.Lock()
	defer muPrepareTestNode.Unlock()

	manager := GetNodeManager()
	if manager.HasNode() {
		return nil
	}

	syncRequired := false
	if _, err := os.Stat(TestDataDir); os.IsNotExist(err) {
		syncRequired = true
	}

	// prepare node directory
	dataDir, err := PreprocessDataDir(TestDataDir)
	if err != nil {
		glog.V(logger.Warn).Infoln("make node failed:", err)
		return err
	}

	// import test account (with test ether on it)
	dst := filepath.Join(TestDataDir, "testnet", "keystore", "test-account.pk")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		err = CopyFile(dst, filepath.Join("../data", "test-account.pk"))
		if err != nil {
			glog.V(logger.Warn).Infof("cannot copy test account PK: %v", err)
			return err
		}
	}

	// start geth node and wait for it to initialize
	// internally once.Do() is used, so call below is thread-safe
	CreateAndRunNode(dataDir, 8546) // to avoid conflicts with running react-native app, run on different port

	manager = GetNodeManager()
	if !manager.HasNode() {
		panic(ErrInvalidGethNode)
	}
	if !manager.HasClientRestartWrapper() {
		panic(ErrInvalidGethNode)
	}
	if !manager.HasWhisperService() {
		panic(ErrInvalidGethNode)
	}
	if !manager.HasLightEthereumService() {
		panic(ErrInvalidGethNode)
	}

	manager.AddPeer("enode://409772c7dea96fa59a912186ad5bcdb5e51b80556b3fe447d940f99d9eaadb51d4f0ffedb68efad232b52475dd7bd59b51cee99968b3cc79e2d5684b33c4090c@139.162.166.59:30303")

	if syncRequired {
		glog.V(logger.Warn).Infof("Sync is required, it will take %d seconds", TestNodeSyncSeconds)
		time.Sleep(TestNodeSyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
	} else {
		time.Sleep(5 * time.Second)
	}

	return nil
}

func RemoveTestNode() {
	err := os.RemoveAll(TestDataDir)
	if err != nil {
		glog.V(logger.Warn).Infof("could not clean up temporary datadir")
	}
}

func PreprocessDataDir(dataDir string) (string, error) {
	testDataDir := path.Join(dataDir, "testnet", "keystore")
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testDataDir, 0755); err != nil {
			return dataDir, ErrDataDirPreprocessingFailed
		}
	}

	// copy over static peer nodes list (LES auto-discovery is not stable yet)
	dst := filepath.Join(dataDir, "testnet", "static-nodes.json")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		src := filepath.Join("../data", "static-nodes.json")
		if err := CopyFile(dst, src); err != nil {
			return dataDir, err
		}
	}

	return dataDir, nil
}

// PanicAfter throws panic() after waitSeconds, unless abort channel receives notification
func PanicAfter(waitSeconds time.Duration, abort chan bool, desc string) {
	// panic if function takes too long
	timeout := make(chan bool, 1)

	go func() {
		time.Sleep(waitSeconds)
		timeout <- true
	}()

	go func() {
		select {
		case <-abort:
			return
		case <-timeout:
			panic("whatever you were doing takes toooo long: " + desc)
		}
	}()
}
