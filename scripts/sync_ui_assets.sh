#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="${ROOT_DIR}/ui"
TARGET_DIR="${ROOT_DIR}/internal/templates/assets"

FILES=(
  base.css
  layout.css
  components.css
  utilities.css
  patterns.css
  tool.js
)

if [[ "${1:-}" == "--check" ]]; then
  for file in "${FILES[@]}"; do
    if ! cmp -s "${SOURCE_DIR}/${file}" "${TARGET_DIR}/${file}"; then
      echo "out of sync: ${file}" >&2
      echo "run scripts/sync_ui_assets.sh" >&2
      exit 1
    fi
  done
  echo "ui assets are in sync"
  exit 0
fi

for file in "${FILES[@]}"; do
  cp "${SOURCE_DIR}/${file}" "${TARGET_DIR}/${file}"
  echo "synced ${file}"
done
