# C Sharp Example


## Requirements
- git bash  (which is installed as part of [Git](https://git-scm.com/downloads))
- [chocolatey](https://chocolatey.org/install)
- [make](https://community.chocolatey.org/packages/make)
- [mingw](https://community.chocolatey.org/packages/mingw)
- [go](https://go.dev/doc/install)


## Running this example
These instructions should be executed in git bash: 
```bash
# Clone the repository
git clone https://github.com/waku-org/go-waku.git
cd go-waku

# Build the .dll
make dynamic-library

# Copy the library into `libs/` folder
cp ./build/lib/libgowaku.* ./build/examples/waku-csharp/waku-csharp/libs/.
```

Open the solution `waku-csharp.sln` in Visual Studio and run the program.

## Description
The following files are available:
- `Program.cs` contains an example program which uses the waku library
- `Waku.cs`: file containing the `Waku` namespace with classes that allows you to instantiate a Go-Waku node
    - `Waku.Config`: class used to configure the waku node when instantiation
    - `Waku.Node`: waku node. The following methods are available:
        - `Node` - constructor. Initializes a waku node. Receives an optional `Waku.Config`
        - `Start` - mounts all the waku2 protocols
        - `Stop` - stops the waku node
        - `PeerId` - obtain the peer ID of the node.
        - `PeerCnt` - obtain the number of connected peers
        - `ListenAddresses` - obtain the multiaddresses the node is listening to


## Help wanted!
- Is it possible to build go-waku automatically by executing `make dynamic-library` and copying the .dll automatically into `libs/` in Visual Studio?