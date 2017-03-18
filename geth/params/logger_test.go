package params_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/status-im/status-go/geth/params"
)

func TestLogger(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-logger-tests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	nodeConfig, err := params.NewNodeConfig(tmpDir, params.TestNetworkId)
	if err != nil {
		t.Fatal("cannot create config object")
	}
	nodeLogger, err := params.SetupLogger(nodeConfig)
	if err != nil {
		t.Fatal("cannot create logger object")
	}
	if nodeLogger != nil {
		t.Fatalf("logger is not empty (while logs are disabled): %v", nodeLogger)
	}

	nodeConfig.LogEnabled = true
	nodeLogger, err = params.SetupLogger(nodeConfig)
	if err != nil {
		t.Fatal("cannot create logger object")
	}
	if nodeLogger == nil {
		t.Fatal("logger is empty (while logs are enabled)")
	}

	logReader := make(chan string, 100)
	loggerStarted := nodeLogger.Observe(logReader)
	<-loggerStarted // allow logger to setup itself

	expectedLogTextInLogFile := "" // aggregate log contents accross all tests
	validateLoggerObserverText := func(observer chan string, expectedLogText string) {
		logText := ""

		select {
		case logText = <-observer:
			expectedLogTextInLogFile += logText + "\n"
			logText = logText[len(logText)-len(expectedLogText):] // as logs can be prepended with glog info
		case <-time.After(3 * time.Second):
		}

		if logText != expectedLogText {
			t.Fatalf("invalid log, expected: %#v, got: %#v", expectedLogText, logText)
		}
	}

	loggerTestCases := []struct {
		name     string
		log      func()
		validate func()
	}{
		{
			"log using standard log package",
			func() {
				log.Println("use standard log package")
			},
			func() {
				validateLoggerObserverText(logReader, "use standard log package")
			},
		},
		{
			"log using standard glog package",
			func() {
				glog.V(logger.Info).Infoln("use glog package")
			},
			func() {
				validateLoggerObserverText(logReader, "use glog package")
			},
		},
		{
			"log using os.Stderr (write directly to it)",
			func() {
				fmt.Fprintln(os.Stderr, "use os.Stderr package")
			},
			func() {
				validateLoggerObserverText(logReader, "use os.Stderr package")
			},
		},
		{
			"log using DEBUG log level (with appropriate level set)",
			func() {
				nodeLogger.SetV("DEBUG")
				glog.V(logger.Debug).Info("logged DEBUG log level message")
			},
			func() {
				validateLoggerObserverText(logReader, "logged DEBUG log level message")
			},
		},
		{
			"log using DEBUG log level (with appropriate level NOT set)",
			func() {
				nodeLogger.SetV("INFO")
				glog.V(logger.Info).Info("logged INFO log level message")
				glog.V(logger.Debug).Info("logged DEBUG log level message")
			},
			func() {
				validateLoggerObserverText(logReader, "logged INFO log level message")
			},
		},
	}

	for _, testCase := range loggerTestCases {
		t.Log("test: " + testCase.name)
		testCase.log()
		testCase.validate()
	}

	logFileContents, err := ioutil.ReadFile(filepath.Join(tmpDir, nodeConfig.LogFile))
	if err != nil {
		t.Fatalf("cannot read logs file: %v", err)
	}
	if string(logFileContents) != expectedLogTextInLogFile {
		t.Fatalf("wrong content of log file, expected:\n%v\ngot:\n%v", expectedLogTextInLogFile, string(logFileContents))
	}

	go func() {
		for i := 0; i < 10; i++ {
			glog.Infoln("logging message: ", i)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// stop logger and see if os.Stderr and glog continue functioning
	<-nodeLogger.Stop()

	glog.Infoln("logging message: this message happens after custom logger has been stopped")
}
