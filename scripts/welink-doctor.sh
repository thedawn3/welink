#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 not found. Install Python 3 before running scripts/welink-doctor.sh." >&2
  exit 1
fi
python3 "$SCRIPT_DIR/welink_doctor.py" "$@"
