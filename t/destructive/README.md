Destructive tests
=================

The goal is to test behaviour of status-go and underlying protocols under
erroneous conditions, such as losing network connection.

Test could cause unpredictable side effects, such as change of network configuration.
I don't advice to run them locally on your machine, just use docker container.
Also note that tests are relying on real data, such as number of peers.

```bash
make docker-test ARGS="./t/destructive/ -v -network=3"
```