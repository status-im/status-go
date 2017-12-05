package main

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/status-im/status-go/geth/params"
)

var (
	prodMode    = flag.Bool("production", false, "Whether production settings should be loaded")
	dataDir     = flag.String("datadir", "wnode-data", "Data directory for the databases and keystore")
	networkID   = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby)")
	listenAddr  = flag.String("listenaddr", params.ListenAddr, "IP address and port of this node (e.g. 127.0.0.1:30303)")
	httpEnabled = flag.Bool("http", false, "HTTP RPC enpoint enabled (default: false)")
	httpPort    = flag.Int("httpport", params.HTTPPort, "HTTP RPC server's listening port")
	ipcEnabled  = flag.Bool("ipc", false, "IPC RPC endpoint enabled")
	logLevel    = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
	logFile     = flag.String("logfile", "", "Path to the log file")

	// Whisper
	identity     = flag.String("identity", "", "Protocol identity file (private key used for asymmetric encryption)")
	passwordFile = flag.String("passwordfile", "", "Password file (password is used for symmetric encryption)")
	minPow       = flag.Float64("pow", params.WhisperMinimumPoW, "PoW for messages to be added to queue, in float format")
	ttl          = flag.Int("ttl", params.WhisperTTL, "Time to live for messages, in seconds")

	// MailServer
	enableMailServer = flag.Bool("mailserver", false, "Delivers expired messages on demand")

	// Push Notification
	enablePN     = flag.Bool("notify", false, "Node is capable of sending Push Notifications")
	firebaseAuth = flag.String("firebaseauth", "", "FCM Authorization Key used for sending Push Notifications")

	// Tesing and debug
	injectAccounts = flag.Bool("injectaccounts", false, "Whether test account should be injected or not")
)

// makeNodeConfig creates node configuration object from flags
func makeNodeConfig() (*params.NodeConfig, error) {
	devMode := !*prodMode
	nodeConfig, err := params.NewNodeConfig(*dataDir, uint64(*networkID), devMode)
	if err != nil {
		return nil, err
	}

	nodeConfig.ListenAddr = *listenAddr

	nodeConfig.LogLevel = *logLevel
	if filepath.IsAbs(*logFile) {
		nodeConfig.LogFile = *logFile
	} else if *logFile != "" {
		nodeConfig.LogFile = filepath.Join(*dataDir, *logFile)
	}

	// disable les and swarm for wnode
	nodeConfig.LightEthConfig.Enabled = false
	nodeConfig.SwarmConfig.Enabled = false

	// whisper configuration
	whisperConfig := nodeConfig.WhisperConfig
	whisperConfig.Enabled = true
	whisperConfig.IdentityFile = *identity
	whisperConfig.PasswordFile = *passwordFile
	whisperConfig.EnablePushNotification = *enablePN
	whisperConfig.EnableMailServer = *enableMailServer
	whisperConfig.MinimumPoW = *minPow
	whisperConfig.TTL = *ttl

	if whisperConfig.EnableMailServer && whisperConfig.Password == "" && whisperConfig.PasswordFile == "" {
		return nil, errors.New("either a password file or a password should be specified for mail server")
	}

	if whisperConfig.PasswordFile == "" {
		//fixme(@jekamas): hard coded for mail server MVP - issue https://github.com/status-im/status-go/issues/488
		whisperConfig.Password = "status-offline-inbox"
	}

	if whisperConfig.EnablePushNotification && whisperConfig.IdentityFile == "" {
		return nil, errors.New("notification server requires -identity file to be specified")
	}

	if whisperConfig.PasswordFile != "" && whisperConfig.Password == "" {
		if _, err := whisperConfig.ReadPasswordFile(); err != nil {
			return nil, fmt.Errorf("read password file: %v", err)
		}
	}

	if whisperConfig.IdentityFile != "" {
		if _, err := whisperConfig.ReadIdentityFile(); err != nil {
			return nil, fmt.Errorf("read identity file: %v", err)
		}
	}

	// firebase configuration
	firebaseConfig := whisperConfig.FirebaseConfig
	firebaseConfig.AuthorizationKeyFile = *firebaseAuth
	if firebaseConfig.AuthorizationKeyFile != "" {
		if _, err := firebaseConfig.ReadAuthorizationKeyFile(); err != nil {
			return nil, err
		}
	}

	// RPC configuration
	// TODO(adam): clarify all these IPC/RPC/HTTPHost
	if !*httpEnabled {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}
	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled
	nodeConfig.RPCEnabled = *httpEnabled

	return nodeConfig, nil
}
