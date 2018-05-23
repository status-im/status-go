Using custom bootnode
======================

To make custom bootnode usable it must participate in the same DHT as the target cluster.

1. Generate private key for a node

To generate a private key you can use bootnode command line tool from go-ethereum repository.
It is also available as a part of the image: `ethereum/client-go:alltools-latest`.

```bash
docker run --entrypoint=bootnode -v $(pwd):/keys ethereum/client-go:alltools-latest -genkey /keys/node.key
```

2. Run status bootnode with new key and nursery nodes from the target cluster

Status bootnode is available as an image `statusteam/bootnode:latest`.

```bash
./build/bin/bootnode --nodekey=node.key --addr=192.168.0.102:30777 -n=enode://1b843c7697f6fc42a1f606fb3cfaac54e025f06789dc20ad9278be3388967cf21e3a1b1e4be51faecd66c2c3adef12e942b4fcdeb8727657abe60636efb6224f@206.189.6.46:30404
```

3. Construct enode or copy it from bootnode logs.

```
INFO [05-24|07:50:38] UDP listener up                          net=enode://30e0735299e9e516fe9295580c122abedd12766188516be4cca1e48ebfb3a8b29da6b377a7abf5cd1685a14bc9baa99ae06e3d9d659e1d1e086deb333f7f4a59@192.168.0.102:30777
```