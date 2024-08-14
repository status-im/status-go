# List of running node examples

> All code snippets are run from the root project directory.

## Run Waku node

Running Waku node is a matter of a correct configuration. To enable Waku and JSON-RPC HTTP interface use:
```shell script
{
  "APIModules": "waku",
  "HTTPEnabled": true,
  "HTTPHost": "localhost",
  "HTTPPort": 8545,
  "WakuConfig": {
    "Enabled": true
  }
}
```

This command will start a Waku node using the `status.prod` fleet:
```shell script
$ ./build/bin/statusd -c ./_examples/waku.json
```

From now on, you can interact with Waku using HTTP interface:
```shell script
$ curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"waku_info","params":[],"id":1}'
```

## Whisper-Waku bridge

This example demonstrates how bridging between Whisper and Waku works.

First, start a Whisper node and listen to messages:
```shell script
# start node
$ ./build/bin/statusd -c ./_examples/whisper.json -fleet eth.test -dir ./test-bridge-whisper -addr=:30313

# create a symmetric key
$ echo '{"jsonrpc":"2.0","method":"shh_generateSymKeyFromPassword","params":["test-channel"],"id":1}' | \
    nc -U ./test-bridge-whisper/geth.ipc
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "7d521b2501ec6ed99787ecdec98390f141e6d823c703ad88b73a09d81b07e35e"
}

# create a message filter
$ echo '{"jsonrpc":"2.0","method":"shh_newMessageFilter","params":[{"topics": ["0xaabbccdd"], "symKeyID":"7d521b2501ec6ed99787ecdec98390f141e6d823c703ad88b73a09d81b07e35e"}],"id":1}' | \
    nc -U ./test-bridge-whisper/geth.ipc
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "8fd6c01721a90c6650223f8180afe10c66ea5ab30669797d8b42d09f65a819a6"
}
```

In another terminal, start a Waku node and send messages:
```shell script
$ ./build/bin/statusd -c ./_examples/waku.json -fleet eth.test -dir ./test-bridge-waku -addr=:30303

# create a symmetric key
$ echo '{"jsonrpc":"2.0","method":"waku_generateSymKeyFromPassword","params":["test-channel"],"id":1}' | \
    nc -U ./test-waku-bridge/geth.ipc
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "1e07adfcb80c9e9853fb2c4cce3d91c17edd17ab6e950387833d64878fe91624"
}

# send a message
$ echo '{"jsonrpc":"2.0","method":"waku_post","params":[{"symKeyID":"1e07adfcb80c9e9853fb2c4cce3d91c17edd17ab6e950387833d64878fe91624", "ttl":100, "topic": "0xaabbccdd", "payload":"0x010203", "powTarget": 5.0, "powTime": 3}],"id":1}' | \
    nc -U ./test-waku-bridge/geth.ipc
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x1832693cdb951b2cf459c9a6e98755407851af401ee1e7859a919ae95f79ef7a"
}
```

Finally, check messages in Whisper node:
```shell script
$ echo '{"jsonrpc":"2.0","method":"shh_getFilterMessages","params":["8fd6c01721a90c6650223f8180afe10c66ea5ab30669797d8b42d09f65a819a6"],"id":1}' | \
    nc -U ./test-whisper-bridge/geth.ipc | jq .
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    {
      "ttl": 100,
      "timestamp": 1582652341,
      "topic": "0xaabbccdd",
      "payload": "0xd31d35d36d37",
      "padding": "0x925591dc6f6b7d01bd687700a222004e133141b7bb8048ac45d90232d8025f02292aa83befe91fe8ec46a47e7bcfb09d8f2d3529afe4e1835315351248b6735a190c9915b021e54de1975ac9d801aff9dec7bfee4cbe9245c3caca70694fa95718e17f8a5b8385bfc3e7196328cdb4fe722e49368c308c35fe73573c639a54b944bc2e35b080b9d36e7d298340bed253be3a26ac609e19df25de90fd9ab4237423772077046805f8dc3d5ad028cc602fd687e98cbb2c4226cba54b7c3e28f6d22bee510db445fe64bfcc996ddcc40423e1fc9e7fd39e2c0b838ded69c451022fe9202b386d9bd17d47d33942c60172f22ab0d38675b0d92c",
      "pow": 17.418205980066446,
      "hash": "0x1832693cdb951b2cf459c9a6e98755407851af401ee1e7859a919ae95f79ef7a"
    }
  ]
}
```

