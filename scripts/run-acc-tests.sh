#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
#
# Run acceptance tests against a real Superset instance via docker-compose.
#
# Usage:
#   ./scripts/run-acc-tests.sh [superset-version] [go-test-args]
#
# Examples:
#   ./scripts/run-acc-tests.sh                          # default: 6.1.0, all tests
#   ./scripts/run-acc-tests.sh 6.0.0                    # test against 6.0
#   ./scripts/run-acc-tests.sh 6.1.0 -run TestAccRole   # filter tests
#   ./scripts/run-acc-tests.sh 6.1.0 -run TestAccRole -v -count=1
#
# Env overrides (export before running):
#   SUPERSET_PORT  – host port to expose Superset on (default: 8088)
#   KEEP_STACK     – set to "1" to leave the compose stack running after tests

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/docker-compose/docker-compose.yml"
COMPOSE_PROJECT="superset-acc-test"

SUPERSET_VERSION="${1:-6.1.0}"
shift 2>/dev/null || true   # consume the version arg; everything left becomes extra go-test flags

# Capture remaining positional args into an array.
# Use ${@+"$@"} instead of "$@" so that 'set -u' does not error when there are
# no extra args: the outer ${...+...} only expands when $@ is non-empty.
EXTRA_ARGS=(${@+"$@"})

SUPERSET_PORT="${SUPERSET_PORT:-8088}"
SUPERSET_HOST="http://localhost:${SUPERSET_PORT}"
KEEP_STACK="${KEEP_STACK:-0}"

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
warning() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# ── Cleanup ───────────────────────────────────────────────────────────────────
cleanup() {
  if [[ "${KEEP_STACK}" == "1" ]]; then
    warning "KEEP_STACK=1 — compose stack left running on port ${SUPERSET_PORT}"
  else
    info "Stopping compose stack..."
    SUPERSET_VERSION="${SUPERSET_VERSION}" SUPERSET_PORT="${SUPERSET_PORT}" \
      docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" down -v --remove-orphans 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ── Start Superset ────────────────────────────────────────────────────────────
info "Starting Superset ${SUPERSET_VERSION} on port ${SUPERSET_PORT}..."
SUPERSET_VERSION="${SUPERSET_VERSION}" SUPERSET_PORT="${SUPERSET_PORT}" \
  docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" pull --quiet
SUPERSET_VERSION="${SUPERSET_VERSION}" SUPERSET_PORT="${SUPERSET_PORT}" \
  docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" up -d --wait

# ── Wait for the API to be ready ─────────────────────────────────────────────
info "Waiting for Superset API at ${SUPERSET_HOST}/health ..."
MAX_WAIT=120
ELAPSED=0
until curl -sf "${SUPERSET_HOST}/health" > /dev/null 2>&1; do
  if (( ELAPSED >= MAX_WAIT )); then
    error "Superset did not become healthy within ${MAX_WAIT}s"
    SUPERSET_VERSION="${SUPERSET_VERSION}" SUPERSET_PORT="${SUPERSET_PORT}" \
      docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" logs --tail=50
    exit 1
  fi
  sleep 3
  (( ELAPSED += 3 ))
done
info "Superset is healthy after ${ELAPSED}s"

# ── Run acceptance tests ──────────────────────────────────────────────────────
info "Running acceptance tests (version=${SUPERSET_VERSION})..."
cd "${REPO_ROOT}"

TF_ACC=1 \
SUPERSET_HOST="${SUPERSET_HOST}" \
SUPERSET_USERNAME="admin" \
SUPERSET_PASSWORD="admin" \
  go test ./internal/provider/ -v -timeout 30m \
  ${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"}
# ↑ The ${array[@]+"${array[@]}"} pattern is required under 'set -u':
#   - the outer ${...+...} only expands when the array is non-empty,
#     so an empty EXTRA_ARGS produces nothing instead of an "unbound variable" error.
#   - when EXTRA_ARGS is non-empty the inner "${EXTRA_ARGS[@]}" expands each
#     element as a separate, properly-quoted word (safe for args with spaces).

info "All tests passed for Superset ${SUPERSET_VERSION}"
