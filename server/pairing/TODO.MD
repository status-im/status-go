# Local Pairing To-dos

- [Network Address / Peer Discovery](#network-address--peer-discovery)
- [Network Debug Diagnostics](#network-debug-diagnostics)

## Network Address / Peer Discovery

Revamp the preferred network discovery process

Note this functionality is pre-secret share so doesn't ...

- **Pre-shared secret: scan the QR code first, containing the secret and then attempt to discover the other device(s)**
  - ⚠️ **Problem:** Only works if UDP discovery works. [See possible solution for UDP discovery failure mitigation](#udp-discovery-failure-mitigation)
  - Start pairing server on relevant device, listen to `0.0.0.0`.
    - get selected port number
    - get TLS pub key
    - get secret
  - Scan QR code which including:
    - selected port number
    - TLS pub key
    - secret
  - UDP discovery happens 🧙🪄✨
  - devices acquire ip addresses of peer(s), and themselves.
  - client device attempts connection as normal
- **Post-shared secret, which is confirmed later with the QR code**
  - UDP discovery happens 🧙🪄✨
  - devices acquire ip addresses of peer(s), and themselves.
    - can also optionally give feedback to the user with the pairing options.
  - server device starts server listener, listen to known network ip address
    - get selected port number
    - get TLS pub key
    - get secret
  - Scan QR code which including:
    - selected port number
    - TLS pub key
    - secret
  - client device attempts connection as normal

### Self IP address acquisition

1. Ask network interface for its preferred outbound IP address
    - The process currently relied on
    - Add a required a self "confirmation" ping, to ensure the given address is network addressable.
    - On fail, fall through to stage 2
2. device gets a list of its network addresses.
   - Use `netIfaces, err := net.Interfaces()` [See snippet](#get-all-network-addresses)
   - run filter for only local private net addrs
   - open listener on 0.0.0.0
   - make an outbound network call to all ips
      - call should include the addressing ip so device knows which address is network addressable
   - record all ip addresses that are able to ping the listener.
      - hopefully the list will only ever contain 1 IP address
   - On fail, fall through to stage 3
3. Attempt discovery via UDP multicast
    - **Question :** How do **both** devices know when to start UDP discovery?
      - Perhaps the QR code device can add details to the application
    - no guarantee that both devices get an IP address
      - [See UDP discovery failure mitigation](#udp-discovery-failure-mitigation)

## Network Debug Diagnostics

We need to check a few things if a connection isn't working

- What network ip address does the device think it has?
- What is the port number the listener is listening on?
- Can the device access its own network IP address?
- Can a device find, and be found by, other devices?
  - attempt UDP discovery
- What is the device's network connection(s) status?
  - `ipconfig` / `ifconfig`
- Test each in series, all above functions
  - Capture everything, even errors
    - DO NOT early return errors, log the errors, proceed with the next function.
  - log all outputs to standard logger
  - log all outputs memory for export / return
  - When test suite is complete return log out put

## Refs

###### UDP discovery failure mitigation

 - Both devices listen on a tcp port chosen by the network interface
 - Attempt UDP discovery
 - In the UDP packet include:
   - the device's listener port number
   - ensure the notifier's id is stored for later connections via tcp
 - When peer A discovers peer B, peer A pings peer B's "confirmation" listener with peer B's IP address and network info about peer A
 - peer B attempts an outbound **TLS** connection to its own "confirmation" server, with a random code, to confirm its own IP address.


###### Get all network addresses

```go
package test

import (
   "fmt"
   "net"
)

func AllNetworkAddresses() error {
   netIfaces, err := net.Interfaces()
   if err != nil {
      return err
   }

   for _, i := range netIfaces {
      addrs, err := i.Addrs()
      if err != nil {
         return err
      }

      for _, addr := range addrs {
         var ip net.IP
         switch v := addr.(type) {
         case *net.IPNet:
            ip = v.IP
         case *net.IPAddr:
            ip = v.IP
         default:
			 return fmt.Errorf("unknown ip type %s", v)
         }
         fmt.Print(ip)
      }
   }
   return nil
}
```