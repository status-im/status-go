package ens

import (
	"bytes"
	"context"
	"crypto/elliptic"
	"database/sql"
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
	"github.com/wealdtech/go-ens/v3"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/bgtrace"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/timesource"
)

const contractQueryTimeout = 5000 * time.Millisecond

type Verifier struct {
	online          bool
	persistence     *Persistence
	logger          *zap.Logger
	timesource      timesource.Provider
	subscriptions   []chan []*VerificationRecord
	rpcEndpoint     string
	contractAddress string
	quit            chan struct{}
	quitWg          sync.WaitGroup
}

func New(logger *zap.Logger, timesource timesource.Provider, db *sql.DB, rpcEndpoint, contractAddress string) *Verifier {
	persistence := NewPersistence(db)
	return &Verifier{
		logger:          logger,
		persistence:     persistence,
		timesource:      timesource,
		rpcEndpoint:     rpcEndpoint,
		contractAddress: contractAddress,
		quit:            make(chan struct{}),
	}
}

func (v *Verifier) Start() error {
	v.quitWg.Add(1)
	go v.verifyLoop()
	return nil
}

func (v *Verifier) Stop() error {
	close(v.quit)
	v.quitWg.Wait()
	return nil
}

// ENSVerified adds an already verified entry to the ens table
func (v *Verifier) ENSVerified(pk, ensName string, clock uint64) error {

	// Add returns nil if no record was available
	oldRecord, err := v.Add(pk, ensName, clock)
	if err != nil {
		return err
	}

	var record *VerificationRecord

	if oldRecord != nil {
		record = oldRecord
	} else {
		record = &VerificationRecord{PublicKey: pk, Name: ensName, Clock: clock}
	}

	record.VerifiedAt = clock
	record.Verified = true
	records := []*VerificationRecord{record}
	err = v.persistence.UpdateRecords(records)
	if err != nil {
		return err
	}
	v.publish(records)
	return nil
}

func (v *Verifier) GetVerifiedRecord(pk string) (*VerificationRecord, error) {
	return v.persistence.GetVerifiedRecord(pk)
}

func (v *Verifier) Add(pk, ensName string, clock uint64) (*VerificationRecord, error) {
	record := VerificationRecord{PublicKey: pk, Name: ensName, Clock: clock}
	return v.persistence.AddRecord(record)
}

func (v *Verifier) SetOnline(online bool) {
	v.online = online
}

func (v *Verifier) verifyLoop() {
	defer gocommon.LogOnPanic()
	defer v.quitWg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-v.quit:
			return
		case <-ticker.C:
			bgtrace.Tick("ens.verifyLoop", false, 30*time.Second)
			if !v.online || v.rpcEndpoint == "" || v.contractAddress == "" {
				continue
			}
			err := v.verify(context.Background(), v.rpcEndpoint, v.contractAddress)
			if err != nil {
				v.logger.Error("verify loop failed", zap.Error(err))
			}

		}
	}
}

func (v *Verifier) Subscribe() chan []*VerificationRecord {
	c := make(chan []*VerificationRecord)
	v.subscriptions = append(v.subscriptions, c)
	return c
}

func (v *Verifier) publish(records []*VerificationRecord) {
	v.logger.Info("publishing records", zap.Any("records", records))
	// Publish on channels, drop if buffer is full
	for _, s := range v.subscriptions {
		select {
		case s <- records:
		default:
			v.logger.Warn("ens subscription channel full, dropping message")
		}
	}
}

func (v *Verifier) ReverseResolve(ctx context.Context, address gethcommon.Address) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, contractQueryTimeout)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, v.rpcEndpoint)
	if err != nil {
		return "", err
	}

	ethClient := ethclient.NewClient(rpcClient)
	return ens.ReverseResolve(ethClient, address)
}

// Verify verifies that a registered ENS name matches the expected public key
func (v *Verifier) verify(ctx context.Context, rpcEndpoint, contractAddress string) error {
	v.logger.Debug("verifying ENS Names")

	var ensDetails []Details

	// Now in seconds
	now := v.timesource.Now().Unix()
	ensToBeVerified, err := v.persistence.GetENSToBeVerified(uint64(now))
	if err != nil {
		return err
	}

	recordsMap := make(map[string]*VerificationRecord)

	for _, r := range ensToBeVerified {
		recordsMap[r.PublicKey] = r
		ensDetails = append(ensDetails, Details{
			PublicKeyString: r.PublicKey[2:],
			Name:            r.Name,
		})
		v.logger.Debug("verifying ens name", zap.Any("record", r))
	}

	ctx, cancel := context.WithTimeout(ctx, contractQueryTimeout)
	defer cancel()

	ch := make(chan Response)
	ensResponse := make(map[string]Response)

	rpcClient, err := rpc.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc endpoint")
	}

	ethClient := ethclient.NewClient(rpcClient)

	for _, ensInfo := range ensDetails {
		v.quitWg.Add(1)
		go func(info Details) {
			defer gocommon.LogOnPanic()
			defer v.quitWg.Done()
			select {
			case ch <- v.verifyENSName(info, ethClient):
				return
			case <-ctx.Done():
				return
			case <-v.quit:
				return
			}
		}(ensInfo)
	}

	for range ensDetails {
		r := <-ch
		ensResponse[r.PublicKeyString] = r
	}
	close(ch)

	var records []*VerificationRecord

	for _, details := range ensResponse {
		pk := "0x" + details.PublicKeyString
		record := recordsMap[pk]

		if details.Error == nil {
			record.Verified = details.Verified
			if !record.Verified {
				record.VerificationRetries++
			}
		} else {
			v.logger.Warn("failed to resolve ens name",
				zap.String("name", details.Name),
				zap.String("publicKey", details.PublicKeyString),
				zap.Error(details.Error),
			)
			record.VerificationRetries++
		}
		record.VerifiedAt = uint64(now)
		record.CalculateNextRetry()

		records = append(records, record)
	}

	err = v.persistence.UpdateRecords(records)
	if err != nil {

		v.logger.Error("failed to update records", zap.Error(err))
		return err
	}

	v.publish(records)

	return nil
}

func (v *Verifier) verifyENSName(ensInfo Details, ethclient *ethclient.Client) Response {
	publicKeyStr := ensInfo.PublicKeyString
	ensName := ensInfo.Name
	v.logger.Info("Resolving ENS name", zap.String("name", ensName), zap.String("publicKey", publicKeyStr))
	response := Response{
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
		v.logger.Error("error while creating ENS name resolver", zap.Error(err))
		response.Error = err
		return response
	}
	x, y, err := resolver.PubKey()
	if err != nil {
		v.logger.Error("error while resolving public key from ENS name", zap.Error(err))
		response.Error = err
		return response
	}

	// Assemble the bytes returned for the pubkey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), new(big.Int).SetBytes(x[:]), new(big.Int).SetBytes(y[:]))

	response.PublicKey = publicKey
	response.Verified = bytes.Equal(pubKeyBytes, expectedPubKeyBytes)
	return response
}
