#!/usr/bin/env sh
set -eu

docker compose -f deployments/docker-compose.yml up --build
