Discovering nodes with DNS Discovery
===
DNS Discovery enables anyone to register an ENR tree in the TXT field of a domain name.

ENR is the format used to store node connection details (ip, port, multiaddr, etc).

This enables a separation of software development and operations as dApp developers can include one or several domain names to use for DNS discovery, while operators can handle the update of the dns record.

It also enables more decentralized bootstrapping as anyone can register a domain name and publish it for others to use.

**Pros**:
- Low latency, low resource requirements,
- Bootstrap list can be updated by editing a domain name: no code change is needed,
- Can reference to a greater list of nodes by pointing to other domain names in the code or in the ENR tree.

**Cons**:
- Prone to censorship: domain names can be blocked,
- Limited: Static number of nodes, operators must provide their ENR to the domain owner to get their node listed.


## Retrieving peers
```go
package main

import (
	"context"
	"fmt"

	"github.com/waku-org/go-waku/waku/v2/dnsdisc"
)

func main() {
	discoveryURL := "enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im"
	nodes, err := dnsdisc.RetrieveNodes(context.Background(), discoveryURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(nodes)
}
```

`dnsdisc.RetrieveNodes` can also accept a `WithNameserver(nameserver string)` option which can be used to specify the nameserver to use to retrieve the TXT record from the domain name
