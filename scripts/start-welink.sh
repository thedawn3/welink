#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DATA_DIR="${WELINK_DATA_DIR:-}"
MSG_DIR="${WELINK_MSG_DIR:-}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found. Install Docker Desktop first." >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "'docker compose' is unavailable. Enable Docker Compose v2." >&2
  exit 1
fi

args=()
if [ -n "${DATA_DIR}" ]; then
  args+=(--data-dir "${DATA_DIR}")
fi
if [ -n "${MSG_DIR}" ]; then
  args+=(--msg-dir "${MSG_DIR}")
fi

"${SCRIPT_DIR}/welink-doctor.sh" "${args[@]}" --write-env

cd "${REPO_ROOT}"
docker compose up -d --build
echo "WeLink started."
echo "Frontend: http://localhost:3000"
echo "Backend : http://localhost:8080"
