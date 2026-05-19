package ccipread

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// --- test helpers -----------------------------------------------------------

// stubBackend lets a test script a sequence of CallContract responses. The
// embedded interface is nil; any non-CallContract method would panic, but
// the tests only exercise CallContract via the wrapper.
type stubBackend struct {
	bind.ContractBackend
	responses []callResp
	calls     []ethereum.CallMsg
}

type callResp struct {
	data []byte
	err  error
}

func (s *stubBackend) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	s.calls = append(s.calls, msg)
	if len(s.calls)-1 >= len(s.responses) {
		return nil, errors.New("stubBackend: no more responses scripted")
	}
	r := s.responses[len(s.calls)-1]
	return r.data, r.err
}

// ofcError mimics go-ethereum's rpc.DataError surface so the wrapper's
// extractRevertData can pull the revert bytes off it.
type ofcError struct {
	hexData string
}

func (e *ofcError) Error() string          { return "execution reverted" }
func (e *ofcError) ErrorData() interface{} { return e.hexData }

func encodeOffchainLookupRevert(t *testing.T, sender common.Address, urls []string, callData []byte, cb [4]byte, extraData []byte) string {
	t.Helper()
	addrTy, _ := abi.NewType("address", "", nil)
	stringArrTy, _ := abi.NewType("string[]", "", nil)
	bytesTy, _ := abi.NewType("bytes", "", nil)
	bytes4Ty, _ := abi.NewType("bytes4", "", nil)
	args := abi.Arguments{
		{Type: addrTy}, {Type: stringArrTy}, {Type: bytesTy}, {Type: bytes4Ty}, {Type: bytesTy},
	}
	body, err := args.Pack(sender, urls, callData, cb, extraData)
	if err != nil {
		t.Fatalf("pack OffchainLookup: %v", err)
	}
	full := append([]byte{0x55, 0x6f, 0x18, 0x30}, body...)
	return "0x" + hex.EncodeToString(full)
}

// --- tests ------------------------------------------------------------------

func TestCaller_PassThrough(t *testing.T) {
	stub := &stubBackend{responses: []callResp{
		{data: []byte{0xde, 0xad, 0xbe, 0xef}, err: nil},
	}}
	c := New(stub)
	c.allowInsecure = true

	to := common.HexToAddress("0x1111111111111111111111111111111111111111")
	got, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to}, nil)
	if err != nil {
		t.Fatalf("CallContract: %v", err)
	}
	if hex.EncodeToString(got) != "deadbeef" {
		t.Fatalf("unexpected result: %x", got)
	}
}

func TestCaller_OneHop(t *testing.T) {
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	gatewayResponse := []byte{0xca, 0xfe}
	finalResult := []byte{0x12, 0x34, 0x56}

	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sanity-check the request body shape (POST JSON).
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Sender string `json:"sender"`
			Data   string `json:"data"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			t.Errorf("gateway: bad body: %v", err)
		}
		if !strings.EqualFold(in.Sender, to.Hex()) {
			t.Errorf("gateway: sender %q != %q", in.Sender, to.Hex())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"data": "0x" + hex.EncodeToString(gatewayResponse)})
	}))
	defer gw.Close()

	revertHex := encodeOffchainLookupRevert(t, to, []string{gw.URL}, []byte{0xaa}, [4]byte{0x01, 0x02, 0x03, 0x04}, []byte{0xbb})

	stub := &stubBackend{responses: []callResp{
		{data: nil, err: &ofcError{hexData: revertHex}},
		{data: finalResult, err: nil},
	}}
	c := New(stub)
	c.allowInsecure = true

	got, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to, Data: []byte{0xff}}, nil)
	if err != nil {
		t.Fatalf("CallContract: %v", err)
	}
	if hex.EncodeToString(got) != hex.EncodeToString(finalResult) {
		t.Fatalf("unexpected final result: %x", got)
	}
	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 inner calls, got %d", len(stub.calls))
	}
	// Second call must use callback selector and re-target the same address.
	second := stub.calls[1]
	if second.To == nil || *second.To != to {
		t.Fatalf("callback retargeted: to=%v want=%v", second.To, to)
	}
	if len(second.Data) < 4 || hex.EncodeToString(second.Data[:4]) != "01020304" {
		t.Fatalf("callback selector wrong: %x", second.Data[:4])
	}
}

func TestCaller_MaxHopsExceeded(t *testing.T) {
	to := common.HexToAddress("0x3333333333333333333333333333333333333333")
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"data": "0x00"})
	}))
	defer gw.Close()

	revertHex := encodeOffchainLookupRevert(t, to, []string{gw.URL}, []byte{0xaa}, [4]byte{0x01, 0x02, 0x03, 0x04}, []byte{0xbb})

	// Script every response to revert; eventually hops run out.
	resps := make([]callResp, 0, 10)
	for i := 0; i < 10; i++ {
		resps = append(resps, callResp{err: &ofcError{hexData: revertHex}})
	}
	stub := &stubBackend{responses: resps}
	c := New(stub)
	c.allowInsecure = true
	c.maxHops = 2

	_, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to}, nil)
	if !errors.Is(err, errMaxHopsExceeded) {
		t.Fatalf("expected errMaxHopsExceeded, got %v", err)
	}
}

func TestCaller_InsecureGatewayRejected(t *testing.T) {
	to := common.HexToAddress("0x4444444444444444444444444444444444444444")
	// Forge a URL that's clearly http:// and don't allow insecure.
	revertHex := encodeOffchainLookupRevert(t, to, []string{"http://insecure.example/{sender}/{data}"}, []byte{0xaa}, [4]byte{0x01, 0x02, 0x03, 0x04}, []byte{0xbb})

	stub := &stubBackend{responses: []callResp{{err: &ofcError{hexData: revertHex}}}}
	c := New(stub) // no allowInsecureURLs()

	_, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to}, nil)
	if !errors.Is(err, errAllGatewaysFailed) {
		t.Fatalf("expected errAllGatewaysFailed, got %v", err)
	}
	// And the inner cause must be errInsecureGateway.
	if !strings.Contains(err.Error(), errInsecureGateway.Error()) {
		t.Fatalf("expected insecure gateway in chain, got %v", err)
	}
}

func TestCaller_SenderMismatch(t *testing.T) {
	to := common.HexToAddress("0x5555555555555555555555555555555555555555")
	other := common.HexToAddress("0x6666666666666666666666666666666666666666")
	revertHex := encodeOffchainLookupRevert(t, other, []string{"https://x.example/{sender}/{data}"}, []byte{0xaa}, [4]byte{0x01, 0x02, 0x03, 0x04}, []byte{0xbb})

	stub := &stubBackend{responses: []callResp{{err: &ofcError{hexData: revertHex}}}}
	c := New(stub)

	_, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to}, nil)
	if !errors.Is(err, errSenderMismatch) {
		t.Fatalf("expected errSenderMismatch, got %v", err)
	}
}

func TestCaller_GetRequestExpansion(t *testing.T) {
	to := common.HexToAddress("0x7777777777777777777777777777777777777777")
	gatewayResponse := []byte{0x99}
	finalResult := []byte{0xee}

	var gotPath string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"data": "0x" + hex.EncodeToString(gatewayResponse)})
	}))
	defer gw.Close()

	// Build a URL with the {sender}/{data} template.
	gatewayURL := gw.URL + "/lookup/{sender}/{data}"
	revertHex := encodeOffchainLookupRevert(t, to, []string{gatewayURL}, []byte{0xaa, 0xbb}, [4]byte{0x01, 0x02, 0x03, 0x04}, []byte{0xcc})

	stub := &stubBackend{responses: []callResp{
		{err: &ofcError{hexData: revertHex}},
		{data: finalResult},
	}}
	c := New(stub)
	c.allowInsecure = true

	_, err := c.CallContract(context.Background(), ethereum.CallMsg{To: &to}, nil)
	if err != nil {
		t.Fatalf("CallContract: %v", err)
	}
	expectedSender := strings.ToLower(to.Hex())
	if !strings.Contains(gotPath, expectedSender) {
		t.Fatalf("URL missing sender substitution: %s (want contains %s)", gotPath, expectedSender)
	}
	if !strings.Contains(gotPath, "0xaabb") {
		t.Fatalf("URL missing data substitution: %s", gotPath)
	}
}
