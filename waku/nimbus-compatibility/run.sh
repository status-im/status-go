#!/bin/bash

mkdir nimbus
wget https://raw.githubusercontent.com/status-im/nimbus/master/waku/docker/Dockerfile -O ./nimbus/Dockerfile
docker-compose run test --build --force-recreate --rm
exit_code=$?
docker-compose down
exit $exit_code
