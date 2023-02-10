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

First add the user as a contact:

```
   curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendContactRequest","params":[{"id": "0x04d3c86dfc77b195b705e1831935066076018aa0d7c40044829801ebbfe9b06480ce4662072bf16a3ca7cb8f6289207614deceaf7d33e099dfc9281610375fec08", "message": "hello"}],"id":1}'
```

Accept the contact request in the receiving device (you should see a notification in the activity center)


```
curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendOneToOneMessage","params":[{"id": "0x04e431a0baaac2602052f259d4304371d0e0d86cb024497899cf3e82211ff17a9723d8ca67b6575a700086b2aa6ab0df4dab1f8e94114912f269fc6b1ee6764a58", "message": "hello"}],"id":1}'
```

Just replace `id` with the public key you want to use, and `message` with the text you want to send.


### Creating a private group chat

To create a private group chat, you need interactions on both devices.

First add the user as a contact:

```
   curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendContactRequest","params":[{"id": "0x04d3c86dfc77b195b705e1831935066076018aa0d7c40044829801ebbfe9b06480ce4662072bf16a3ca7cb8f6289207614deceaf7d33e099dfc9281610375fec08", "message": "hello"}],"id":1}'
```

Accept the contact request in the receiving device (you should see a notification in the activity center)

Then create a group chat with the member(s):

```
curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_createGroupChatWithMembers","params":[null, "group-chat-name", ["0x04d3c86dfc77b195b705e1831935066076018aa0d7c40044829801ebbfe9b06480ce4662072bf16a3ca7cb8f6289207614deceaf7d33e099dfc9281610375fec08"]],"id":1}'
```

You will need to note the ID returned by the response, for example, in the response:

```
{"jsonrpc":"2.0","id":1,"result":{"chats":[{"id":"8291eae1-338c-4481-9997-04edd2d2bbed-0x0490cbce029eaf094c7f2dcf1feb2d60e91ab1498847eb29fa98cc5ea5a36666b3f9ada142f3080f5074abd942c863438f6af9475f30781790c7e36f9acd2ac93e","name":"group-chat-name",
```
The ID is:

```
"8291eae1-338c-4481-9997-04edd2d2bbed-0x0490cbce029eaf094c7f2dcf1feb2d60e91ab1498847eb29fa98cc5ea5a36666b3f9ada142f3080f5074abd942c863438f6af9475f30781790c7e36f9acd2ac93e"
```

You can then send messages to this group chat similarly as you send messages for 1-to-1 chats, using the id of the newly created chat:

```
curl -XPOST http://localhost:8545 -H 'Content-type: application/json' -d '{"jsonrpc":"2.0","method":"wakuext_sendGroupChatMessage","params":[{"id": "8291eae1-338c-4481-9997-04edd2d2bbed-0x0490cbce029eaf094c7f2dcf1feb2d60e91ab1498847eb29fa98cc5ea5a36666b3f9ada142f3080f5074abd942c863438f6af9475f30781790c7e36f9acd2ac93e", "message": "hello"}],"id":1}'
```

Mind that if you restart the node, you will need to create a new group chat, since we are currently not keeping storage on restart.
