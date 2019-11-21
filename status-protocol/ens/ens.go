package ens

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	ens "github.com/wealdtech/go-ens/v3"
	"go.uber.org/zap"
	"math/big"
	"time"
)

const (
	contractQueryTimeout = 5000 * time.Millisecond
)

type ENSVerifier struct {
	logger *zap.Logger
}

type ENSDetails struct {
	Name            string `json:"name"`
	PublicKeyString string `json:"publicKey"`
}

type ENSResponse struct {
	Name            string           `json:"name"`
	Verified        bool             `json:"verified"`
	VerifiedAt      int64            `json:"verifiedAt"`
	Error           error            `json:"error"`
	PublicKey       *ecdsa.PublicKey `json:"-"`
	PublicKeyString string           `json:"publicKey"`
}

func NewVerifier(logger *zap.Logger) *ENSVerifier {
	return &ENSVerifier{logger: logger}
}

func (m *ENSVerifier) verifyENSName(ensInfo ENSDetails, ethclient *ethclient.Client) ENSResponse {
	publicKeyStr := ensInfo.PublicKeyString
	ensName := ensInfo.Name
	m.logger.Info("Resolving ENS name", zap.String("name", ensName), zap.String("publicKey", publicKeyStr))
	response := ENSResponse{
		Name:            ensName,
		PublicKeyString: publicKeyStr,
		VerifiedAt:      time.Now().Unix(),
	}

	expectedPubKeyBytes, err := hex.DecodeString(publicKeyStr)
	if err != nil {
		response.Error = err
		return response
	}

	publicKey, err := crypto.UnmarshalPubkey(expectedPubKeyBytes)
	if err != nil {
		response.Error = err
		return response
	}

	// Resolve ensName
	resolver, err := ens.NewResolver(ethclient, ensName)
	if err != nil {
		m.logger.Error("error while creating ENS name resolver", zap.String("ensName", ensName), zap.Error(err))
		response.Error = err
		return response
	}
	x, y, err := resolver.PubKey()
	if err != nil {
		m.logger.Error("error while resolving public key from ENS name", zap.String("ensName", ensName), zap.Error(err))
		response.Error = err
		return response
	}

	// Assemble the bytes returned for the pubkey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), new(big.Int).SetBytes(x[:]), new(big.Int).SetBytes(y[:]))

	response.PublicKey = publicKey
	response.Verified = bytes.Equal(pubKeyBytes, expectedPubKeyBytes)
	return response
}

// CheckBatch verifies that a registered ENS name matches the expected public key
func (m *ENSVerifier) CheckBatch(ensDetails []ENSDetails, rpcEndpoint, contractAddress string) (map[string]ENSResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), contractQueryTimeout)
	defer cancel()

	ch := make(chan ENSResponse)
	response := make(map[string]ENSResponse)

	ethclient, err := ethclient.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return nil, err
	}

	for _, ensInfo := range ensDetails {
		go func(info ENSDetails) { ch <- m.verifyENSName(info, ethclient) }(ensInfo)
	}

	for range ensDetails {
		r := <-ch
		response[r.PublicKeyString] = r
	}
	close(ch)

	return response, nil
}
