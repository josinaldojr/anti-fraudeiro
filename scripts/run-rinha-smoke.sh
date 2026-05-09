#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:9999}"
K6_DOCKER_NETWORK="${K6_DOCKER_NETWORK:-}"
export K6_NO_USAGE_REPORT=true

if command -v k6 >/dev/null 2>&1; then
  k6 run --env BASE_URL="$BASE_URL" scripts/k6/smoke.js
  exit 0
fi

DOCKER_BASE_URL="$BASE_URL"
if [ "$DOCKER_BASE_URL" = "http://localhost:9999" ]; then
  DOCKER_BASE_URL="http://host.docker.internal:9999"
fi

DOCKER_NETWORK_ARGS=()
if [ -n "$K6_DOCKER_NETWORK" ]; then
  DOCKER_NETWORK_ARGS+=(--network "$K6_DOCKER_NETWORK")
fi

docker run --rm \
  "${DOCKER_NETWORK_ARGS[@]}" \
  -e K6_NO_USAGE_REPORT=true \
  -e BASE_URL="$DOCKER_BASE_URL" \
  -v "$(pwd):/workdir" \
  -w /workdir \
  grafana/k6:latest \
  run --env BASE_URL="$DOCKER_BASE_URL" scripts/k6/smoke.js
