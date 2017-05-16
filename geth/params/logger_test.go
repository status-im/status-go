package params_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/params"
)

func TestLogger(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-logger-tests")
	if err != nil {
		t.Fatal(err)
	}
	//defer os.RemoveAll(tmpDir)

	nodeConfig, err := params.NewNodeConfig(tmpDir, params.RopstenNetworkID, true)
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
	nodeConfig.LogToStderr = false // just capture logs to file
	nodeLogger, err = params.SetupLogger(nodeConfig)
	if err != nil {
		t.Fatal("cannot create logger object")
	}
	if nodeLogger == nil {
		t.Fatal("logger is empty (while logs are enabled)")
	}

	validateLogText := func(expectedLogText string) {
		logFilePath := filepath.Join(nodeConfig.DataDir, nodeConfig.LogFile)
		logBytes, err := ioutil.ReadFile(logFilePath)
		if err != nil {
			panic(err)
		}
		logText := string(logBytes)
		logText = strings.Trim(logText, "\n")
		logText = logText[len(logText)-len(expectedLogText):] // as logs can be prepended with log info

		if expectedLogText != logText {
			t.Fatalf("invalid log, expected: [%s], got: [%s]", expectedLogText, string(logText))
		} else {
			t.Logf("log match found, expected: [%s], got: [%s]", expectedLogText, string(logText))
		}
	}

	// sample log message
	log.Info("use log package")
	validateLogText(`msg="use log package"`)

	// log using DEBUG log level (with appropriate level set)
	nodeLogger.SetV("DEBUG")
	log.Info("logged DEBUG log level message")
	validateLogText(`msg="logged DEBUG log level message"`)

	// log using DEBUG log level (with appropriate level set)
	nodeLogger.SetV("INFO")
	log.Info("logged INFO log level message")
	validateLogText(`msg="logged INFO log level message"`)
	log.Debug("logged DEBUG log level message")
	validateLogText(`msg="logged INFO log level message"`) // debug level message is NOT logged

	// stop logger and see if os.Stderr and gethlog continue functioning
	if err = nodeLogger.Stop(); err != nil {
		t.Fatal(err)
	}

	log.Info("logging message: this message happens after custom logger has been stopped")
}
