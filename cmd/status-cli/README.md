# Status CLI

The Status CLI is a command line interface for interacting with the Status messenging process. It is a tool for developers and QAs to test the communication workflow without running Status desktop and mobile app.

## Features

- Create a new account
- Send and receive contact request
- DM between contacts

## Build

Go to `cmd/status-cli` directory and build the binary

```bash
go build
```

You can also run `make status-cli` in the root directory to build the binary.

## Run

### Run `serve` command:

```bash
# run alice's server
./status-cli serve

# run charlie's server in another terminal with the logged pubkey of Alice
./status-cli serve -n charlie -p 8565 -a <alice-pubkey>
```

You can send direct messages through JSON RPC. If you also want to send messages through terminal enable `interactive` mode (with the `-i` flag)

JSON RPC examples:

```bash
# get waku info
curl -XPOST http://127.0.0.1:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"waku_info","params":[],"id":1}'

# get peer info
curl --request POST --url http://127.0.0.1:8545 --header 'Content-type: application/json' --data '{"jsonrpc": "2.0", "method": "wakuext_peers", "params": [], "id": 1}'

# send contact request from charlie to alice (use -a flag will automatacally send contact request when starting)
curl -XPOST http://127.0.0.1:8565 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendContactRequest","params":[{"id": "0x0436470da23039f10c1588bc6b9fcbd4b815bf9fae4dc09c0fb05a7eaaf1670b5dbdbc757630d54bf2f8be45a796304dc42506c3f4172f499f610a9ed85d9b0d4c", "message": "hello"}],"id":1}'

# send dm from charlie to alice
curl -XPOST http://127.0.0.1:8565 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x0436470da23039f10c1588bc6b9fcbd4b815bf9fae4dc09c0fb05a7eaaf1670b5dbdbc757630d54bf2f8be45a796304dc42506c3f4172f499f610a9ed85d9b0d4c", "message": "how are you"}],"id":1}'

# send dm from alice to charlie
curl -XPOST http://127.0.0.1:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x042c0ce856c41ad6d3f651a84c83f646cdafdf3a26a3d69bce3a6ccf59b23b5a366c12162045d5066abad7912741a6e6c6e8e11e7826c4c850a1de7a2bae24a79c", "message": "Im fine, and you?"}],"id":1}'
```

### Run `serve-account` command

The `./status-cli serve` command will generate a new account, it will print in the console the key UID of that account, if you want to re-run that created account (i.e.: run the account with the same public key), you can do so with this command:

```bash
./status-cli serve-account -n alice -kid 0x02887ff8dddb774ad836c00c8fd30ef9bc45d6b23f1f8cad1bff07d09cb378c3
```

You will need the same name and key

### Run `simulate` command

```bash
# simulate DM between two accounts
./status-cli simulate

# simulate DM in a interactive way
./status-cli simulate -i

# simulate DM with 3 messages
./status-cli simulate -c 3

# run in light mode
./status-cli simulate --light
```

You can run the commands with `--light` to work as a light client.

Logs are recorded in file `*.log` and terminal.

## JSON-RPC use cases

### Start two CLIs and making them contacts

```bash
# terminal 1 (alice)
./status-cli serve -n alice -p5500
# note the public key and the key id from the output

# terminal 2 (bob)
./status-cli serve -n bob -p5501 -a <alice_pub_key>
```

## Restart any existing account

```bash
# notice we need the key id not the pub key here
./status-cli serve -n bob -kid <bob_key_id>
```

### Create community

Have two CLIs running (`alice` and `bob`)

```bash
# 1. (alice) create community
curl --request POST \
  --url http://127.0.0.1:5500/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_createCommunity",
 "params": [
  {
   "membership": 3,
   "name": "cli-test-1",
   "color": "#ffffff",
   "description": "cli-test-1"  
  }
 ],
 "id": 1
}'

# 2. (bob & alice) fetch community
curl --request POST \
  --url http://127.0.0.1:5501/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_fetchCommunity",
 "params": [
  {
   "communityKey": "0x02bea5af5779d5f742f2419cc0d819d3ce33adb922e8e90bdf3533fd121d52d4bc",
   "waitForResponse": true,
   "tryDatabase": true
  }
 ],
 "id": 1
}'

# 3. (bob) request to join community
curl --request POST \
  --url http://127.0.0.1:5501/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_requestToJoinCommunity",
 "params": [
  {
   "communityId": "0x02bea5af5779d5f742f2419cc0d819d3ce33adb922e8e90bdf3533fd121d52d4bc"
  }
 ],
 "id": 1
}'

# 4. (alice) accept request to join community from bob
curl --request POST \
  --url http://127.0.0.1:5500/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_acceptRequestToJoinCommunity",
 "params": [
  {
   "id": "0x1b828fe8c778403268ffcf80b892f8be46cf9a85ba2c9f479bfb0c0a807a71f4"
  }
 ],
 "id": 1
}'

# 5. (alice) send chat message (bob should receive it)
# chatId is the community id concatenated to the chat id
curl --request POST \
  --url http://127.0.0.1:5500/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_sendChatMessage",
 "params": [
  {
   "chatId": "0x02bea5af5779d5f742f2419cc0d819d3ce33adb922e8e90bdf3533fd121d52d4bcdfe601d1-096c-4201-b692-fcdb81ef0cec",
   "text": "hello there",
   "contentType": 1
  }
 ],
 "id": 1
}'

# 6. (bob) leave the community
curl --request POST \
  --url http://127.0.0.1:5501/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_leaveCommunity",
 "params": [
  "0x02bea5af5779d5f742f2419cc0d819d3ce33adb922e8e90bdf3533fd121d52d4bc"
 ],
 "id": 1
}'

# Optional:
# 7. (bob & alice) fetch community again and verify the members (curl from step 2.)
# 8. Instead of creating a community always you can restart alice and bob and proceed from step 2. Alice is the owner

```

### Private group chat

Have two CLIs running (`alice` and `bob`)

```bash
# 1. (alice) create the group chat including bob in it, save the id of the response
curl --request POST \
  --url http://127.0.0.1:8545/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_createGroupChatWithMembers",
 "params": [
  null,
  "group-chat-name",
  [
   "0x04d3c86dfc77b195b705e1831935066076018aa0d7c40044829801ebbfe9b06480ce4662072bf16a3ca7cb8f6289207614deceaf7d33e099dfc9281610375fec08"
  ]
 ],
 "id": 1
}'

# 2. (alice) send the message to the id of the group chat
curl --request POST \
  --url http://127.0.0.1:5500/ \
  --header 'Content-type: application/json' \
  --data '{
 "jsonrpc": "2.0",
 "method": "wakuext_sendGroupChatMessage",
 "params": [
  {
   "id": "8291eae1-338c-4481-9997-04edd2d2bbed-0x0490cbce029eaf094c7f2dcf1feb2d60e91ab1498847eb29fa98cc5ea5a36666b3f9ada142f3080f5074abd942c863438f6af9475f30781790c7e36f9acd2ac93e",
   "message": "hello"
  }
 ],
 "id": 1
}'
```
