package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/status-im/status-go/geth/params"
)

// makeNodeConfig creates node configuration object from flags
func makeNodeConfig() (*params.NodeConfig, error) {
	devMode := !*prodMode
	nodeConfig, err := params.NewNodeConfig(*dataDir, uint64(*networkID), devMode)
	if err != nil {
		return nil, err
	}

	// TODO(divan): move this logic into params package?
	if *nodeKeyFile != "" {
		nodeConfig.NodeKeyFile = *nodeKeyFile
	}

	// disable log
	nodeConfig.LogLevel = "CRIT"
	nodeConfig.LogFile = ""

	// disable les and swarm for wnode
	nodeConfig.LightEthConfig.Enabled = false
	nodeConfig.SwarmConfig.Enabled = false

	nodeConfig.RPCEnabled = *httpEnabled

	// whisper configuration
	whisperConfig := nodeConfig.WhisperConfig

	whisperConfig.Enabled = true
	whisperConfig.IdentityFile = *identity
	whisperConfig.PasswordFile = *password
	whisperConfig.EchoMode = *echo
	whisperConfig.BootstrapNode = *bootstrap
	whisperConfig.ForwarderNode = *forward
	whisperConfig.NotificationServerNode = *notify
	whisperConfig.MailServerNode = *mailserver
	whisperConfig.Port = *port
	whisperConfig.TTL = *ttl
	whisperConfig.MinimumPoW = *pow

	if whisperConfig.MailServerNode && whisperConfig.PasswordFile == "" {
		return nil, errors.New("mail server requires -password to be specified")
	}

	if whisperConfig.NotificationServerNode && whisperConfig.IdentityFile == "" {
		return nil, errors.New("notification server requires either -identity file to be specified")
	}

	if whisperConfig.PasswordFile != "" {
		if err := verifyPasswordFile(whisperConfig); err != nil {
			return nil, fmt.Errorf("read password file: %v", err)
		}
	}

	if whisperConfig.IdentityFile != "" {
		if err := verifyIdentityFile(whisperConfig); err != nil {
			return nil, fmt.Errorf("read identity file: %v", err)
		}
	}

	// firebase configuration
	firebaseConfig := whisperConfig.FirebaseConfig
	firebaseConfig.AuthorizationKeyFile = *firebaseAuth
	if len(firebaseConfig.AuthorizationKeyFile) > 0 { // make sure authorization key can be loaded
		if firebaseConfig.AuthorizationKeyFile, err = filepath.Abs(firebaseConfig.AuthorizationKeyFile); err != nil {
			return nil, err
		}
		if _, err := firebaseConfig.ReadAuthorizationKeyFile(); err != nil {
			return nil, err
		}
	}

	// RPC configuration
	if !*httpEnabled {
		nodeConfig.HTTPHost = "" // HTTP RPC is disabled
	}
	nodeConfig.HTTPPort = *httpPort
	nodeConfig.IPCEnabled = *ipcEnabled

	return nodeConfig, nil
}

// verifyPasswordFile verifies that we can load password file
func verifyPasswordFile(config *params.WhisperConfig) error {
	// TODO(divan): why do we need it here?
	absPath, err := filepath.Abs(config.PasswordFile)
	if err != nil {
		return err
	}
	config.PasswordFile = absPath
	if _, err = config.ReadPasswordFile(); err != nil {
		return err
	}
	return nil
}

// verifyIdentityFile verifies that we can load identity file
func verifyIdentityFile(config *params.WhisperConfig) error {
	// TODO(divan): why do we need it here?
	absPath, err := filepath.Abs(config.IdentityFile)
	if err != nil {
		return err
	}
	config.IdentityFile = absPath
	if _, err = config.ReadIdentityFile(); err != nil {
		return err
	}
	return nil
}
