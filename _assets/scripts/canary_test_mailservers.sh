#!/usr/bin/env bash

json_path='.ClusterConfig.TrustedMailServers'
mailservers=$(jq -r "${json_path} | .[]" $1)
count=$(jq -r "${json_path} | length" $1)

echo "Will test ${count} mailservers..."
failed_count=0

while read -r mailserver; do
  echo "Testing $mailserver ..."
  ./build/bin/node-canary -log=ERROR -log-without-color=true -mailserver $mailserver || failed_count=$((failed_count + 1))
done <<< "$mailservers"

if [ $failed_count -gt 0 ]; then
  echo "${failed_count}/${count} mailservers failed the test"
  exit 1
else
  echo "All mailservers replied correctly"
fi
