// Package ccipread implements an ERC-3668 (CCIP-Read) aware
// bind.ContractBackend wrapper.
//
// The wrapper transparently handles the OffchainLookup revert pattern:
// when an eth_call reverts with OffchainLookup(sender, urls, callData,
// callbackFunction, extraData), the wrapper fetches the gateway response
// over HTTPS and re-invokes the contract's callback function so that the
// caller sees a normal successful eth_call result.
package ccipread

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultMaxHops        = 4
	defaultGatewayTimeout = 5 * time.Second
	defaultHTTPTimeout    = 10 * time.Second
)

// OffchainLookup(address,string[],bytes,bytes4,bytes) selector per ERC-3668.
var offchainLookupSelector = [4]byte{0x55, 0x6f, 0x18, 0x30}

var (
	errMaxHopsExceeded   = errors.New("ccipread: max gateway hops exceeded")
	errSenderMismatch    = errors.New("ccipread: OffchainLookup sender does not match contract")
	errInsecureGateway   = errors.New("ccipread: gateway URL must use https://")
	errAllGatewaysFailed = errors.New("ccipread: all gateways failed")
)

var (
	offchainLookupArgs abi.Arguments
	callbackArgs       abi.Arguments
)

func init() {
	addrTy, _ := abi.NewType("address", "", nil)
	stringArrTy, _ := abi.NewType("string[]", "", nil)
	bytesTy, _ := abi.NewType("bytes", "", nil)
	bytes4Ty, _ := abi.NewType("bytes4", "", nil)

	offchainLookupArgs = abi.Arguments{
		{Type: addrTy},
		{Type: stringArrTy},
		{Type: bytesTy},
		{Type: bytes4Ty},
		{Type: bytesTy},
	}
	callbackArgs = abi.Arguments{
		{Type: bytesTy},
		{Type: bytesTy},
	}
}

// Caller wraps a bind.ContractBackend with ERC-3668 support. CallContract
// intercepts OffchainLookup reverts; all other methods delegate to the inner
// backend unchanged via the embedded interface. Test-only knobs (maxHops,
// allowInsecure) are set by tests directly on the returned struct.
type Caller struct {
	bind.ContractBackend
	http           *http.Client
	maxHops        int
	gatewayTimeout time.Duration
	allowInsecure  bool
}

// New wraps the given backend with CCIP-Read support using production defaults.
func New(inner bind.ContractBackend) *Caller {
	return &Caller{
		ContractBackend: inner,
		http:            &http.Client{Timeout: defaultHTTPTimeout},
		maxHops:         defaultMaxHops,
		gatewayTimeout:  defaultGatewayTimeout,
	}
}

// CallContract overrides the embedded backend's method to add CCIP-Read.
func (c *Caller) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.callWithCCIP(ctx, msg, blockNumber, c.maxHops)
}

func (c *Caller) callWithCCIP(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int, hopsLeft int) ([]byte, error) {
	result, err := c.ContractBackend.CallContract(ctx, msg, blockNumber)
	if err == nil {
		return result, nil
	}
	revertData, ok := extractRevertData(err)
	if !ok || !bytes.HasPrefix(revertData, offchainLookupSelector[:]) {
		return nil, err
	}
	if hopsLeft <= 0 {
		return nil, errMaxHopsExceeded
	}
	sender, urls, callData, callbackFn, extraData, decodeErr := decodeOffchainLookup(revertData)
	if decodeErr != nil {
		return nil, fmt.Errorf("ccipread: decode OffchainLookup: %w", decodeErr)
	}
	if msg.To == nil || sender != *msg.To {
		return nil, errSenderMismatch
	}
	response, gatewayErr := c.fetchFromGateways(ctx, urls, sender, callData)
	if gatewayErr != nil {
		return nil, gatewayErr
	}
	callbackCalldata, encErr := buildCallbackCalldata(callbackFn, response, extraData)
	if encErr != nil {
		return nil, fmt.Errorf("ccipread: build callback calldata: %w", encErr)
	}
	newMsg := msg
	newMsg.Data = callbackCalldata
	return c.callWithCCIP(ctx, newMsg, blockNumber, hopsLeft-1)
}

func decodeOffchainLookup(data []byte) (sender common.Address, urls []string, callData []byte, callbackFn [4]byte, extraData []byte, err error) {
	values, err := offchainLookupArgs.Unpack(data[4:])
	if err != nil {
		return
	}
	if len(values) != 5 {
		err = fmt.Errorf("expected 5 fields, got %d", len(values))
		return
	}
	var ok bool
	if sender, ok = values[0].(common.Address); !ok {
		err = errors.New("sender not address")
		return
	}
	if urls, ok = values[1].([]string); !ok {
		err = errors.New("urls not []string")
		return
	}
	if callData, ok = values[2].([]byte); !ok {
		err = errors.New("callData not []byte")
		return
	}
	if callbackFn, ok = values[3].([4]byte); !ok {
		err = errors.New("callbackFunction not bytes4")
		return
	}
	if extraData, ok = values[4].([]byte); !ok {
		err = errors.New("extraData not []byte")
		return
	}
	return
}

func buildCallbackCalldata(callbackFn [4]byte, response, extraData []byte) ([]byte, error) {
	encoded, err := callbackArgs.Pack(response, extraData)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, 4+len(encoded))
	out = append(out, callbackFn[:]...)
	out = append(out, encoded...)
	return out, nil
}

func (c *Caller) fetchFromGateways(ctx context.Context, urls []string, sender common.Address, callData []byte) ([]byte, error) {
	if len(urls) == 0 {
		return nil, errAllGatewaysFailed
	}
	var lastErr error
	for _, u := range urls {
		resp, err := c.fetchOne(ctx, u, sender, callData)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("%w: %v", errAllGatewaysFailed, lastErr)
}

func (c *Caller) fetchOne(ctx context.Context, gatewayURL string, sender common.Address, callData []byte) ([]byte, error) {
	if !c.allowInsecure && !strings.HasPrefix(strings.ToLower(gatewayURL), "https://") {
		return nil, errInsecureGateway
	}
	senderHex := strings.ToLower(sender.Hex())
	dataHex := "0x" + hex.EncodeToString(callData)

	var req *http.Request
	var err error
	if strings.Contains(gatewayURL, "{data}") {
		expanded := strings.ReplaceAll(gatewayURL, "{sender}", senderHex)
		expanded = strings.ReplaceAll(expanded, "{data}", dataHex)
		if _, parseErr := url.Parse(expanded); parseErr != nil {
			return nil, parseErr
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, expanded, nil)
		if err != nil {
			return nil, err
		}
	} else {
		body, mErr := json.Marshal(map[string]string{
			"sender": senderHex,
			"data":   dataHex,
		})
		if mErr != nil {
			return nil, mErr
		}
		expanded := strings.ReplaceAll(gatewayURL, "{sender}", senderHex)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, expanded, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	callCtx, cancel := context.WithTimeout(req.Context(), c.gatewayTimeout)
	defer cancel()
	req = req.WithContext(callCtx)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ccipread: gateway returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Data string `json:"data"`
	}
	if jErr := json.Unmarshal(b, &decoded); jErr != nil {
		return nil, fmt.Errorf("ccipread: parse gateway response: %w", jErr)
	}
	raw := strings.TrimPrefix(decoded.Data, "0x")
	out, hexErr := hex.DecodeString(raw)
	if hexErr != nil {
		return nil, fmt.Errorf("ccipread: invalid hex in gateway response: %w", hexErr)
	}
	return out, nil
}

// rpcDataError is the structural shape of go-ethereum's rpc.DataError. We
// declare it here to avoid importing rpc just for the interface.
type rpcDataError interface {
	ErrorData() interface{}
}

func extractRevertData(err error) ([]byte, bool) {
	var de rpcDataError
	if !errors.As(err, &de) {
		return nil, false
	}
	raw := de.ErrorData()
	s, ok := raw.(string)
	if !ok {
		return nil, false
	}
	s = strings.TrimPrefix(s, "0x")
	b, hErr := hex.DecodeString(s)
	if hErr != nil {
		return nil, false
	}
	return b, true
}
