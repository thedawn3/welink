#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

MODE="${WELINK_MODE:-analysis-only}"
PLATFORM="${WELINK_PLATFORM:-auto}"
DATA_DIR="${WELINK_ANALYSIS_DATA_DIR:-${WELINK_DATA_DIR:-}}"
SOURCE_DATA_DIR="${WELINK_SOURCE_DATA_DIR:-}"
WORK_DIR="${WELINK_WORK_DIR:-}"
MSG_DIR="${WELINK_MSG_DIR:-}"
WECHAT_DECRYPT_DIR="${WELINK_WECHAT_DECRYPT_DIR:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="$2"
      shift 2
      ;;
    --platform)
      PLATFORM="$2"
      shift 2
      ;;
    --data-dir|--analysis-data-dir)
      DATA_DIR="$2"
      shift 2
      ;;
    --source-data-dir)
      SOURCE_DATA_DIR="$2"
      shift 2
      ;;
    --work-dir)
      WORK_DIR="$2"
      shift 2
      ;;
    --msg-dir)
      MSG_DIR="$2"
      shift 2
      ;;
    --wechat-decrypt-dir)
      WECHAT_DECRYPT_DIR="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found. Install Docker Desktop first." >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "'docker compose' is unavailable. Enable Docker Compose v2." >&2
  exit 1
fi

args=()
args+=(--mode "${MODE}")
args+=(--platform "${PLATFORM}")
if [ -n "${DATA_DIR}" ]; then
  args+=(--data-dir "${DATA_DIR}")
fi
if [ -n "${SOURCE_DATA_DIR}" ]; then
  args+=(--source-data-dir "${SOURCE_DATA_DIR}")
fi
if [ -n "${WORK_DIR}" ]; then
  args+=(--work-dir "${WORK_DIR}")
fi
if [ -n "${MSG_DIR}" ]; then
  args+=(--msg-dir "${MSG_DIR}")
fi
if [ -n "${WECHAT_DECRYPT_DIR}" ]; then
  args+=(--wechat-decrypt-dir "${WECHAT_DECRYPT_DIR}")
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
mode="$(read_env_value WELINK_MODE "${MODE}")"
resolved_platform="$(read_env_value WELINK_PLATFORM "${PLATFORM}")"
resolved_source_data_dir="$(read_env_value WELINK_SOURCE_DATA_DIR "${SOURCE_DATA_DIR}")"
resolved_work_dir="$(read_env_value WELINK_WORK_DIR "${WORK_DIR}")"
resolved_wechat_decrypt_dir="$(read_env_value WELINK_WECHAT_DECRYPT_DIR "${WECHAT_DECRYPT_DIR}")"
frontend_port="$(read_env_value WELINK_FRONTEND_PORT 3000)"
backend_port="$(read_env_value WELINK_BACKEND_PORT 8080)"
echo "WeLink started."
echo "Local frontend: http://localhost:${frontend_port}"
echo "Local backend : http://localhost:${backend_port}"
if [ -n "${resolved_wechat_decrypt_dir}" ]; then
  echo "wechat-decrypt: ${resolved_wechat_decrypt_dir}"
fi
if [ "${mode}" = "decrypt-first" ]; then
  echo ""
  echo "decrypt-first mode detected."
  echo "Backend is configured to auto-start decrypt on boot."
  echo "Manual override example:"
  echo "curl -X POST \"http://localhost:${backend_port}/api/system/decrypt/start\" \\"
  echo "  -H \"Content-Type: application/json\" \\"
  echo "  -d '{\"platform\":\"${resolved_platform}\",\"source_data_dir\":\"${resolved_source_data_dir}\",\"work_dir\":\"${resolved_work_dir}\",\"auto_refresh\":true,\"wal_enabled\":true}'"
  echo ""
  echo "Check runtime:"
  echo "curl \"http://localhost:${backend_port}/api/system/runtime\""
fi
