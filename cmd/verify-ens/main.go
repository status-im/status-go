package main

import (
	"fmt"

	"github.com/status-im/status-go/eth-node/bridge/geth/ens"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"go.uber.org/zap"
)

const infuraToken = "putYourTokenHere"
const verifyENSContractAddress = "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"

var verifyENSURL = fmt.Sprintf("https://mainnet.infura.io/v3/%s", infuraToken)

func main() {
	logger := zap.NewExample()
	defer logger.Sync()

	verifier := ens.NewVerifier(logger)

	request := enstypes.ENSDetails{
		PublicKeyString: "0408195cccdc7124ff5ca0cd3f406d8057fe5e6de3acc34da00e783d06f057f640e9b5af96c47ae64aa9209b4764f3849a9ec27818c3e88c6c1f3d2d2b1cd5dac4",
		Name:            "magnus.stateofus.eth",
	}

	response, err := verifier.CheckBatch([]enstypes.ENSDetails{request}, verifyENSURL, verifyENSContractAddress)
	if err != nil {
		logger.Error("failed to verify batch", zap.Error(err))
		return
	}

	if response[request.PublicKeyString].Error != nil {
		logger.Error("verify error", zap.Error(response[request.PublicKeyString].Error))
		return
	}

	logger.Info("verified", zap.Bool("value", response[request.PublicKeyString].Verified))
}
