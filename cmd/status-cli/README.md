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

# run Bob's server in another terminal with the logged pubkey of Alice
./status-cli serve -n bob -p 8565 -a <alice-pubkey>
```

You can send direct messages through terminal or JSON RPC.

JSON RPC examples:

```bash
# get waku info
curl -XPOST http://127.0.0.1:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"waku_info","params":[],"id":1}'

# send contact request from bob to alice (use -a flag will automatacally send contact request when starting)
curl -XPOST http://127.0.0.1:8565 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendContactRequest","params":[{"id": "0x0436470da23039f10c1588bc6b9fcbd4b815bf9fae4dc09c0fb05a7eaaf1670b5dbdbc757630d54bf2f8be45a796304dc42506c3f4172f499f610a9ed85d9b0d4c", "message": "hello"}],"id":1}'

# send dm from bob to alice
curl -XPOST http://127.0.0.1:8565 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x0436470da23039f10c1588bc6b9fcbd4b815bf9fae4dc09c0fb05a7eaaf1670b5dbdbc757630d54bf2f8be45a796304dc42506c3f4172f499f610a9ed85d9b0d4c", "message": "how are you"}],"id":1}'

# send dm from alice to bob
curl -XPOST http://127.0.0.1:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x042c0ce856c41ad6d3f651a84c83f646cdafdf3a26a3d69bce3a6ccf59b23b5a366c12162045d5066abad7912741a6e6c6e8e11e7826c4c850a1de7a2bae24a79c", "message": "Im fine, and you?"}],"id":1}'
```


### Run `simulate` command:

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