Peer pool signals
=================

Peer pool sends 3 types of signals.

Discovery started signal will be sent once discovery server is started.
And every time node will have to re-start discovery server because peer number dropped too low.

```json
{
  "type": "discovery.started",
  "event": null
}
```


Discovery stopped signal will be sent once discovery found max limit of peers
for every registered topic.

```json
{
  "type": "discovery.stopped",
  "event": null
}
```


Discovery summary signal will be sent every time new peer is added or removed
from a cluster. It will contain a map with capability as a key and total numbers
of peers with that capability as a value.

```json
{
  "type": "discovery.summary",
  "event": {
    "shh/6": 1
  }
}
```

Or if we don't have any peers:

```json
{
  "type": "discovery.summary",
  "event": {}
}
```
