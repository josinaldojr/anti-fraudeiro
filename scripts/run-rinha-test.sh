#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:9999}"
RESULTS_PATH="${RESULTS_PATH:-artifacts/rinha/results.json}"
K6_DOCKER_NETWORK="${K6_DOCKER_NETWORK:-}"

mkdir -p "$(dirname "$RESULTS_PATH")"
export K6_NO_USAGE_REPORT=true

if command -v k6 >/dev/null 2>&1; then
  k6 run \
    --env BASE_URL="$BASE_URL" \
    --env RESULTS_PATH="$RESULTS_PATH" \
    scripts/k6/test.js
else
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
    -e RESULTS_PATH="$RESULTS_PATH" \
    -v "$(pwd):/workdir" \
    -w /workdir \
    grafana/k6:latest \
    run \
    --env BASE_URL="$DOCKER_BASE_URL" \
    --env RESULTS_PATH="$RESULTS_PATH" \
    scripts/k6/test.js
fi

if command -v jq >/dev/null 2>&1; then
  jq . "$RESULTS_PATH"
else
  cat "$RESULTS_PATH"
fi
