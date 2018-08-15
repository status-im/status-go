Canary service
======================

The mailserver canary service's goal is to provide feedback on whether a specified mailserver is responding
correctly to historic messages request. It sends a request for 1 message in a specified chat room (defaults
to #status) to the mailserver within a specified time window (default is last 24 hours) and succeeds if the
mailserver responds with an acknowledgement to the request message (using the request's hash value as a
match).

## How to run it

```shell
make mailserver-canary

./build/bin/mailserver-canary -log=INFO --mailserver=enode://69f72baa7f1722d111a8c9c68c39a31430e9d567695f6108f31ccb6cd8f0adff4991e7fdca8fa770e75bc8a511a87d24690cbc80e008175f40c157d6f6788d48@206.189.240.16:30504
```