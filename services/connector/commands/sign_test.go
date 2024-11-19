package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
)

func preparePersonalSignRequest(dApp signal.ConnectorDApp, challenge, address string) (RPCRequest, error) {
	return ConstructRPCRequest("personal_sign", []interface{}{challenge, address}, &dApp)
}

func prepareTypedDataV4SignRequest(dApp signal.ConnectorDApp, challenge, address string) (RPCRequest, error) {
	return ConstructRPCRequest("eth_signTypedData_v4", []interface{}{address, challenge}, &dApp)
}

func TestFailToPersonalSignWithMissingDAppFields(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	// Missing DApp fields
	request, err := ConstructRPCRequest("personal_sign", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToPersonalSignForUnpermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	request, err := preparePersonalSignRequest(testDAppData,
		"0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e",
		"0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7",
	)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
	assert.Empty(t, result)
}

func TestFailToPersonalSignWithoutParams(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	request, err := ConstructRPCRequest("personal_sign", nil, &testDAppData)
	assert.NoError(t, err)

	result, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrEmptyRPCParams, err)
	assert.Empty(t, result)
}

func TestFailToPersonalSignWithSignalTimout(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := preparePersonalSignRequest(testDAppData,
		"0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e",
		"0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7",
	)
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestPersonalSignWithSignalAccepted(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	fakedSignature := "0x051"

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	challenge := "0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e"
	address := "0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7"
	request, err := preparePersonalSignRequest(testDAppData, challenge, address)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)
			assert.Equal(t, ev.Challenge, challenge)
			assert.Equal(t, ev.Address, address)

			err = state.handler.SignAccepted(SignAcceptedArgs{
				Signature: fakedSignature,
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, response, fakedSignature)
}

func TestPersonalSignWithSignalRejected(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	challenge := "0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e"
	address := "0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7"
	request, err := preparePersonalSignRequest(testDAppData, challenge, address)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.SignRejected(RejectedArgs{
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrSignRejectedByUser, err)
}

func TestTypedDataV4SignRequestWithSignalAccepted(t *testing.T) {
	state, close := setupCommand(t, Method_SignTypedDataV4)
	t.Cleanup(close)

	fakedSignature := "0x051"

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	challenge := "{\"domain\":{\"chainId\":\"1\",\"name\":\"Ether Mail\",\"verifyingContract\":\"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC\",\"version\":\"1\"},\"message\":{\"contents\":\"Hello, Bob!\",\"from\":{\"name\":\"Cow\",\"wallets\":[\"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826\",\"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF\"]},\"to\":[{\"name\":\"Bob\",\"wallets\":[\"0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB\",\"0xB0BdaBea57B0BDABeA57b0bdABEA57b0BDabEa57\",\"0xB0B0b0b0b0b0B000000000000000000000000000\"]}],\"attachment\":\"0x\"},\"primaryType\":\"Mail\",\"types\":{\"EIP712Domain\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"version\",\"type\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\"}],\"Group\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"members\",\"type\":\"Person[]\"}],\"Mail\":[{\"name\":\"from\",\"type\":\"Person\"},{\"name\":\"to\",\"type\":\"Person[]\"},{\"name\":\"contents\",\"type\":\"string\"},{\"name\":\"attachment\",\"type\":\"bytes\"}],\"Person\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"wallets\",\"type\":\"address[]\"}]}}"
	address := "0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7"
	request, err := prepareTypedDataV4SignRequest(testDAppData, challenge, address)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)
			assert.Equal(t, ev.Challenge, challenge)
			assert.Equal(t, ev.Address, address)

			err = state.handler.SignAccepted(SignAcceptedArgs{
				Signature: fakedSignature,
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, response, fakedSignature)
}

func TestTypedDataV4SignRequestWithSignalRejected(t *testing.T) {
	state, close := setupCommand(t, Method_SignTypedDataV4)
	t.Cleanup(close)

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	challenge := "{\"domain\":{\"chainId\":\"1\",\"name\":\"Ether Mail\",\"verifyingContract\":\"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC\",\"version\":\"1\"},\"message\":{\"contents\":\"Hello, Bob!\",\"from\":{\"name\":\"Cow\",\"wallets\":[\"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826\",\"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF\"]},\"to\":[{\"name\":\"Bob\",\"wallets\":[\"0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB\",\"0xB0BdaBea57B0BDABeA57b0bdABEA57b0BDabEa57\",\"0xB0B0b0b0b0b0B000000000000000000000000000\"]}],\"attachment\":\"0x\"},\"primaryType\":\"Mail\",\"types\":{\"EIP712Domain\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"version\",\"type\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\"}],\"Group\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"members\",\"type\":\"Person[]\"}],\"Mail\":[{\"name\":\"from\",\"type\":\"Person\"},{\"name\":\"to\",\"type\":\"Person[]\"},{\"name\":\"contents\",\"type\":\"string\"},{\"name\":\"attachment\",\"type\":\"bytes\"}],\"Person\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"wallets\",\"type\":\"address[]\"}]}}"
	address := "0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7"
	request, err := preparePersonalSignRequest(testDAppData, challenge, address)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.SignRejected(RejectedArgs{
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrSignRejectedByUser, err)
}

func TestUnsupportedSignMethod(t *testing.T) {
	state, close := setupCommand(t, Method_PersonalSign)
	t.Cleanup(close)

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	challenge := "0x506c65617365207369676e2074686973206d65737361676520746f20636f6e6669726d20796f7572206964656e746974792e"
	address := "0x4B0897b0513FdBeEc7C469D9aF4fA6C0752aBea7"
	request, err := preparePersonalSignRequest(testDAppData, challenge, address)
	assert.NoError(t, err)

	request.Method = "eth_signTypedData"
	fakedSignature := "0x051"

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSign:
			var ev signal.ConnectorSignSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)
			assert.Equal(t, ev.Challenge, challenge)
			assert.Equal(t, ev.Address, address)

			err = state.handler.SignAccepted(SignAcceptedArgs{
				Signature: fakedSignature,
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrInvalidMethod, err)
	assert.Equal(t, response, "")
}
