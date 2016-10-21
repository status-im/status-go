
var _status_catalog = {
    commands: {},
    responses: {}
};

function call(pathStr, paramsStr) {
    var params = JSON.parse(paramsStr),
        path = JSON.parse(pathStr),
        fn, res;

    fn = path.reduce(function(catalog, name) {
        if (catalog && catalog[name]) {
            return catalog[name];
        }
    }, _status_catalog);

    if (!fn) {
        return null;
    }

    res = fn(params);
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
    // used in tx tests, to check that non-context, non-message-is requests work too
    // so actual balance is not important
    return {"balance": 42}
};