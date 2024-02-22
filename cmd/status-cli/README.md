# Status CLI

The Status CLI is a command line interface for interacting with the Status messenging process. It is a tool for developers and QAs to test the communication workflow without running Status desktop and mobile app.

## Features

- Create a new account
- Send and receive contact request
- DM between contacts

## Run

```bash
go run main.go dm
```

Run in light mode
```bash
go run main.go dm --light
```

Logs are recorded in file `*.log`.