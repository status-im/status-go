## Ping community

This script publishes a post every 2 seconds on the given community/chat-id, using the seed phrase provided.
### How to build

You must have go installed.
Then you can run, from `cmd/ping-community`

```
go build
```

which should create a `ping-community` executable

### How to run
```
./ping-community --seed-phrase "your seed phrase" --community-id "community-id" --chat-id "chat-id"
```

The parameters are:

`community-id`: the ID of the community
`chat-id`: the ID of the chat we want to post on
