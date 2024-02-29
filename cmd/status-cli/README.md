# Status CLI

The Status CLI is a command line interface for interacting with the Status messenging process. It is a tool for developers and QAs to test the communication workflow without running Status desktop and mobile app.

## Features

- Create a new account
- Send and receive contact request
- DM between contacts

## Run

Run `dm` command:

```bash
# simulate DM between two accounts
go run main.go dm

# simulate DM in a interactive way
go run main.go dm -i

# simulate DM with 3 messages
go run main.go dm -c 3

# run in light mode
go run main.go dm --light
```

Run `serve` command:

```bash
# run alice's server
go run main.go serve

# run Bob's server in another terminal with the logged pubkey of Alice
go run main.go serve -n bob -a <alice-pubkey>
```

Logs are recorded in file `*.log` and terminal.