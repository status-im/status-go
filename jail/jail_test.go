package jail_test

import (
	"reflect"
	"testing"

	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/jail"
)

const (
	TEST_ADDRESS         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	CHAT_ID_INIT         = "CHAT_ID_INIT_TEST"
	CHAT_ID_CALL         = "CHAT_ID_CALL_TEST"
	CHAT_ID_NON_EXISTENT = "CHAT_IDNON_EXISTENT"

	TESTDATA_STATUS_JS = "testdata/status.js"
)

func TestJailUnInited(t *testing.T) {
	errorWrapper := func(err error) string {
		return `{"error":"` + err.Error() + `"}`
	}

	expectedError := errorWrapper(jail.ErrInvalidJail)

	var jailInstance *jail.Jail
	response := jailInstance.Parse(CHAT_ID_CALL, ``)
	if response != expectedError {
		t.Errorf("error expected, but got: %v", response)
	}

	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	if response != expectedError {
		t.Errorf("error expected, but got: %v", response)
	}

	_, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != jail.ErrInvalidJail {
		t.Errorf("error expected, but got: %v", err)
	}

	_, err = jailInstance.ClientRestartWrapper()
	if err != jail.ErrInvalidJail {
		t.Errorf("error expected, but got: %v", err)
	}

	// now make sure that if Init is called, then Parse doesn't produce any error
	jailInstance = jail.Init(``)
	if jailInstance == nil {
		t.Error("jail instance shouldn't be nil at this point")
		return
	}
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	response = jailInstance.Parse(CHAT_ID_CALL, statusJS)
	expectedResponse := `{"result": {"commands":{},"responses":{}}}`
	if response != expectedResponse {
		t.Errorf("unexpected response received: %v", response)
	}

	// however, we still expect issue voiced if somebody tries to execute code with Call
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	if response != errorWrapper(geth.ErrInvalidGethNode) {
		t.Errorf("error expected, but got: %v", response)
	}

	// make sure that Call() succeeds when node is started
	err = geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse = `{"result": 144}`
	if response != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestJailInit(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	initCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`
	jailInstance := jail.Init(initCode)

	extraCode := `
	var extraFunc = function (x) {
	  return x * x;
	};
	`
	response := jailInstance.Parse(CHAT_ID_INIT, extraCode)

	expectedResponse := `{"result": {"foo":"bar"}}`

	if !reflect.DeepEqual(expectedResponse, response) {
		t.Error("Expected output not returned from jail.Parse()")
		return
	}
}

func TestJailFunctionCall(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	// load Status JS and add test command to it
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	jailInstance.Parse(CHAT_ID_CALL, statusJS)

	// call with wrong chat id
	response := jailInstance.Call(CHAT_ID_NON_EXISTENT, "", "")
	expectedError := `{"error":"VM[CHAT_IDNON_EXISTENT] doesn't exist."}`
	if response != expectedError {
		t.Errorf("expected error is not returned: expected %s, got %s", expectedError, response)
		return
	}

	// call extraFunc()
	response = jailInstance.Call(CHAT_ID_CALL, `["commands", "testCommand"]`, `{"val": 12}`)
	expectedResponse := `{"result": 144}`
	if response != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, response)
		return
	}
}

func TestJailRPCSend(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	// load Status JS and add test command to it
	statusJS := geth.LoadFromFile(TESTDATA_STATUS_JS)
	jailInstance.Parse(CHAT_ID_CALL, statusJS)

	// obtain VM for a given chat (to send custom JS to jailed version of Send())
	vm, err := jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("cannot get VM: %v", err)
		return
	}

	_, err = vm.Run(`
	    var data = {"jsonrpc":"2.0","method":"eth_getBalance","params":["` + TEST_ADDRESS + `", "latest"],"id":1};
	    var sendResult = web3.currentProvider.send(data)
		console.log(JSON.stringify(sendResult))
		var sendResult = web3.fromWei(sendResult.result, "ether")
	`)
	if err != nil {
		t.Errorf("cannot run custom code on VM: %v", err)
		return
	}

	value, err := vm.Get("sendResult")
	if err != nil {
		t.Errorf("cannot obtain result of balance check operation: %v", err)
		return
	}

	balance, err := value.ToFloat()
	if err != nil {
		t.Errorf("cannot obtain result of balance check operation: %v", err)
		return
	}

	if balance < 90 || balance > 100 {
		t.Error("wrong balance (there should be lots of test Ether on that account)")
		return
	}

	t.Logf("Balance of %.2f ETH found on '%s' account", balance, TEST_ADDRESS)
}

func TestJailMultipleInitSingletonJail(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance1 := jail.Init("")
	jailInstance2 := jail.Init("")
	jailInstance3 := jail.New()
	jailInstance4 := jail.GetInstance()

	if !reflect.DeepEqual(jailInstance1, jailInstance2) {
		t.Error("singleton property of jail instance is violated")
	}
	if !reflect.DeepEqual(jailInstance2, jailInstance3) {
		t.Error("singleton property of jail instance is violated")
	}
	if !reflect.DeepEqual(jailInstance3, jailInstance4) {
		t.Error("singleton property of jail instance is violated")
	}
}

func TestJailGetVM(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	jailInstance := jail.Init("")

	expectedError := `VM[` + CHAT_ID_NON_EXISTENT + `] doesn't exist.`
	_, err = jailInstance.GetVM(CHAT_ID_NON_EXISTENT)
	if err == nil || err.Error() != expectedError {
		t.Error("expected error, but call succeeded")
	}

	// now let's create VM..
	jailInstance.Parse(CHAT_ID_CALL, ``)
	// ..and see if VM becomes available
	_, err = jailInstance.GetVM(CHAT_ID_CALL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
