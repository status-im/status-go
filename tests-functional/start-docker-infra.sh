#!/bin/sh

echo "Starting docker containers"
docker compose -f docker-compose.anvil.yml -f docker-compose.test.status-go.yml -f docker-compose.status-go.local.yml up --build --remove-orphans -d

echo "Deploying smart contracts"
python -m clients.contract_deployers.deployer