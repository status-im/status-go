// jail.Send() expects to find the current message id in `status.message_id`
// (if not found message id will not be injected, and operation will proceed)
var status = {
    message_id: '42' // global message id, gets replaced in sendTransaction (or any other method)
};

var _status_catalog = {
    commands: {},
    responses: {}
};

var context = {};
function addContext(ns, key, value) { // this function is expected to be present, as status-go uses it to set context
    if (!(ns in context)) {
        context[ns] = {}
    }
    context[ns][key] = value;
}

function call(pathStr, paramsStr) {
    var params = JSON.parse(paramsStr),
        path = JSON.parse(pathStr),
        fn, res;

    // Since we allow next request to proceed *immediately* after jail obtains message id
    // we should be careful overwritting global context variable.
    // We probably should limit/scope to context[message_id] = {}
    context = {};

    fn = path.reduce(function(catalog, name) {
        if (catalog && catalog[name]) {
            return catalog[name];
        }
    }, _status_catalog);

    if (!fn) {
        return null;
    }

    // while fn wll be executed context will be populated
    // by addContext calls from status-go
    callResult = fn(params);
    res = {
        result: callResult,
        // So, context could contain {eth_transactionSend: true}
        // additionally, context gets `message_id` as well.
        // You can scope returned context by returning context[message_id],
        // however since serialization guard will be released immediately after message id
        // is obtained, you need to be careful if you use global message id (it
        // works below, in test, it will not work as expected in highly concurrent environment)
        context: context[status.message_id]
    };

    return JSON.stringify(res);
}

function sendAsyncTransaction(params) {
    var data = {
        from: params.from,
        to: params.to,
        value: web3.toWei(params.value, "ether")
    };

    // message_id allows you to distinguish between !send invocations
    // (when you receive transaction queued event, message_id will be
    // attached along the queued transaction id)
    status.message_id = 'foobar';

    // Blocking call, it will return when transaction is complete.
    // While call is executing, status-go will call up the application,
    // allowing it to validate and complete transaction
    var hash

    web3.eth.sendTransaction(data, function(h){
      hash = h
    });

    return {"transaction-hash": hash};
}

function sendTransaction(params) {
    var data = {
        from: params.from,
        to: params.to,
        value: web3.toWei(params.value, "ether")
    };

    // message_id allows you to distinguish between !send invocations
    // (when you receive transaction queued event, message_id will be
    // attached along the queued transaction id)
    status.message_id = 'foobar';

    // Blocking call, it will return when transaction is complete.
    // While call is executing, status-go will call up the application,
    // allowing it to validate and complete transaction
    var hash = web3.eth.sendTransaction(data);

    return {"transaction-hash": hash};
}

_status_catalog.commands['send'] = sendTransaction;
_status_catalog.commands['sendAsync'] = sendAsyncTransaction;
_status_catalog.commands['getBalance'] = function (params) {
    var balance = web3.eth.getBalance(params.address);
    balance = web3.fromWei(balance, "ether");
    if (balance < 90) {
        console.log("Unexpected balance (<90): ", balance)
    }
    // used in tx tests, to check that non-context, non-message-is requests work too
    // so actual balance is not important
    return {"balance": 42}
};
