package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/eth-node/crypto"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

const (
	contractAddress = "0xD1FaDDF951F0177dBCF457dFC88Ea3Ee14bE08C0" // with 0x
	infuraKey       = "KEY"
	privateKeyHex   = "PRIVATE KEY WITHOUT 0x" // without 0x
	chainID         = walletCommon.OptimismMainnet
	rpcBaseUrl      = "https://optimism-mainnet.infura.io/v3/"
)

func getWalletAddresses() []common.Address {
	return []common.Address{
		// Addresses to airrop to go here
		common.HexToAddress("0x1"),
		common.HexToAddress("0x2"),
		common.HexToAddress("0x3"),
		common.HexToAddress("0x4"),
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

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddressCrypto := crypto.PubkeyToAddress(*publicKeyECDSA)
	fromAddress := common.HexToAddress(fromAddressCrypto.Hex())

	collectiblesContract, err := collectibles.NewCollectibles(common.HexToAddress(contractAddress), client)
	if err != nil {
		log.Fatal(err)
	}

	signerFn := func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, privateKey)
	}

	walletAddresses := getWalletAddresses()

	limit := 180

	var addressesToSend []common.Address

	for i, address := range walletAddresses {
		addressesToSend = append(addressesToSend, address)
		// Append addresses until we reach the limit
		// Make sure it's not the final address though to not skip it
		if len(addressesToSend) < limit && i < len(walletAddresses)-1 {
			continue
		}

		// Time to send
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Fatal(err)
		}

		transactOpts := &bind.TransactOpts{
			From:   fromAddress,
			Nonce:  big.NewInt(int64(nonce)),
			Signer: signerFn,
			Value:  big.NewInt(0),
		}

		log.Print("Minting collectibles to ", addressesToSend)
		tx, err := collectiblesContract.MintTo(transactOpts, addressesToSend)
		if err != nil {
			log.Fatal("Failed", "error", err)
			break
		}
		log.Print("Success! ", "tx ", tx.Hash().Hex())

		log.Print("Sleeping to avoid Infura limit")
		time.Sleep(5 * time.Second) // Sleep to avoid Infura limit

		// Empty slice to start the next batch
		addressesToSend = nil
	}

	log.Println("End")
}
