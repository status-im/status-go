### How to build

You must have go installed.
Then you can run, from `cmd/spiff-workflow`

```
go build --mod=vendor
```

which should create a `spiff-workflow` executable

### How to run
```
./spiff-workflow --seed-phrase "your seed phrase"
```


The parameters are:

`seed-phrase`: the seed phrase of the account to be created

The db will be created in the `./tmp` directory, and it will erase any previous data
The server will be listening on `localhost:8545` and it will respond to RPC calls.


### Sending a message

```
curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x04e431a0baaac2602052f259d4304371d0e0d86cb024497899cf3e82211ff17a9723d8ca67b6575a700086b2aa6ab0df4dab1f8e94114912f269fc6b1ee6764a58", "message": "hello"}],"id":1}'
```

Just replace `id` with the public key you want to use, and `message` with the text you want to send.
