#!/bin/sh

echo "Stoping docker containers"
docker compose -f docker-compose.anvil.yml -f docker-compose.test.status-go.yml -f docker-compose.status-go.local.yml down
