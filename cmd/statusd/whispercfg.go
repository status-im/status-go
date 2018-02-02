package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/status-im/status-go/geth/params"
)

// whisperConfig creates node configuration object from flags
func whisperConfig(nodeConfig *params.NodeConfig) (*params.NodeConfig, error) {

	nodeConfig.ListenAddr = *listenAddr

	nodeConfig.LogLevel = *logLevel

	// whisper configuration
	whisperConfig := nodeConfig.WhisperConfig
	whisperConfig.Enabled = true
	whisperConfig.IdentityFile = *identityFile
	whisperConfig.EnablePushNotification = *enablePN
	whisperConfig.EnableMailServer = *enableMailServer
	whisperConfig.MinimumPoW = *minPow
	whisperConfig.TTL = *ttl

	if whisperConfig.EnablePushNotification && whisperConfig.IdentityFile == "" {
		return nil, errors.New("notification server requires -identity file to be specified")
	}

	if whisperConfig.IdentityFile != "" {
		if _, err := whisperConfig.ReadIdentityFile(); err != nil {
			return nil, fmt.Errorf("read identity file: %v", err)
		}
	}

	if whisperConfig.EnableMailServer {
		if *passwordFile == "" {
			return nil, errors.New("passwordfile should be specified if MailServer is enabled")
		}

		password, err := readFile(*passwordFile)
		if err != nil {
			return nil, fmt.Errorf("password file: %v", err)
		}

		whisperConfig.Password = string(password)
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
	nodeConfig.HTTPHost = "0.0.0.0"

	return nodeConfig, nil
}

func readFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimRight(data, "\n")

	if len(data) == 0 {
		return nil, errors.New("file is empty")
	}

	return data, nil
}
