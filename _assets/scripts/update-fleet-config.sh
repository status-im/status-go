#!/usr/bin/env bash

DIR="$(cd $(dirname "$0")/../../config/cli; pwd)"

echo "Downloading https://fleets.status.im/"
json=$(curl --silent https://fleets.status.im/)
fleets=(
    'eth.prod'
    'eth.staging'
    'eth.test'
)

for fleet in ${fleets[@]}; do 
    echo "Processing $fleet fleet..."
    fleetJSON=$(echo $json | jq ".fleets.\"$fleet\"")
    boot=$(echo $fleetJSON | jq ".boot | map(.)" -r)
    mail=$(echo $fleetJSON | jq ".mail | map(.)" -r)
    rendezvous=$(echo $fleetJSON | jq ".rendezvous | map(.)" -r)

    # Get random nodes from whisper node list
    maxStaticNodeCount=2
    staticNodeCount=$(echo $fleetJSON | jq ".whisper | length")
    index=$(($RANDOM % ($staticNodeCount - ($maxStaticNodeCount - 1))))
    whisper=$(echo $fleetJSON | jq ".whisper | map(.) | .[$index:($index + $maxStaticNodeCount)]" -r)

    git checkout $DIR/fleet-$fleet.json \
        && jq \
              ".ClusterConfig.BootNodes = $boot \
             | .ClusterConfig.TrustedMailServers = $mail \
             | .ClusterConfig.StaticNodes = $whisper \
             | .ClusterConfig.RendezvousNodes = $rendezvous" \
             $DIR/fleet-$fleet.json \
        | tee "$DIR/tmp.json" >/dev/null \
        && mv $DIR/tmp.json $DIR/fleet-$fleet.json
done
