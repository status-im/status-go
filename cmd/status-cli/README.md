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

## Run

Run `serve` command:

```bash
# run alice's server
./status-cli serve

# run Bob's server in another terminal with the logged pubkey of Alice
./status-cli serve -n bob -a <alice-pubkey>
```

Run `dm` command:

```bash
# simulate DM between two accounts
./status-cli dm

# simulate DM in a interactive way
./status-cli dm -i

# simulate DM with 3 messages
./status-cli dm -c 3

# run in light mode
./status-cli dm --light
```

You can run the commands with `--light` to work as a light client.

Logs are recorded in file `*.log` and terminal.