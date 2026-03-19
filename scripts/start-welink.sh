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

read_env_value() {
  local key="$1"
  local fallback="$2"
  local env_file="${REPO_ROOT}/.env"
  if [ -f "${env_file}" ]; then
    local line
    line="$(grep -E "^${key}=" "${env_file}" | tail -n 1 || true)"
    if [ -n "${line}" ]; then
      echo "${line#*=}"
      return
    fi
  fi
  echo "${fallback}"
}

"${SCRIPT_DIR}/welink-doctor.sh" "${args[@]}" --write-env

cd "${REPO_ROOT}"
docker compose up -d --build
frontend_port="$(read_env_value WELINK_FRONTEND_PORT 3000)"
backend_port="$(read_env_value WELINK_BACKEND_PORT 8080)"
echo "WeLink started."
echo "Local frontend: http://localhost:${frontend_port}"
echo "Local backend : http://localhost:${backend_port}"
