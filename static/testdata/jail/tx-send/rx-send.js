var status = {
    message_id: '42',
};

function sendAsync(params) {
    console.log("Recieving sendAsync: ", params);

    var data = {
        from: params.from,
        to: params.to,
        value: web3.toWei(params.value, "ether")
    };

    var hash

    return { "transaction-hash": hash };
};

var _status_catalog = {
    commands: {
        send: sendAsync,
    },
    responses: {},
};
