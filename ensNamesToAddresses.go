package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/wealdtech/go-ens/v3"
)

const (
	infuraKey  = "INFURA KEY HERE"
	chainID    = walletCommon.EthereumMainnet
	rpcBaseUrl = "https://mainnet.infura.io/v3/"
)

func getENSNames() []string {
	return []string{
		"vitalik.eth",
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Begin")

	infuraUrl := fmt.Sprintf("%s%s", rpcBaseUrl, infuraKey)
	client, err := ethclient.Dial(infuraUrl)
	if err != nil {
		log.Fatal(err)
	}

	ensNames := getENSNames()

	for _, domain := range ensNames {
		// log.Println("ENS Name: ", domain)
		address, err := ens.Resolve(client, domain)
		if err != nil {
			fmt.Println("Error: ", err, "name", domain)
			break
		}
		fmt.Println(address.Hex())
		time.Sleep(10 * time.Second)
	}

	log.Println("End")
}
