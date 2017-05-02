// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/log"
)

// deployFaucet queries the user for various input on deploying a faucet, after
// which it executes it.
func (w *wizard) deployFaucet() {
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active faucet configurations from the server
	infos, err := checkFaucet(client, w.network)
	if err != nil {
		infos = &faucetInfos{
			node:    &nodeInfos{portFull: 30303, peersTotal: 25},
			port:    80,
			host:    client.server,
			amount:  1,
			minutes: 1440,
		}
	}
	infos.node.genesis, _ = json.MarshalIndent(w.conf.genesis, "", "  ")
	infos.node.network = w.conf.genesis.Config.ChainId.Int64()

	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which port should the faucet listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure which virtual-host to deploy ethstats on
	if infos.host, err = w.ensureVirtualHost(client, infos.port, infos.host); err != nil {
		log.Error("Failed to decide on faucet host", "err", err)
		return
	}
	// Port and proxy settings retrieved, figure out the funcing amount per perdion configurations
	fmt.Println()
	fmt.Printf("How many Ethers to release per request? (default = %d)\n", infos.amount)
	infos.amount = w.readDefaultInt(infos.amount)

	fmt.Println()
	fmt.Printf("How many minutes to enforce between requests? (default = %d)\n", infos.minutes)
	infos.minutes = w.readDefaultInt(infos.minutes)

	// Accessing GitHub gists requires API authorization, retrieve it
	if infos.githubUser != "" {
		fmt.Println()
		fmt.Printf("Reused previous (%s) GitHub API authorization (y/n)? (default = yes)\n", infos.githubUser)
		if w.readDefaultString("y") != "y" {
			infos.githubUser, infos.githubToken = "", ""
		}
	}
	if infos.githubUser == "" {
		// No previous authorization (or new one requested)
		fmt.Println()
		fmt.Println("Which GitHub user to verify Gists through?")
		infos.githubUser = w.readString()

		fmt.Println()
		fmt.Println("What is the GitHub personal access token of the user? (won't be echoed)")
		infos.githubToken = w.readPassword()

		// Do a sanity check query against github to ensure it's valid
		req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
		req.SetBasicAuth(infos.githubUser, infos.githubToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error("Failed to verify GitHub authentication", "err", err)
			return
		}
		defer res.Body.Close()

		var msg struct {
			Login   string `json:"login"`
			Message string `json:"message"`
		}
		if err = json.NewDecoder(res.Body).Decode(&msg); err != nil {
			log.Error("Failed to decode authorization response", "err", err)
			return
		}
		if msg.Login != infos.githubUser {
			log.Error("GitHub authorization failed", "user", infos.githubUser, "message", msg.Message)
			return
		}
	}
	// Figure out where the user wants to store the persistent data
	fmt.Println()
	if infos.node.datadir == "" {
		fmt.Printf("Where should data be stored on the remote machine?\n")
		infos.node.datadir = w.readString()
	} else {
		fmt.Printf("Where should data be stored on the remote machine? (default = %s)\n", infos.node.datadir)
		infos.node.datadir = w.readDefaultString(infos.node.datadir)
	}
	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which TCP/UDP port should the light client listen on? (default = %d)\n", infos.node.portFull)
	infos.node.portFull = w.readDefaultInt(infos.node.portFull)

	// Set a proper name to report on the stats page
	fmt.Println()
	if infos.node.ethstats == "" {
		fmt.Printf("What should the node be called on the stats page?\n")
		infos.node.ethstats = w.readString() + ":" + w.conf.ethstats
	} else {
		fmt.Printf("What should the node be called on the stats page? (default = %s)\n", infos.node.ethstats)
		infos.node.ethstats = w.readDefaultString(infos.node.ethstats) + ":" + w.conf.ethstats
	}
	// Load up the credential needed to release funds
	if infos.node.keyJSON != "" {
		var key keystore.Key
		if err := json.Unmarshal([]byte(infos.node.keyJSON), &key); err != nil {
			infos.node.keyJSON, infos.node.keyPass = "", ""
		} else {
			fmt.Println()
			fmt.Printf("Reuse previous (%s) funding account (y/n)? (default = yes)\n", key.Address.Hex())
			if w.readDefaultString("y") != "y" {
				infos.node.keyJSON, infos.node.keyPass = "", ""
			}
		}
	}
	if infos.node.keyJSON == "" {
		fmt.Println()
		fmt.Println("Please paste the faucet's funding account key JSON:")
		infos.node.keyJSON = w.readJSON()

		fmt.Println()
		fmt.Println("What's the unlock password for the account? (won't be echoed)")
		infos.node.keyPass = w.readPassword()

		if _, err := keystore.DecryptKey([]byte(infos.node.keyJSON), infos.node.keyPass); err != nil {
			log.Error("Failed to decrypt key with given passphrase")
			return
		}
	}
	// Try to deploy the faucet server on the host
	if out, err := deployFaucet(client, w.network, w.conf.bootLight, infos); err != nil {
		log.Error("Failed to deploy faucet container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	w.networkStats(false)
}
