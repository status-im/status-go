Canary service
======================

The P2P node canary service's goal is to provide feedback on whether a specified node is responding
correctly. It can:

- test whether a static peer is responding correctly;
- test whether a mailserver responds to historic messages request. It sends a request for 1 message in a specified chat room (defaults to #status) to the mailserver within a specified time window (default is last 24 hours) and succeeds if the mailserver responds with an acknowledgement to the request message (using the request's hash value as a match).

## How to run it

```shell
make node-canary

./build/bin/node-canary -log=INFO --mailserver=enode://69f72baa7f1722d111a8c9c68c39a31430e9d567695f6108f31ccb6cd8f0adff4991e7fdca8fa770e75bc8a511a87d24690cbc80e008175f40c157d6f6788d48@206.189.240.16:30504

./build/bin/node-canary -log=INFO --staticnode=enode://9c2b82304d988cd78bf290a09b6f81c6ae89e71f9c0f69c41d21bd5cabbd1019522d5d73d7771ea933adf0727de5e847c89e751bd807ba1f7f6fc3a0cd88d997@47.52.91.239:30305
```

It will return with exit code 0 if the enode responded correctly, and a positive number otherwise.
