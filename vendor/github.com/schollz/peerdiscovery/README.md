# peerdiscovery

<img src="https://img.shields.io/badge/coverage-89%25-brightgreen.svg?style=flat-square" alt="Code coverage">&nbsp;<a href="https://goreportcard.com/report/github.com/schollz/peerdiscovery"><img src="https://goreportcard.com/badge/github.com/schollz/peerdiscovery?style=flat-square" alt="Go Report"></a>&nbsp;<a href="https://godoc.org/github.com/schollz/peerdiscovery"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square" alt="Go Doc"></a> 

Pure-go library for cross-platform thread-safe local peer discovery using UDP multicast. I needed to use peer discovery for [croc](https://github.com/schollz/croc) and everything I tried had problems, so I made another one.


## Install

Make sure you have Go 1.5+.

```
go get -u github.com/schollz/peerdiscovery
```

## Usage 

The following is a code to find the first peer on the local network and print it out.

```golang
discoveries, _ := peerdiscovery.Discover(peerdiscovery.Settings{Limit: 1})
for _, d := range discoveries {
    fmt.Printf("discovered '%s'\n", d.Address)
}
```

Here's the output when running on two computers. (*Run these gifs in sync by hitting Ctl + F5*).

**Computer 1:**

![computer 1](https://user-images.githubusercontent.com/6550035/39165714-ba7167d8-473a-11e8-82b5-fb7401ce2138.gif)

**Computer 2:**

![computer 1](https://user-images.githubusercontent.com/6550035/39165716-ba8db9ec-473a-11e8-96f7-e8c64faac676.gif)

For more examples, see the scanning examples ([ipv4](https://github.com/schollz/peerdiscovery/blob/master/examples/ipv4/main.go) and [ipv6](https://github.com/schollz/peerdiscovery/blob/master/examples/ipv6/main.go)) or [the docs](https://pkg.go.dev/github.com/schollz/peerdiscovery).


## Testing

To test the peer discovery with just one host, one can launch multiple containers. The provided `Dockerfile` will run the example code.
Please make sure to enable [Docker's IPv6 support](https://docs.docker.com/v17.09/engine/userguide/networking/default_network/ipv6/) if you are using IPv6 for peer discovery.

```console
# Build the container, named peertest
$ docker build -t peertest .

# Execute the following command in multiple terminals
$ docker run -t --rm peertest
Scanning for 10 seconds to find LAN peers
 100% |████████████████████████████████████████|  [9s:0s]Found 1 other computers
0) '172.17.0.2' with payload 'zqrecHipCO'
```


## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## Thanks

Thanks [@geistesk](https://github.com/geistesk) for adding IPv6 support and a `Notify` func, and helping maintain! Thanks [@Kunde21](https://github.com/Kunde21) for providing a bug fix and massively refactoring the code in a much better way. Thanks [@robpre](https://github.com/robpre) for finding and fixing bugs. Thanks [@shvydky](https://github.com/shvydky) for adding dynamic payloads.

## License

MIT
