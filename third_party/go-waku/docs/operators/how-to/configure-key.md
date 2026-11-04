# Generate and configure a node key

By default a node will generate a new, random key pair each time it boots,
resulting in a different public libp2p `multiaddrs` after each restart.

To maintain consistent addressing across restarts, there are different options:

### Using a previously generated key
It is possible to configure the node with a previously generated private key using the `--nodekey`.

```shell
wakunode2 --nodekey=<64_char_hex>
```

This option takes a [Secp256k1](https://en.bitcoin.it/wiki/Secp256k1) private key in 64 char hexstring format.

To generate such a key on Linux systems,
use the openssl `rand` command to generate a pseudo-random 32 byte hexstring.

```sh
openssl rand -hex 32
```

Example output:

```sh
$ openssl rand -hex 32
6a29e767c96a2a380bb66b9a6ffcd6eb54049e14d796a1d866307b8beb7aee58
```

where the key `6a29e767c96a2a380bb66b9a6ffcd6eb54049e14d796a1d866307b8beb7aee58` can be used as `nodekey`.

To create a reusable keyfile on Linux using `openssl`,
use the `ecparam` command coupled with some standard utilities
whenever you want to extract the 32 byte private key in hex format.

```sh
# Generate keyfile
openssl ecparam -genkey -name secp256k1 -out my_private_key.pem
# Extract 32 byte private key
openssl ec -in my_private_key.pem -outform DER | tail -c +8 | head -c 32| xxd -p -c 32
```

Example output:

```sh
read EC key
writing EC key
0c687bb8a7984c770b566eae08520c67f53d302f24b8d4e5e47cc479a1e1ce23
```

where the key `0c687bb8a7984c770b566eae08520c67f53d302f24b8d4e5e47cc479a1e1ce23` can be used as `nodekey`.

```sh
waku --nodekey=0c687bb8a7984c770b566eae08520c67f53d302f24b8d4e5e47cc479a1e1ce23
```

### generating a keyfile
go-waku can generate an encrypted keyfile containing a random private key:
```sh
waku --generate-key
```
This will create a private key file at path specified in `--key-file` with the password defined by `--key-password`. By default the path is `./nodekey`, and the password for the keyfile is `secret`. This command will not overwrite an existing key file. For that, the `--overwrite` flag can be used.

### using a keyfile
go-waku will attempt to read any file existing at the path specified in the `--key-file` flag. If such file exists, go-waku will attempt to decrypt it using the `--key-password`, and read the private key.

