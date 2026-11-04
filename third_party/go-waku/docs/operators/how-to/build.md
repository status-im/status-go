# Build go-waku

Go-waku can be built on Linux, macOS and Windows

## Installing dependencies

Cloning and building go-waku requires Go +1.19, a C compiler, Make, Bash and Git.

Go can be installed by following [these instructions](https://go.dev/doc/install)

### Linux

On common Linux distributions the dependencies can be installed with

```sh
# Debian and Ubuntu
sudo apt-get install build-essential git

# Fedora
dnf install @development-tools

# Archlinux, using an AUR manager
yourAURmanager -S base-devel
```

### macOS

Assuming you use [Homebrew](https://brew.sh/) to manage packages

```sh
brew install cmake
```

## Building go-waku

### 1. Clone the go-waku repository

```sh
git clone https://github.com/waku-org/go-waku
cd go-waku
```

### 2. Build waku

```sh
make
```

This will create a `waku` binary in the `./build/` directory.

> **Note:** Building `waku` requires 2GB of RAM.
The build will fail on systems not fulfilling this requirement. 

> Setting up `waku` on the smallest [digital ocean](https://docs.digitalocean.com/products/droplets/how-to/) droplet, you can either
> * compile on a stronger droplet featuring the same CPU architecture and downgrade after compiling, or
> * activate swap on the smallest droplet, or
> * use Docker.
