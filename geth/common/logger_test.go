package common_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/suite"
)

type LoggerTestSuite struct {
	suite.Suite
}

func TestLogger(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}

func (s *LoggerTestSuite) TestLocalLogger() {
	require := s.Require()
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-logger-tests")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	nodeConfig, err := params.NewNodeConfig(tmpDir, params.RopstenNetworkID, true)
	require.NoError(err, "cannot create config object")

	loggerConfig := nodeConfig.LoggerConfig
	nodeLogger, err := common.NewLogger(loggerConfig)
	require.EqualError(err, common.ErrLoggerDisabled.Error())
	require.Nil(nodeLogger, "logger is not empty (while logs are disabled)")

	loggerConfig.Enabled = true
	loggerConfig.LogToStderr = false // just capture logs to file
	loggerConfig.LogToFile = true
	nodeLogger, err = common.NewLogger(loggerConfig)
	require.NoError(err, "cannot create logger object")
	require.NotNil(nodeLogger, "logger is empty (while logs are enabled)")

	require.NoError(nodeLogger.Attach()) // start capturing logs using our logger

	validateLogText := func(expectedLogText string) {
		logBytes, err := ioutil.ReadFile(loggerConfig.LogFile)
		require.NoError(err)

		logText := string(logBytes)
		logText = strings.Trim(logText, "\n")
		logText = logText[len(logText)-len(expectedLogText):] // as logs can be prepended with log info

		require.Equal(expectedLogText, logText)
		s.T().Logf("log match found, expected: [%s], got: [%s]", expectedLogText, string(logText))
	}

	// sample log message
	log.Info("use log package")
	validateLogText(`msg="use log package"`)

	// log using DEBUG log level (with appropriate level set)
	nodeLogger.SetV(log.LvlDebug)
	log.Info("logged DEBUG log level message")
	validateLogText(`msg="logged DEBUG log level message"`)

	// log using DEBUG log level (with appropriate level set)
	nodeLogger.SetV(log.LvlInfo)
	log.Info("logged INFO log level message")
	validateLogText(`msg="logged INFO log level message"`)
	log.Debug("logged DEBUG log level message")
	validateLogText(`msg="logged INFO log level message"`) // debug level message is NOT logged

	// stop logger and see if os.Stderr and gethlog continue functioning
	require.NoError(nodeLogger.Detach())

	log.Info("logging message: this message happens after custom logger has been stopped")
}

func (s *LoggerTestSuite) TestRemoteLogger() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Dragons!!", "recovered", r)
			time.Sleep(5 * time.Second)
		}
	}()

	require := s.Require()

	loggerConfig := &params.LoggerConfig{
		Enabled:             true,
		RemoteHostName:      "LoggerTest",
		Level:               "TRACE",
		RemoteAPIKey:        params.LoggerRemoteAPIKey,
		RemoteFlushInterval: 1,
		RemoteBufferSize:    100,
		LogToRemote:         true,
		LogToStderr:         false,
		LogToFile:           false,
	}

	nodeLogger, err := common.NewLogger(loggerConfig)
	require.NoError(err)
	require.NotNil(nodeLogger)
	require.NoError(nodeLogger.Attach())

	log.Trace("tracing the local object", "logger", nodeLogger)
	log.Debug("debug message should provide some insite", "config", loggerConfig)
	log.Info("test log")
	log.Warn("I warn you it, the last time..")
	log.Error("something strange here")

	var nilLogger common.Logger
	nilLogger.Attach() // dragons!!!

	require.NoError(nodeLogger.Detach())
}
