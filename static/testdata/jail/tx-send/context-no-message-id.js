var _status_catalog = {
    commands: {},
    responses: {}
};

var context = {};
function addContext(ns, key, value) {
    // we ignore ns here (as ns is message id, and in this UC we do not have one)
    context[key] = value;
}

function call(pathStr, paramsStr) {
    var params = JSON.parse(paramsStr),
        path = JSON.parse(pathStr),
        fn, res;

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
        // so context could contain
        // {transaction-sent: true}
        context: context
    };

    return JSON.stringify(res);
}

function sendTransaction(params) {
    var data = {
        from: params.from,
        to: params.to,
        value: web3.toWei(params.value, "ether")
    };

    // Blocking call, it will return when transaction is complete.
    // While call is executing, status-go will call up the application,
    // allowing it to validate and complete transaction
    var hash = web3.eth.sendTransaction(data);

    return {"transaction-hash": hash};
}

_status_catalog.commands['send'] = sendTransaction;
_status_catalog.commands['getBalance'] = function (params) {
    var balance = web3.eth.getBalance(params.address);
    balance = web3.fromWei(balance, "ether")
    if (balance < 90) {
        console.log("Unexpected balance (<90): ", balance)
    }
    // used in tx tests, to check that non-context, non-message-id requests work too,
    // so actual balance is not important
    return {"balance": 42}
};
