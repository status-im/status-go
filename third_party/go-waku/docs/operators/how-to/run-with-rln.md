# How to run spam prevention on your go-waku node (RLN)

This guide explains how to run a go-waku node with RLN (Rate Limiting Nullifier) enabled.

[RLN](https://rfc.vac.dev/spec/32/) is a protocol integrated into waku v2, 
which prevents spam-based attacks on the network.

For further background on the research for RLN tailored to waku, refer
to [this](https://rfc.vac.dev/spec/17/) RFC.

Registering to the membership group has been left out for brevity.
If you would like to register to the membership group and send messages with RLN,
refer to the [on-chain chat2 tutorial](../../tutorial/onchain-rln-relay-chat2.md).

This guide specifically allows a node to participate in RLN testnet
You may alter the rln-specific arguments as required.


## 1. Update the runtime arguments

Follow the steps from the [build](./build.md) and [run](./run.md) guides while replacing the run command with -

```bash
export WAKU_FLEET=<enrtree of the fleet>
export SEPOLIA_WS_NODE_ADDRESS=<WS RPC URL to a Sepolia Node>
export RLN_RELAY_CONTRACT_ADDRESS="0xF471d71E9b1455bBF4b85d475afb9BB0954A29c4" # Replace this with any compatible implementation
$WAKUNODE_DIR/build/waku \
--dns-discovery \
--dns-discovery-url="$WAKU_FLEET" \
--discv5-discovery=true \
--rln-relay=true \
--rln-relay-dynamic=true \
--rln-relay-eth-contract-address="$RLN_RELAY_CONTRACT_ADDRESS" \
--rln-relay-eth-client-address="$SEPOLIA_WS_NODE_ADDRESS"
```

OR 

If you installed go-waku using a `.dpkg` or `.rpm` package, you can use the `waku` command instead of building go-waku yourself

OR

If you have the go-waku node within docker, you can replace the run command with -

```bash
export WAKU_FLEET=<enrtree of the fleet>
export SEPOLIA_WS_NODE_ADDRESS=<WS RPC URL to a Sepolia Node>
export RLN_RELAY_CONTRACT_ADDRESS="0xF471d71E9b1455bBF4b85d475afb9BB0954A29c4" # Replace this with any compatible implementation
docker run -i -t -p 60000:60000 -p 9000:9000/udp \
  -v /absolute/path/to/your/rlnKeystore.json:/rlnKeystore.json:ro \
  wakuorg/go-waku:latest \
  --dns-discovery=true \
  --dns-discovery-url="$WAKU_FLEET" \
  --discv5-discovery \
  --rln-relay=true \
  --rln-relay-dynamic=true \
  --rln-relay-eth-contract-address="$RLN_RELAY_CONTRACT_ADDRESS" \
  --rln-relay-eth-client-address="$SEPOLIA_WS_NODE_ADDRESS"
```

Following is the list of additional fields that have been added to the
runtime arguments -

1. `--rln-relay`: Allows waku-rln-relay to be mounted into the setup of the go-waku node. All messages sent and received in this node will require to contain a valid proof that will be verified, and nodes that relay messages with invalid proofs will have their peer scoring affected negatively and will be eventually disconnected.
2. `--rln-relay-dynamic`: Enables waku-rln-relay to connect to an ethereum node to fetch the membership group
3. `--rln-relay-eth-contract-address`: The contract address of an RLN membership group
4. `--rln-relay-eth-client-address`: The websocket url to a Sepolia ethereum node

The `--dns-discovery-url` flag should contain a valid URL with nodes encoded according to EIP-1459. You can read more about DNS Discovery [here](https://github.com/waku-org/nwaku/blob/master/docs/tutorial/dns-disc.md)

You should now have go-waku running, with RLN enabled!


> Note: This guide will be updated in the future to include features like slashing.