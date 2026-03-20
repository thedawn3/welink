#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WECHAT_DECRYPT_DIR="${WELINK_WECHAT_DECRYPT_DIR:-}"
RESTART=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --wechat-decrypt-dir)
      WECHAT_DECRYPT_DIR="$2"
      shift 2
      ;;
    --restart)
      RESTART=1
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

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

WECHAT_DECRYPT_DIR="$(read_env_value WELINK_WECHAT_DECRYPT_DIR "${WECHAT_DECRYPT_DIR}")"

if [ -z "${WECHAT_DECRYPT_DIR}" ]; then
  echo "WELINK_WECHAT_DECRYPT_DIR is empty. Pass --wechat-decrypt-dir or configure it in .env first." >&2
  exit 1
fi

if [ ! -d "${WECHAT_DECRYPT_DIR}" ]; then
  echo "wechat-decrypt dir not found: ${WECHAT_DECRYPT_DIR}" >&2
  exit 1
fi

echo "Open 2-3 full-size images in WeChat first, then continue."

action_ok=0
if [ -x "${WECHAT_DECRYPT_DIR}/find_image_key" ]; then
  (cd "${WECHAT_DECRYPT_DIR}" && sudo ./find_image_key)
  action_ok=1
elif [ -f "${WECHAT_DECRYPT_DIR}/find_image_key.py" ]; then
  (cd "${WECHAT_DECRYPT_DIR}" && python3 find_image_key.py)
  action_ok=1
fi

if [ "${action_ok}" -ne 1 ]; then
  echo "No supported image-key extractor found under ${WECHAT_DECRYPT_DIR}" >&2
  exit 1
fi

if [ -f "${WECHAT_DECRYPT_DIR}/image_keys.json" ]; then
  echo "image_keys.json generated: ${WECHAT_DECRYPT_DIR}/image_keys.json"
elif python3 - <<'PY' "${WECHAT_DECRYPT_DIR}/config.json"
import json, pathlib, sys
p = pathlib.Path(sys.argv[1])
if not p.is_file():
    raise SystemExit(1)
obj = json.loads(p.read_text(encoding='utf-8'))
raise SystemExit(0 if obj.get('image_aes_key') else 1)
PY
then
  echo "config.json now contains image_aes_key"
else
  echo "Extractor finished, but no image_keys.json or config.json.image_aes_key was detected." >&2
  exit 1
fi

if [ "${RESTART}" -eq 1 ]; then
  (cd "${REPO_ROOT}" && docker compose restart backend frontend)
fi

echo "Done. Refresh WeLink system page or restart Docker if it is already running."
